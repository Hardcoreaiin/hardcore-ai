package tui

import (
	"fmt"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/bubbles"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// tickable is implemented by bubbles that animate via a tick counter.
type tickable interface {
	Tick(t int)
}

// themeable is implemented by bubbles that accept theme changes.
type themeable interface {
	SetTheme(*theme.Theme)
}

// scrollable is implemented by bubbles that accept keyboard scroll events.
type scrollable interface {
	Scroll(msg tea.Msg)
}

// BubbleID is a stable identifier for a masonry slot.
type BubbleID struct {
	Kind string
	Gen  int
}

type masonryEntry struct {
	id        BubbleID
	bubble    bubbles.Bubble
	dismissed bool
	focused   bool
	minimized bool
}

// MasonryManager owns all right-panel bubbles and produces a dynamic masonry layout.
type MasonryManager struct {
	entries []*masonryEntry
	gens    map[string]int
	theme   *theme.Theme
}

func NewMasonryManager(t *theme.Theme) *MasonryManager {
	return &MasonryManager{
		theme: t,
		gens:  make(map[string]int),
	}
}

// Activate ensures a bubble of the given kind exists and is not dismissed.
// If none exists, it creates one. Returns the entry for direct field access.
func (m *MasonryManager) Activate(kind string) *masonryEntry {
	for _, e := range m.entries {
		if e.id.Kind == kind && !e.dismissed {
			return e
		}
	}
	m.gens[kind]++
	e := &masonryEntry{
		id:     BubbleID{Kind: kind, Gen: m.gens[kind]},
		bubble: m.newBubble(kind),
	}
	m.entries = append(m.entries, e)
	return e
}

func (m *MasonryManager) AddToolCall(start agent.ToolStartEvent) {
	m.gens["toolcall"]++
	m.entries = append(m.entries, &masonryEntry{
		id:     BubbleID{Kind: "toolcall", Gen: m.gens["toolcall"]},
		bubble: bubbles.NewToolCall(start, m.theme),
	})
}

func (m *MasonryManager) ApplyToolResult(result agent.ToolResultEvent) {
	for i := len(m.entries) - 1; i >= 0; i-- {
		tc, ok := m.entries[i].bubble.(*bubbles.ToolCall)
		if ok && tc.Name == result.Name && !tc.HasResult {
			tc.ApplyResult(result)
			return
		}
	}
}

func (m *MasonryManager) ApplyArtifact(a agent.ArtifactEvent) {
	for i := len(m.entries) - 1; i >= 0; i-- {
		if tc, ok := m.entries[i].bubble.(*bubbles.ToolCall); ok {
			tc.ApplyArtifact(a)
			return
		}
	}
}

func (m *MasonryManager) newBubble(kind string) bubbles.Bubble {
	switch kind {
	case "code":
		return bubbles.NewCodeBubble(m.theme)
	case "filetree":
		return bubbles.NewFileTreeBubble(m.theme)
	case "buildstatus":
		return bubbles.NewBuildStatusBubble(m.theme)
	case "console":
		return bubbles.NewConsoleBubble(m.theme)
	default:
		// Return a no-op placeholder rather than panicking the whole TUI.
		return bubbles.NewErrorBubble(m.theme, "unknown kind: "+kind)
	}
}

func (m *MasonryManager) active() []*masonryEntry {
	out := make([]*masonryEntry, 0, len(m.entries))
	for _, e := range m.entries {
		if !e.dismissed {
			out = append(out, e)
		}
	}
	return out
}

func (m *MasonryManager) HasContent() bool { return len(m.active()) > 0 }

func (m *MasonryManager) Reset() {
	m.entries = nil
	m.gens = make(map[string]int)
}

func (m *MasonryManager) SetTheme(t *theme.Theme) {
	m.theme = t
	for _, e := range m.entries {
		if th, ok := e.bubble.(themeable); ok {
			th.SetTheme(t)
		}
	}
}

// FocusNext cycles keyboard focus to the next active bubble. After the last
// bubble it cycles to an unfocused state (control returns to the input), then
// back to the first bubble — so tab alone can both enter and leave the panel.
func (m *MasonryManager) FocusNext() {
	live := m.active()
	if len(live) == 0 {
		return
	}
	cur := -1
	for i, e := range live {
		if e.focused {
			cur = i
			break
		}
	}
	for _, e := range live {
		e.focused = false
	}
	// next index in range [0, len]; len means "no focus".
	next := cur + 1
	if next >= len(live) {
		return // unfocused state
	}
	live[next].focused = true
}

// FocusPrev cycles keyboard focus to the previous active bubble, including an
// unfocused state so shift+tab can also leave the panel.
func (m *MasonryManager) FocusPrev() {
	live := m.active()
	if len(live) == 0 {
		return
	}
	cur := -1
	for i, e := range live {
		if e.focused {
			cur = i
			break
		}
	}
	for _, e := range live {
		e.focused = false
	}
	if cur <= 0 {
		if cur == 0 {
			return // step from first bubble to unfocused state
		}
		live[len(live)-1].focused = true // from unfocused, wrap to last
		return
	}
	live[cur-1].focused = true
}

// AnyFocused reports whether any active bubble currently has keyboard focus.
// Used by the TUI to gate m/x/- keys so they don't fire while the user is
// typing a prompt.
func (m *MasonryManager) AnyFocused() bool {
	for _, e := range m.entries {
		if e.focused && !e.dismissed {
			return true
		}
	}
	return false
}

// DismissFocused removes the focused bubble. Returns true if one was dismissed.
// Only acts when a bubble has explicit focus (via tab) — never auto-focuses.
func (m *MasonryManager) DismissFocused() bool {
	for _, e := range m.entries {
		if e.focused && !e.dismissed {
			e.dismissed = true
			e.focused = false
			return true
		}
	}
	return false
}

// ToggleMinimizeFocused toggles minimized state of the focused bubble.
// Only acts when a bubble has explicit focus (via tab) — never auto-focuses.
func (m *MasonryManager) ToggleMinimizeFocused() bool {
	for _, e := range m.entries {
		if e.focused && !e.dismissed {
			e.minimized = !e.minimized
			return true
		}
	}
	return false
}

// UpdateFocused passes a tea.Msg to the focused bubble. Returns true if handled.
func (m *MasonryManager) UpdateFocused(msg tea.Msg) bool {
	for _, e := range m.entries {
		if e.focused && !e.dismissed {
			if s, ok := e.bubble.(scrollable); ok {
				s.Scroll(msg)
				return true
			}
		}
	}
	return false
}

// ClearFocus removes focus from all bubbles.
func (m *MasonryManager) ClearFocus() {
	for _, e := range m.entries {
		e.focused = false
	}
}

// View renders all active bubbles into a masonry grid within rightW columns.
func (m *MasonryManager) View(rightW int, tick int) string {
	// Minimum bubble width large enough for readable code diffs.
	// Capped at 2 columns max so code/diff bubbles are never squeezed below ~60 chars.
	const minBubbleWidth = 62
	const maxCols = 2

	live := m.active()
	if len(live) == 0 {
		return ""
	}

	nCols := rightW / minBubbleWidth
	if nCols < 1 {
		nCols = 1
	}
	if nCols > maxCols {
		nCols = maxCols
	}
	if nCols > len(live) {
		nCols = len(live)
	}
	colW := rightW / nCols

	cols := make([][]string, nCols)
	for i, e := range live {
		rendered := safeView(e, colW, tick, m.theme)
		cols[i%nCols] = append(cols[i%nCols], rendered)
	}

	colStrs := make([]string, nCols)
	for i, col := range cols {
		colStrs[i] = lipgloss.JoinVertical(lipgloss.Left, col...)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, colStrs...)
}

// safeView calls bubble.View() inside a recover so a panicking bubble cannot
// crash the TUI. It also injects the dismiss hint when the entry is focused.
func safeView(e *masonryEntry, width int, tick int, t *theme.Theme) (result string) {
	defer func() {
		if r := recover(); r != nil {
			result = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("196")).
				Padding(0, 1).
				Width(width - 2).
				Render(fmt.Sprintf("[%s] render panic: %v", e.id.Kind, r))
		}
	}()

	// Tick inside the recover region so panicking tickables can't escape.
	if tk, ok := e.bubble.(tickable); ok && !e.minimized {
		tk.Tick(tick)
	}

	var inner string
	if e.minimized {
		title := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render(e.bubble.Title())
		inner = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(t.BorderMuted).
			Padding(0, 1).
			Width(width - 2).
			Render(title + lipgloss.NewStyle().Foreground(t.Muted).Render(" (minimized — press m to expand)"))
	} else {
		inner = e.bubble.View(width)
	}

	// Controls: show actionable keys only when focused (they require tab first).
	// Unfocused bubbles show a dim 'tab to focus' hint instead.
	var ctrls string
	if e.focused {
		ctrlsStr := "m:min  x:close"
		if e.minimized {
			ctrlsStr = "m:expand  x:close"
		}
		ctrls = lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("▸ " + ctrlsStr)
	} else {
		ctrls = lipgloss.NewStyle().Foreground(t.Muted).Italic(true).Render("tab to focus")
	}
	header := lipgloss.NewStyle().Width(width - 2).Align(lipgloss.Right).Render(ctrls)
	
	result = lipgloss.JoinVertical(lipgloss.Top, header, inner)

	if e.focused {
		focusStyle := lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(t.Accent).
			Width(width - 2)
		hint := lipgloss.NewStyle().Foreground(t.Muted).Render("  tab next • x close • m minimize/expand")
		result = focusStyle.Render(strings.TrimRight(result, "\n") + "\n" + hint)
	}

	return result
}

// Typed bubble accessors — callers use these instead of raw type assertions.

func (m *MasonryManager) BuildStatus() *bubbles.BuildStatusBubble {
	return m.Activate("buildstatus").bubble.(*bubbles.BuildStatusBubble)
}

func (m *MasonryManager) Console() *bubbles.ConsoleBubble {
	return m.Activate("console").bubble.(*bubbles.ConsoleBubble)
}

func (m *MasonryManager) FileTree() *bubbles.FileTreeBubble {
	return m.Activate("filetree").bubble.(*bubbles.FileTreeBubble)
}

// Code returns the current (latest) CodeBubble, creating one if absent.
func (m *MasonryManager) Code() *bubbles.CodeBubble {
	return m.Activate("code").bubble.(*bubbles.CodeBubble)
}

// LatestCode returns the most recently added (non-dismissed) code bubble,
// or nil if none exists. Used to update the bubble that was just created
// by NewCodeBubble without creating another one.
func (m *MasonryManager) LatestCode() *bubbles.CodeBubble {
	for i := len(m.entries) - 1; i >= 0; i-- {
		e := m.entries[i]
		if e.id.Kind == "code" && !e.dismissed {
			return e.bubble.(*bubbles.CodeBubble)
		}
	}
	return nil
}

// NewCodeBubble creates a fresh CodeBubble entry in its own masonry slot so
// each file_write is shown as a separate bubble instead of overwriting one.
func (m *MasonryManager) NewCodeBubble() *bubbles.CodeBubble {
	m.gens["code"]++
	bubble := bubbles.NewCodeBubble(m.theme)
	m.entries = append(m.entries, &masonryEntry{
		id:     BubbleID{Kind: "code", Gen: m.gens["code"]},
		bubble: bubble,
	})
	return bubble
}
