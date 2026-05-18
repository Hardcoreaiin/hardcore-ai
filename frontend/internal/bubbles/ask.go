package bubbles

import (
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// AskBubble renders an inline questionnaire. The caller drives selection via
// MoveUp/MoveDown and reads the answer via Answer()/IsOther().
// When IsOther() is true the caller must read OtherInput() for the free text.
type AskBubble struct {
	question string
	options  []string // model-provided options; "Other…" is always appended
	cursor   int
	answered bool
	other    textinput.Model
	theme    *theme.Theme
}

const otherLabel = "Other…"

func NewAskBubble(t *theme.Theme, question string, options []string) *AskBubble {
	ti := textinput.New()
	ti.Placeholder = "type your answer…"
	ti.CharLimit = 500

	all := make([]string, len(options))
	copy(all, options)

	return &AskBubble{
		question: question,
		options:  all,
		cursor:   0,
		other:    ti,
		theme:    t,
	}
}

func (b *AskBubble) SetTheme(t *theme.Theme) { b.theme = t }
func (b *AskBubble) Question() string        { return b.question }

// totalOptions is model options + the synthetic "Other…" entry.
func (b *AskBubble) totalOptions() int { return len(b.options) + 1 }
func (b *AskBubble) isOtherRow() bool  { return b.cursor == len(b.options) }

func (b *AskBubble) MoveUp() {
	if b.answered {
		return
	}
	if b.cursor > 0 {
		b.cursor--
	}
	if b.isOtherRow() {
		b.other.Blur()
	}
}

func (b *AskBubble) MoveDown() {
	if b.answered {
		return
	}
	if b.cursor < b.totalOptions()-1 {
		b.cursor++
	}
	if b.isOtherRow() {
		b.other.Focus()
	} else {
		b.other.Blur()
	}
}

// Confirm selects the current option. Returns false if "Other" is selected but
// the text input is empty (caller should keep waiting).
func (b *AskBubble) Confirm() bool {
	if b.answered {
		return true
	}
	if b.isOtherRow() && strings.TrimSpace(b.other.Value()) == "" {
		return false
	}
	b.answered = true
	b.other.Blur()
	return true
}

func (b *AskBubble) Answered() bool { return b.answered }

// IsOther reports whether the user picked the free-text option.
func (b *AskBubble) IsOther() bool { return b.answered && b.cursor == len(b.options) }

// Answer returns the chosen answer string (model option or free text).
func (b *AskBubble) Answer() string {
	if !b.answered {
		return ""
	}
	if b.IsOther() {
		return strings.TrimSpace(b.other.Value())
	}
	return b.options[b.cursor]
}

// UpdateOtherInput forwards key/mouse messages to the text input when "Other"
// is selected. Returns any command the text input needs.
func (b *AskBubble) UpdateOtherInput(msg interface{ isMsg() }) {}

// HandleRune feeds a character into the Other text input (called by the TUI).
func (b *AskBubble) OtherInput() *textinput.Model { return &b.other }

func (b *AskBubble) View(width int) string {
	t := b.theme
	questionStyle := lipgloss.NewStyle().Foreground(t.UserFg).Bold(true)
	selectedStyle := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(t.Border)
	answeredStyle := lipgloss.NewStyle().Foreground(t.Muted).Italic(true)
	otherStyle := lipgloss.NewStyle().Foreground(t.Accent)

	var sb strings.Builder
	sb.WriteString(questionStyle.Render(b.question) + "\n")

	allOptions := append(b.options, otherLabel)
	for i, opt := range allOptions {
		cursor := "  "
		var label string
		switch {
		case b.answered && i == b.cursor:
			cursor = "✓ "
			label = answeredStyle.Render(opt)
		case !b.answered && i == b.cursor:
			cursor = "▶ "
			if i == len(b.options) {
				label = otherStyle.Render(opt)
			} else {
				label = selectedStyle.Render(opt)
			}
		default:
			label = normalStyle.Render(opt)
		}

		sb.WriteString(cursor + label)

		// When "Other" row is selected and unanswered, show the text input inline.
		if i == len(b.options) && !b.answered && b.isOtherRow() {
			sb.WriteString("\n    " + b.other.View())
		}

		if i < len(allOptions)-1 {
			sb.WriteByte('\n')
		}
	}

	borderColor := t.BorderMuted
	if !b.answered {
		borderColor = t.Accent
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width - 2).
		Render(sb.String())
}
