package bubbles

import (
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/charmbracelet/lipgloss"
)

type Thought struct {
	thoughts []string
}

func NewThought() *Thought { return &Thought{} }

func (t *Thought) Title() string { return "thought" }

func (t *Thought) Handle(ev agent.Event) bool {
	if e, ok := ev.(agent.ThinkEvent); ok {
		t.thoughts = append(t.thoughts, e.Text)
		return true
	}
	return false
}

var (
	thoughtBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("244")).
		Foreground(lipgloss.Color("244")).
		Italic(true).
		Padding(0, 1)
)

func (t *Thought) View(width int) string {
	if len(t.thoughts) == 0 {
		return thoughtBorder.Width(width - 2).Render("(no thoughts yet)")
	}
	body := strings.Join(t.thoughts, "\n")
	return thoughtBorder.Width(width - 2).Render(body)
}

func (t *Thought) HasContent() bool { return len(t.thoughts) > 0 }
