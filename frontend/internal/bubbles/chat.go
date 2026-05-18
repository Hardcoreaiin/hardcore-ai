package bubbles

import (
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// Chat is the user prompt history. Assistant output lives in the pet bubble,
// and tools/code live in masonry bubbles.
type Chat struct {
	turns []Turn
	theme *theme.Theme
	width int
}

// Turn is one user prompt.
type Turn struct {
	User  string
	Ended bool
}

func NewChat(t *theme.Theme) *Chat {
	return &Chat{theme: t, width: 80}
}

func (c *Chat) Title() string    { return "chat" }
func (c *Chat) HasContent() bool { return len(c.turns) > 0 }

func (c *Chat) SetTheme(t *theme.Theme) { c.theme = t }

// Turns exposes the full history for the viewport.
func (c *Chat) Turns() []Turn { return c.turns }

func (c *Chat) Handle(ev agent.Event) bool {
	switch e := ev.(type) {
	case agent.UserMessageEvent:
		c.turns = append(c.turns, Turn{User: e.Text})
		return true
	case agent.DoneEvent:
		if len(c.turns) > 0 {
			c.turns[len(c.turns)-1].Ended = true
		}
		return true
	}
	return false
}

// RenderLatest returns the latest turn rendered as a string of the given width.
// Assistant prose is intentionally omitted — it is displayed in the pet bubble.
// If there are no turns yet, or the turn has no user content, returns empty.
func (c *Chat) RenderLatest(width int) string {
	if len(c.turns) == 0 {
		return ""
	}
	c.width = width
	t := c.turns[len(c.turns)-1]
	if t.User == "" {
		return ""
	}
	return c.renderTurnNoAsst(t, width)
}

// renderTurnNoAsst renders the user message only, no assistant prose or tools.
func (c *Chat) renderTurnNoAsst(t Turn, width int) string {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c.theme.Border).
		Padding(0, 1)
	userLabel := lipgloss.NewStyle().Foreground(c.theme.UserFg).Bold(true)

	var b strings.Builder
	if t.User != "" {
		b.WriteString(userLabel.Render("you: ") + t.User)
	}
	if b.Len() == 0 {
		return ""
	}
	return border.Width(width - 2).Render(b.String())
}

// RenderHistory returns every turn except the latest, joined with separators.
// Used by the scroll-back viewport.
func (c *Chat) RenderHistory(width int) string {
	if len(c.turns) <= 1 {
		return ""
	}
	c.width = width
	parts := make([]string, 0, len(c.turns)-1)
	for i := 0; i < len(c.turns)-1; i++ {
		parts = append(parts, c.renderTurn(c.turns[i], width))
	}
	return strings.Join(parts, "\n")
}

// RenderAll returns every turn including the latest. Used by the scroll-back
// viewport so the user can page through the full conversation.
func (c *Chat) RenderAll(width int) string {
	if len(c.turns) == 0 {
		return ""
	}
	c.width = width
	parts := make([]string, 0, len(c.turns))
	for i := range c.turns {
		parts = append(parts, c.renderTurn(c.turns[i], width))
	}
	return strings.Join(parts, "\n")
}

func (c *Chat) renderTurn(t Turn, width int) string {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c.theme.Border).
		Padding(0, 1)
	userLabel := lipgloss.NewStyle().Foreground(c.theme.UserFg).Bold(true)

	var b strings.Builder
	if t.User != "" {
		b.WriteString(userLabel.Render("you: ") + t.User)
	}

	return border.Width(width - 2).Render(b.String())
}
