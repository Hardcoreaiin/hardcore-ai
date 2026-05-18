package bubbles

import (
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// ToolPanel manages a live list of ToolCall bubbles for the current turn.
// Each ToolStartEvent spawns a new card; the matching ToolResultEvent fills it.
type ToolPanel struct {
	calls []*ToolCall
	theme *theme.Theme
}

func NewToolPanel(t *theme.Theme) *ToolPanel {
	return &ToolPanel{theme: t}
}

func (p *ToolPanel) SetTheme(t *theme.Theme) {
	p.theme = t
	for _, c := range p.calls {
		c.SetTheme(t)
	}
}

func (p *ToolPanel) HasContent() bool { return len(p.calls) > 0 }

// LastCallArgs returns the args of the most recent call with the given name,
// or nil if none found.
func (p *ToolPanel) LastCallArgs(name string) []any {
	for i := len(p.calls) - 1; i >= 0; i-- {
		if p.calls[i].Name == name {
			return p.calls[i].Args
		}
	}
	return nil
}

func (p *ToolPanel) Handle(ev agent.Event) bool {
	switch e := ev.(type) {
	case agent.UserMessageEvent:
		_ = e
		p.calls = nil
		return true
	case agent.ToolStartEvent:
		p.calls = append(p.calls, NewToolCall(e, p.theme))
		return true
	case agent.ToolResultEvent:
		// Match last pending call with this name.
		for i := len(p.calls) - 1; i >= 0; i-- {
			if p.calls[i].Name == e.Name && !p.calls[i].HasResult {
				p.calls[i].ApplyResult(e)
				return true
			}
		}
	case agent.ArtifactEvent:
		// Attach to last call.
		if len(p.calls) > 0 {
			p.calls[len(p.calls)-1].ApplyArtifact(e)
			return true
		}
	}
	return false
}

func (p *ToolPanel) View(width int) string {
	if len(p.calls) == 0 {
		return ""
	}
	t := p.theme
	titleStyle := lipgloss.NewStyle().Foreground(t.ToolFg).Bold(true)

	var parts []string
	parts = append(parts, titleStyle.Render("tool calls"))
	for _, c := range p.calls {
		parts = append(parts, c.View(width))
	}
	return strings.Join(parts, "\n")
}
