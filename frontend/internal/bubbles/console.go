package bubbles

import (
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// ConsoleBubble shows the output of a build, emulate, or flash run.
// It is shown while running (with a spinner) and persists after completion.
type ConsoleBubble struct {
	title   string
	output  string
	running bool
	failed  bool
	tick    int
	theme   *theme.Theme
}

func NewConsoleBubble(t *theme.Theme) *ConsoleBubble {
	return &ConsoleBubble{theme: t, title: "console"}
}

func (b *ConsoleBubble) Title() string          { return "console" }
func (b *ConsoleBubble) SetTheme(t *theme.Theme) { b.theme = t }
func (b *ConsoleBubble) HasContent() bool        { return b.running || b.output != "" }
func (b *ConsoleBubble) Handle(_ agent.Event) bool { return false }

// Tick updates the animation counter — called by the masonry before View.
func (b *ConsoleBubble) Tick(t int) { b.tick = t }

func (b *ConsoleBubble) Start(title string) {
	b.title = title
	b.output = ""
	b.running = true
	b.failed = false
}

func (b *ConsoleBubble) Done(output string, failed bool) {
	b.output = strings.TrimSpace(output)
	b.running = false
	b.failed = failed
}

func (b *ConsoleBubble) View(width int) string {
	t := b.theme
	titleStyle := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(t.Muted)
	okStyle := lipgloss.NewStyle().Foreground(t.Accent)
	failStyle := lipgloss.NewStyle().Foreground(t.ErrorFg).Bold(true)
	outStyle := lipgloss.NewStyle().Foreground(t.Text)

	spinners := []string{"⠋", "⠙", "⠸", "⠼", "⠴", "⠦"}

	var sb strings.Builder

	if b.running {
		spin := spinners[(b.tick/2)%len(spinners)]
		sb.WriteString(titleStyle.Render(b.title) + "  " + lipgloss.NewStyle().Foreground(t.WarnFg).Render(spin+" running…"))
	} else {
		statusStr := okStyle.Render("✓ done")
		if b.failed {
			statusStr = failStyle.Render("✗ failed")
		}
		sb.WriteString(titleStyle.Render(b.title) + "  " + statusStr)
	}

	if b.output != "" {
		innerWidth := width - 6
		if innerWidth < 20 {
			innerWidth = 20
		}
		lines := strings.Split(b.output, "\n")
		maxLines := 20
		start := 0
		if len(lines) > maxLines {
			start = len(lines) - maxLines
		}
		sb.WriteString("\n")
		for i, line := range lines[start:] {
			if len(line) > innerWidth {
				line = line[:innerWidth-1] + "…"
			}
			sb.WriteString(outStyle.Render(line))
			if i < len(lines[start:])-1 {
				sb.WriteByte('\n')
			}
		}
		if len(lines) > maxLines {
			sb.WriteString("\n" + mutedStyle.Render("… (truncated, showing last 20 lines)"))
		}
	}

	borderColor := t.BorderMuted
	if b.running {
		borderColor = t.WarnFg
	} else if b.failed {
		borderColor = t.ErrorFg
	} else if b.output != "" {
		borderColor = t.Accent
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width - 2).
		Render(sb.String())
}
