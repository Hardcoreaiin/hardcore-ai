package bubbles

import (
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// ErrorBubble is a no-op placeholder rendered when an unknown bubble kind is
// requested. It prevents a panic from killing the whole TUI.
type ErrorBubble struct {
	msg   string
	theme *theme.Theme
}

func NewErrorBubble(t *theme.Theme, msg string) *ErrorBubble {
	return &ErrorBubble{theme: t, msg: msg}
}

func (b *ErrorBubble) Title() string           { return "error" }
func (b *ErrorBubble) Handle(_ agent.Event) bool { return false }
func (b *ErrorBubble) SetTheme(t *theme.Theme) { b.theme = t }

func (b *ErrorBubble) View(width int) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(0, 1).
		Width(width - 2).
		Render(lipgloss.NewStyle().Foreground(b.theme.ErrorFg).Render("⚠ " + b.msg))
}
