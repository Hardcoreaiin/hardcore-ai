package tui

import (
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/commands"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// cmdPopup is the floating autocomplete list shown above the input
// whenever the input starts with '/'. The TUI feeds it the current
// input on every keystroke; it returns the suggestion to insert when
// the user presses tab/enter.
type cmdPopup struct {
	reg     *commands.Registry
	items   []commands.Suggestion
	index   int
	visible bool
}

func newCmdPopup(reg *commands.Registry) *cmdPopup {
	return &cmdPopup{reg: reg}
}

// Refresh recomputes suggestions for the current input.
func (p *cmdPopup) Refresh(input string) {
	if !strings.HasPrefix(strings.TrimLeft(input, " "), "/") {
		p.visible = false
		p.items = nil
		p.index = 0
		return
	}
	p.items = p.reg.Suggest(input)
	p.visible = len(p.items) > 0
	if p.index >= len(p.items) {
		p.index = 0
	}
}

func (p *cmdPopup) Visible() bool { return p.visible }

func (p *cmdPopup) Up() {
	if !p.visible {
		return
	}
	p.index = (p.index - 1 + len(p.items)) % len(p.items)
}

func (p *cmdPopup) Down() {
	if !p.visible {
		return
	}
	p.index = (p.index + 1) % len(p.items)
}

// Selected returns the currently highlighted suggestion, if any.
func (p *cmdPopup) Selected() (commands.Suggestion, bool) {
	if !p.visible || len(p.items) == 0 {
		return commands.Suggestion{}, false
	}
	return p.items[p.index], true
}

func (p *cmdPopup) Hide() {
	p.visible = false
}

func (p *cmdPopup) View(t *theme.Theme, width int) string {
	if !p.visible {
		return ""
	}
	if width < 30 {
		width = 30
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(0, 1).
		Width(width - 2)

	rowSel := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	row := lipgloss.NewStyle().Foreground(t.Text)
	detail := lipgloss.NewStyle().Foreground(t.Muted).Italic(true)

	// Cap visible rows so the popup doesn't dominate the screen.
	const maxRows = 7
	start := 0
	end := len(p.items)
	if end > maxRows {
		// Keep the selected row inside the window.
		if p.index >= maxRows {
			start = p.index - maxRows + 1
			end = p.index + 1
		} else {
			end = maxRows
		}
	}

	var lines []string
	for i := start; i < end; i++ {
		it := p.items[i]
		label := it.Label
		det := it.Detail
		// Width budget: label takes left, detail right-aligned.
		labelW := width / 2
		if labelW < 12 {
			labelW = 12
		}
		labelStyled := row.Width(labelW).Render(label)
		if i == p.index {
			labelStyled = rowSel.Width(labelW).Render("▸ " + label)
		} else {
			labelStyled = row.Width(labelW).Render("  " + label)
		}
		lines = append(lines, labelStyled+" "+detail.Render(det))
	}
	hint := detail.Render("↑↓ select · tab/enter insert · esc dismiss")
	return box.Render(strings.Join(lines, "\n") + "\n" + hint)
}
