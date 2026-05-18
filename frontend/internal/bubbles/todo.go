package bubbles

import (
	"fmt"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// TodoBubble renders a checklist plan emitted by the model.
// Items start unchecked; they are ticked off as ToolStart/TurnEnd events arrive.
type TodoBubble struct {
	items   []todoItem
	current int // index being worked on (-1 = none)
	theme   *theme.Theme
}

type todoItem struct {
	label string
	done  bool
}

func NewTodoBubble(t *theme.Theme, items []string) *TodoBubble {
	ti := make([]todoItem, len(items))
	for i, s := range items {
		ti[i] = todoItem{label: s}
	}
	return &TodoBubble{items: ti, current: 0, theme: t}
}

func (b *TodoBubble) SetTheme(t *theme.Theme) { b.theme = t }

func (b *TodoBubble) Handle(ev agent.Event) {
	switch ev.(type) {
	case agent.ToolStartEvent:
		// Advance current item when a tool fires.
		if b.current >= 0 && b.current < len(b.items) {
			b.items[b.current].done = true
			if b.current+1 < len(b.items) {
				b.current++
			} else {
				b.current = -1
			}
		}
	case agent.TurnEndEvent:
		// Mark all remaining as done when the turn ends.
		for i := range b.items {
			b.items[i].done = true
		}
		b.current = -1
	}
}

func (b *TodoBubble) View(width int) string {
	t := b.theme
	titleStyle := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	doneStyle := lipgloss.NewStyle().Foreground(t.Muted)
	activeStyle := lipgloss.NewStyle().Foreground(t.UserFg).Bold(true)
	pendingStyle := lipgloss.NewStyle().Foreground(t.Border)

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("plan") + "\n")
	for i, item := range b.items {
		var line string
		switch {
		case item.done:
			line = doneStyle.Render(fmt.Sprintf("  ✓  %s", item.label))
		case i == b.current:
			line = activeStyle.Render(fmt.Sprintf("  ▶  %s", item.label))
		default:
			line = pendingStyle.Render(fmt.Sprintf("  ○  %s", item.label))
		}
		sb.WriteString(line)
		if i < len(b.items)-1 {
			sb.WriteByte('\n')
		}
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderMuted).
		Padding(0, 1).
		Width(width - 2).
		Render(sb.String())
}
