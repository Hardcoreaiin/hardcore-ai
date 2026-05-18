package bubbles

import (
	"fmt"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
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
}

func NewToolCall(start agent.ToolStartEvent) *ToolCall {
	return &ToolCall{Name: start.Name, Args: start.Args}
}

func (t *ToolCall) Title() string { return t.Name }

// Handle is unused for ToolCall — the TUI routes result/artifact events
// to the right instance directly. We still satisfy the interface so that
// stray events are safely ignored.
func (t *ToolCall) Handle(ev agent.Event) bool { return false }

func (t *ToolCall) ApplyResult(r agent.ToolResultEvent) {
	t.Result = r.Result
	t.HasResult = true
}

func (t *ToolCall) ApplyArtifact(a agent.ArtifactEvent) {
	t.Artifacts = append(t.Artifacts, a)
}

var (
	callBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("36")).
			Padding(0, 1)
	callBorderPending = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("214")).
				Padding(0, 1)
	callName  = lipgloss.NewStyle().Foreground(lipgloss.Color("36")).Bold(true)
	resultStr = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
	artifact  = lipgloss.NewStyle().Foreground(lipgloss.Color("177"))
)

func (t *ToolCall) View(width int) string {
	var b strings.Builder
	b.WriteString(callName.Render(t.Name+"(") + formatArgs(t.Args) + callName.Render(")"))
	if t.HasResult {
		arrow := " → "
		b.WriteString(arrow + resultStr.Render(truncate(t.Result, width-len(t.Name)-12)))
	} else {
		b.WriteString(" " + lipgloss.NewStyle().Faint(true).Render("…"))
	}
	for _, a := range t.Artifacts {
		b.WriteString("\n" + artifact.Render(fmt.Sprintf("⚑ %s: %v", a.Artifact.Type, a.Artifact.Payload)))
	}
	style := callBorder
	if !t.HasResult {
		style = callBorderPending
	}
	return style.Width(width - 2).Render(b.String())
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
