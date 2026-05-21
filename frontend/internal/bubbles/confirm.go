package bubbles

import (
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmBubble renders a yes/no prompt for a shell command awaiting approval.
// The TUI drives selection with MoveLeft/MoveRight and reads the decision via
// Decided()/Approved() once the user confirms.
type ConfirmBubble struct {
	command  string
	dir      string
	cursor   int // 0 = approve, 1 = reject
	decided  bool
	approved bool
	theme    *theme.Theme
}

func NewConfirmBubble(t *theme.Theme, command, dir string) *ConfirmBubble {
	return &ConfirmBubble{
		command: command,
		dir:     dir,
		cursor:  1, // default to reject — safer
		theme:   t,
	}
}

func (b *ConfirmBubble) SetTheme(t *theme.Theme) { b.theme = t }
func (b *ConfirmBubble) Command() string         { return b.command }
func (b *ConfirmBubble) Decided() bool           { return b.decided }
func (b *ConfirmBubble) Approved() bool          { return b.approved }

func (b *ConfirmBubble) MoveLeft() {
	if !b.decided {
		b.cursor = 0
	}
}

func (b *ConfirmBubble) MoveRight() {
	if !b.decided {
		b.cursor = 1
	}
}

func (b *ConfirmBubble) Toggle() {
	if !b.decided {
		b.cursor = 1 - b.cursor
	}
}

// Confirm locks in the current selection. After this, Decided() is true.
func (b *ConfirmBubble) Confirm() {
	if b.decided {
		return
	}
	b.decided = true
	b.approved = b.cursor == 0
}

// SetApproved forces a decision (used for explicit y/n key shortcuts).
func (b *ConfirmBubble) SetApproved(approved bool) {
	if b.decided {
		return
	}
	b.decided = true
	b.approved = approved
}

func (b *ConfirmBubble) Title() string { return "Run command?" }

func (b *ConfirmBubble) Handle(ev agent.Event) bool { return false }

func (b *ConfirmBubble) View(width int) string {
	t := b.theme
	headStyle := lipgloss.NewStyle().Foreground(t.WarnFg).Bold(true)
	dirStyle := lipgloss.NewStyle().Foreground(t.Muted).Italic(true)
	cmdStyle := lipgloss.NewStyle().Foreground(t.UserFg).Bold(true)
	approveSel := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	rejectSel := lipgloss.NewStyle().Foreground(t.ErrorFg).Bold(true)
	normal := lipgloss.NewStyle().Foreground(t.Border)
	hint := lipgloss.NewStyle().Foreground(t.Muted).Italic(true)

	var sb strings.Builder
	sb.WriteString(headStyle.Render("⚠ The agent wants to run a shell command") + "\n")
	sb.WriteString(dirStyle.Render("in "+b.dir) + "\n\n")
	sb.WriteString(cmdStyle.Render("  $ "+b.command) + "\n\n")

	if b.decided {
		if b.approved {
			sb.WriteString(approveSel.Render("✓ approved"))
		} else {
			sb.WriteString(rejectSel.Render("✗ rejected"))
		}
	} else {
		approve := "[ approve ]"
		reject := "[ reject ]"
		if b.cursor == 0 {
			approve = approveSel.Render("▶ approve ◀")
			reject = normal.Render("  reject  ")
		} else {
			approve = normal.Render("  approve  ")
			reject = rejectSel.Render("▶ reject ◀")
		}
		sb.WriteString("  " + approve + "    " + reject + "\n\n")
		sb.WriteString(hint.Render("←→/tab move • enter confirm • y approve • n reject"))
	}

	borderColor := t.WarnFg
	if b.decided {
		borderColor = t.BorderMuted
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width - 2).
		Render(sb.String())
}
