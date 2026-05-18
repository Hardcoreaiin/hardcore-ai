package bubbles

import (
	"fmt"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// ToolCall is one bubble per tool invocation. The TUI creates a new one
// on each ToolStartEvent and updates it with the matching ToolResultEvent
// + any ArtifactEvents that arrive before the next call.
type ToolCall struct {
	Name      string
	Args      []any
	Result    string
	HasResult bool
	Artifacts []agent.ArtifactEvent
	theme     *theme.Theme
}

func NewToolCall(start agent.ToolStartEvent, t *theme.Theme) *ToolCall {
	return &ToolCall{Name: start.Name, Args: start.Args, theme: t}
}

func (t *ToolCall) Title() string { return t.Name }

func (t *ToolCall) SetTheme(th *theme.Theme) { t.theme = th }

func (t *ToolCall) Handle(ev agent.Event) bool { return false }

func (t *ToolCall) ApplyResult(r agent.ToolResultEvent) {
	t.Result = r.Result
	t.HasResult = true
}

func (t *ToolCall) ApplyArtifact(a agent.ArtifactEvent) {
	t.Artifacts = append(t.Artifacts, a)
}

func (t *ToolCall) View(width int) string {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.theme.ToolFg).
		Padding(0, 1)
	if !t.HasResult {
		border = border.BorderForeground(t.theme.BorderPending)
	}
	callName := lipgloss.NewStyle().Foreground(t.theme.ToolFg).Bold(true)
	resultStr := lipgloss.NewStyle().Foreground(t.theme.Text)
	artifact := lipgloss.NewStyle().Foreground(t.theme.Accent)

	var b strings.Builder
	b.WriteString(callName.Render(t.Name+"(") + formatToolArgs(t.Name, t.Args, width-12) + callName.Render(")"))
	if t.HasResult {
		b.WriteString("\n" + resultStr.Render(truncate(strings.ReplaceAll(t.Result, "\n", " "), width-6)))
	} else {
		b.WriteString(" " + lipgloss.NewStyle().Faint(true).Render("…"))
	}
	for _, a := range t.Artifacts {
		b.WriteString("\n" + artifact.Render(fmt.Sprintf("⚑ %s: %v", a.Artifact.Type, a.Artifact.Payload)))
	}
	return border.Width(width - 2).Render(b.String())
}

func formatToolArgs(name string, args []any, max int) string {
	if max < 16 {
		max = 16
	}
	if name == "file_write" && len(args) >= 2 {
		path, _ := args[0].(string)
		content, _ := args[1].(string)
		return fmt.Sprintf("%q, %d bytes", path, len(content))
	}
	out := formatArgs(args)
	return truncate(out, max)
}

func formatArgs(args []any) string {
	parts := make([]string, len(args))
	for i, a := range args {
		switch v := a.(type) {
		case string:
			parts[i] = fmt.Sprintf("%q", v)
		default:
			parts[i] = fmt.Sprintf("%v", v)
		}
	}
	return strings.Join(parts, ", ")
}

func truncate(s string, n int) string {
	if n < 8 {
		n = 8
	}
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
