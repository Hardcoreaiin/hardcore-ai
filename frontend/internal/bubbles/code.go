package bubbles

import (
	"fmt"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// CodeBubble shows the most recently written file as a numbered, syntax-tinted
// code block. It updates whenever a stm32_write_file tool result arrives.
type CodeBubble struct {
	path    string
	content string
	theme   *theme.Theme
}

func NewCodeBubble(t *theme.Theme) *CodeBubble {
	return &CodeBubble{theme: t}
}

func (b *CodeBubble) SetTheme(t *theme.Theme) { b.theme = t }
func (b *CodeBubble) HasContent() bool         { return b.content != "" }

// Update sets the displayed file. Called by the TUI when stm32_write_file fires.
func (b *CodeBubble) Update(path, content string) {
	b.path = path
	b.content = content
}

func (b *CodeBubble) Handle(ev agent.Event) bool {
	if _, ok := ev.(agent.UserMessageEvent); ok {
		b.path = ""
		b.content = ""
		return true
	}
	return false
}

func (b *CodeBubble) View(width int) string {
	t := b.theme
	titleStyle := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	lineNumStyle := lipgloss.NewStyle().Foreground(t.Muted)
	codeStyle := lipgloss.NewStyle().Foreground(t.Text)

	innerWidth := width - 6
	if innerWidth < 20 {
		innerWidth = 20
	}

	lines := strings.Split(b.content, "\n")
	var sb strings.Builder
	sb.WriteString(titleStyle.Render("file: "+b.path) + "\n")
	for i, line := range lines {
		num := lineNumStyle.Render(fmt.Sprintf("%3d ", i+1))
		// Truncate long lines to fit the bubble.
		if len(line) > innerWidth {
			line = line[:innerWidth-1] + "…"
		}
		sb.WriteString(num + codeStyle.Render(line))
		if i < len(lines)-1 {
			sb.WriteByte('\n')
		}
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.ToolFg).
		Padding(0, 1).
		Width(width - 2).
		Render(sb.String())
}
