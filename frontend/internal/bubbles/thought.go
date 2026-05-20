package bubbles

import (
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// Thought renders the model's reasoning trace. It's scoped to the latest
// turn: each user message resets the visible thoughts so the bubble
// doesn't grow without bound. Older thoughts are gone — they're not
// useful to scroll back to.
type Thought struct {
	current []string
	theme   *theme.Theme
}

func NewThought(t *theme.Theme) *Thought { return &Thought{theme: t} }

func (t *Thought) Title() string { return "thought" }

func (t *Thought) SetTheme(th *theme.Theme) { t.theme = th }

const maxThoughtLines = 100

func (t *Thought) Handle(ev agent.Event) bool {
	switch e := ev.(type) {
	case agent.UserMessageEvent:
		t.current = nil
		return true
	case agent.ThinkEvent:
		t.current = append(t.current, e.Text)
		// Prevent unbounded growth that would OOM / lag the renderer.
		if len(t.current) > maxThoughtLines {
			t.current = t.current[len(t.current)-maxThoughtLines:]
		}
		return true
	}
	return false
}

func (t *Thought) View(width int) string {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.theme.BorderMuted).
		Foreground(t.theme.Muted).
		Italic(true).
		Padding(0, 1)
	body := strings.Join(t.current, "\n")
	return border.Width(width - 2).Render(body)
}

func (t *Thought) HasContent() bool { return len(t.current) > 0 }
