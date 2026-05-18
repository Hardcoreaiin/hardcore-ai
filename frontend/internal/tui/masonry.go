package tui

import (
	"fmt"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/bubbles"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
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
		panic("unknown masonry kind: " + kind)
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

// FocusNext cycles keyboard focus to the next active bubble.
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
	live[(cur+1)%len(live)].focused = true
}

// FocusPrev cycles keyboard focus to the previous active bubble.
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
	live[(cur-1+len(live))%len(live)].focused = true
}

// DismissFocused removes the focused bubble. Returns true if one was dismissed.
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

// ClearFocus removes focus from all bubbles.
func (m *MasonryManager) ClearFocus() {
	for _, e := range m.entries {
		e.focused = false
	}
}

// View renders all active bubbles into a masonry grid within rightW columns.
func (m *MasonryManager) View(rightW int, tick int) string {
	const minBubbleWidth = 36

	live := m.active()
	if len(live) == 0 {
		return ""
	}

	nCols := rightW / minBubbleWidth
	if nCols < 1 {
		nCols = 1
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
	if tk, ok := e.bubble.(tickable); ok {
		tk.Tick(tick)
	}

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

	result = e.bubble.View(width)

	if e.focused {
		focusStyle := lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(t.Accent).
			Width(width - 2)
		hint := lipgloss.NewStyle().Foreground(t.Muted).Render("  tab next • x dismiss")
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

func (m *MasonryManager) Code() *bubbles.CodeBubble {
	return m.Activate("code").bubble.(*bubbles.CodeBubble)
}
