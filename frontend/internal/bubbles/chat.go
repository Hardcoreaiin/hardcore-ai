package bubbles

import (
	"regexp"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// Chat is the conversation. Each user message starts a new turn; assistant
// lines and tool calls from that turn accumulate under it. The view only
// renders the latest turn — older turns are reachable via the viewport.
type Chat struct {
	turns []Turn
	theme *theme.Theme
	width int
}

// Turn is one user prompt and everything that follows from it.
type Turn struct {
	User  string
	Asst  []string   // raw assistant prose lines
	Calls []ToolLine // tool calls executed during this turn
	Ended bool
}

// ToolLine is the compact collapsed view of a tool invocation.
type ToolLine struct {
	Name   string
	Args   []any
	Result string
	Done   bool
}

func NewChat(t *theme.Theme) *Chat {
	return &Chat{theme: t, width: 80}
}

func (c *Chat) Title() string    { return "chat" }
func (c *Chat) HasContent() bool { return len(c.turns) > 0 }

func (c *Chat) SetTheme(t *theme.Theme) { c.theme = t }

// Turns exposes the full history for the viewport.
func (c *Chat) Turns() []Turn { return c.turns }

// junkLine reports whether a line looks like internal tool-call syntax
// that should never reach the chat view.
var junkLine = regexp.MustCompile(`(?i)^\s*(call\b|think:|<\|tool_call)`)

func (c *Chat) Handle(ev agent.Event) bool {
	switch e := ev.(type) {
	case agent.UserMessageEvent:
		c.turns = append(c.turns, Turn{User: e.Text})
		return true
	case agent.LineEvent:
		t := strings.TrimSpace(e.Text)
		if t == "" || junkLine.MatchString(t) {
			return false
		}
		if len(c.turns) == 0 {
			c.turns = append(c.turns, Turn{})
		}
		cur := &c.turns[len(c.turns)-1]
		cur.Asst = append(cur.Asst, e.Text)
		return true
	case agent.ToolStartEvent:
		if len(c.turns) == 0 {
			c.turns = append(c.turns, Turn{})
		}
		cur := &c.turns[len(c.turns)-1]
		cur.Calls = append(cur.Calls, ToolLine{Name: e.Name, Args: e.Args})
		return true
	case agent.ToolResultEvent:
		if len(c.turns) == 0 {
			return false
		}
		cur := &c.turns[len(c.turns)-1]
		for i := len(cur.Calls) - 1; i >= 0; i-- {
			if cur.Calls[i].Name == e.Name && !cur.Calls[i].Done {
				cur.Calls[i].Result = e.Result
				cur.Calls[i].Done = true
				break
			}
		}
		return true
	case agent.DoneEvent:
		if len(c.turns) > 0 {
			c.turns[len(c.turns)-1].Ended = true
		}
		return true
	}
	return false
}

// RenderLatest returns the latest turn rendered as a string of the given width.
// Assistant prose is intentionally omitted — it is displayed in the pet bubble.
// If there are no turns yet, or the turn has no user/tool content, returns empty.
func (c *Chat) RenderLatest(width int) string {
	if len(c.turns) == 0 {
		return ""
	}
	c.width = width
	t := c.turns[len(c.turns)-1]
	if t.User == "" && len(t.Calls) == 0 {
		return ""
	}
	return c.renderTurnNoAsst(t, width)
}

// renderTurnNoAsst renders user message + tool calls only, no assistant prose.
func (c *Chat) renderTurnNoAsst(t Turn, width int) string {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c.theme.Border).
		Padding(0, 1)
	userLabel := lipgloss.NewStyle().Foreground(c.theme.UserFg).Bold(true)
	toolLine := lipgloss.NewStyle().Foreground(c.theme.ToolFg)
	mutedItal := lipgloss.NewStyle().Foreground(c.theme.Muted).Italic(true)

	var b strings.Builder
	if t.User != "" {
		b.WriteString(userLabel.Render("you: ") + t.User)
	}
	for _, call := range t.Calls {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		args := formatArgsList(call.Args)
		head := toolLine.Render("▸ " + call.Name + "(" + args + ")")
		if call.Done {
			res := truncate(strings.ReplaceAll(call.Result, "\n", " "), width-len(call.Name)-len(args)-12)
			head += " " + mutedItal.Render("→ "+res)
		} else {
			head += " " + mutedItal.Render("…")
		}
		b.WriteString(head)
	}
	if b.Len() == 0 {
		return ""
	}
	return border.Width(width - 2).Render(b.String())
}

// RenderHistory returns every turn except the latest, joined with separators.
// Used by the scroll-back viewport.
func (c *Chat) RenderHistory(width int) string {
	if len(c.turns) <= 1 {
		return ""
	}
	c.width = width
	parts := make([]string, 0, len(c.turns)-1)
	for i := 0; i < len(c.turns)-1; i++ {
		parts = append(parts, c.renderTurn(c.turns[i], width))
	}
	return strings.Join(parts, "\n")
}

// RenderAll returns every turn including the latest. Used by the scroll-back
// viewport so the user can page through the full conversation.
func (c *Chat) RenderAll(width int) string {
	if len(c.turns) == 0 {
		return ""
	}
	c.width = width
	parts := make([]string, 0, len(c.turns))
	for i := range c.turns {
		parts = append(parts, c.renderTurn(c.turns[i], width))
	}
	return strings.Join(parts, "\n")
}

func (c *Chat) renderTurn(t Turn, width int) string {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c.theme.Border).
		Padding(0, 1)
	userLabel := lipgloss.NewStyle().Foreground(c.theme.UserFg).Bold(true)
	toolLine := lipgloss.NewStyle().Foreground(c.theme.ToolFg)
	mutedItal := lipgloss.NewStyle().Foreground(c.theme.Muted).Italic(true)

	var b strings.Builder
	if t.User != "" {
		b.WriteString(userLabel.Render("you: ") + t.User)
	}

	// Tool calls render as one collapsed line each, between user and assistant.
	for _, call := range t.Calls {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		args := formatArgsList(call.Args)
		head := toolLine.Render("▸ " + call.Name + "(" + args + ")")
		if call.Done {
			res := truncate(strings.ReplaceAll(call.Result, "\n", " "), width-len(call.Name)-len(args)-12)
			head += " " + mutedItal.Render("→ "+res)
		} else {
			head += " " + mutedItal.Render("…")
		}
		b.WriteString(head)
	}

	if len(t.Asst) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		body := StripLatex(strings.Join(t.Asst, "\n"))
		rendered := c.renderMarkdown(body, width-6)
		b.WriteString(rendered)
	}

	return border.Width(width - 2).Render(b.String())
}

func (c *Chat) renderMarkdown(src string, width int) string {
	style := "dark"
	if c.theme != nil && c.theme.Name == "light" {
		style = "light"
	}
	w := width
	if w < 20 {
		w = 20
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(style),
		glamour.WithWordWrap(w),
	)
	if err != nil || r == nil {
		return src
	}
	out, err := r.Render(src)
	if err != nil {
		return src
	}
	return strings.Trim(out, "\n")
}

func formatArgsList(args []any) string {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = anyToString(a)
	}
	return strings.Join(parts, ", ")
}
