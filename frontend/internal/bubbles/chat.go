package bubbles

import (
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/charmbracelet/lipgloss"
)

// Chat is the always-on conversation bubble. Each user message starts a
// new turn; assistant lines from that turn accumulate under it.
type Chat struct {
	turns []chatTurn
}

type chatTurn struct {
	user  string
	asst  []string
	ended bool
}

func NewChat() *Chat { return &Chat{} }

func (c *Chat) Title() string { return "chat" }

func (c *Chat) HasContent() bool { return len(c.turns) > 0 }

func (c *Chat) Handle(ev agent.Event) bool {
	switch e := ev.(type) {
	case agent.UserMessageEvent:
		c.turns = append(c.turns, chatTurn{user: e.Text})
		return true
	case agent.LineEvent:
		t := strings.TrimSpace(e.Text)
		if t == "" || strings.HasPrefix(strings.ToUpper(t), "CALL") {
			return false
		}
		if len(c.turns) == 0 {
			c.turns = append(c.turns, chatTurn{})
		}
		cur := &c.turns[len(c.turns)-1]
		cur.asst = append(cur.asst, e.Text)
		return true
	case agent.DoneEvent:
		if len(c.turns) > 0 {
			c.turns[len(c.turns)-1].ended = true
		}
		return true
	}
	return false
}

var (
	chatBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1)
	userLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	asstLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true)
)

func (c *Chat) View(width int) string {
	var b strings.Builder
	for i, t := range c.turns {
		if i > 0 {
			b.WriteString("\n")
		}
		if t.user != "" {
			b.WriteString(userLabel.Render("you: ") + t.user)
		}
		if len(t.asst) > 0 {
			if t.user != "" {
				b.WriteString("\n")
			}
			b.WriteString(asstLabel.Render("ai:  ") + strings.Join(t.asst, "\n     "))
		}
	}
	return chatBorder.Width(width - 2).Render(b.String())
}
