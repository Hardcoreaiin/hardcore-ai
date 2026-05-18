package bubbles

import (
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// PetBubble renders the ASCII pet on the left with a speech area on the right.
// During a turn it streams think tokens live, then shows the final answer.
// Pass the current tick counter from a time.Ticker for animations.
type PetBubble struct {
	art        string
	name       string
	msg        string
	thinking   []string
	answer     []string
	liveTokens strings.Builder // raw tokens accumulating before a full line
	phase      petPhase
	theme      *theme.Theme
}

type petPhase int

const (
	phaseIdle petPhase = iota
	phaseThinking
	phaseAnswering
)

// cursorFrames blinks during answering.
var cursorFrames = []string{"▌", " "}

func NewPetBubble(t *theme.Theme, art, name string) *PetBubble {
	return &PetBubble{
		art:   art,
		name:  name,
		msg:   "ready.",
		phase: phaseIdle,
		theme: t,
	}
}

func (p *PetBubble) SetTheme(t *theme.Theme) { p.theme = t }
func (p *PetBubble) SetArt(art string)       { p.art = art }
func (p *PetBubble) SetName(name string)     { p.name = name }

func (p *PetBubble) Handle(ev agent.Event) {
	switch e := ev.(type) {
	case agent.UserMessageEvent:
		_ = e
		p.thinking = nil
		p.answer = nil
		p.liveTokens.Reset()
		p.phase = phaseThinking
		p.msg = ""
	case agent.TokenEvent:
		// Stream raw tokens into the pet live — shows every character as it arrives.
		// Strip newlines so the rolling display stays on one conceptual line.
		tok := strings.Map(func(r rune) rune {
			if r == '\n' || r == '\r' {
				return ' '
			}
			return r
		}, e.Text)
		p.liveTokens.WriteString(tok)
		// Keep a bounded buffer — last 300 chars is plenty for one speech bubble.
		if p.liveTokens.Len() > 300 {
			s := p.liveTokens.String()
			p.liveTokens.Reset()
			p.liveTokens.WriteString(s[len(s)-300:])
		}
		if p.phase == phaseThinking {
			p.phase = phaseAnswering
		}
	case agent.ThinkEvent:
		if p.phase == phaseThinking || p.phase == phaseAnswering {
			p.thinking = append(p.thinking, StripLatex(e.Text))
		}
	case agent.LineEvent:
		line := strings.TrimSpace(StripLatex(e.Text))
		if line != "" {
			p.answer = append(p.answer, line)
		}
	case agent.ToolStartEvent:
		p.liveTokens.Reset()
		p.thinking = nil
		p.answer = nil
		p.phase = phaseAnswering
		p.msg = "running " + e.Name + "…"
	case agent.ToolResultEvent:
		p.liveTokens.Reset()
		p.msg = "done " + e.Name + "."
	case agent.TurnEndEvent:
		p.phase = phaseIdle
		p.thinking = nil
		p.liveTokens.Reset()
		if len(p.answer) == 0 {
			p.msg = "ready."
		}
	case agent.ErrorEvent:
		p.phase = phaseIdle
		p.thinking = nil
		p.answer = nil
		p.liveTokens.Reset()
		p.msg = "something went wrong."
	}
}

// View renders the pet bubble. tick drives animations (increment externally on each clock tick).
func (p *PetBubble) View(width, tick int) string {
	t := p.theme
	nameStyle := lipgloss.NewStyle().Foreground(t.Muted).Italic(true)
	thinkStyle := lipgloss.NewStyle().Foreground(t.Muted).Italic(true)
	idleStyle := lipgloss.NewStyle().Foreground(t.Muted).Italic(true)

	// Build pet column: art + name
	artLines := strings.Split(p.art, "\n")
	renderedArt := strings.Split(renderPetArt(p.art, t), "\n")
	petWidth := 0
	for _, l := range artLines {
		if w := lipgloss.Width(l); w > petWidth {
			petWidth = w
		}
	}
	petColWidth := petWidth + 2

	petCol := append([]string{}, renderedArt...)
	if p.name != "" {
		petCol = append(petCol, nameStyle.Render(p.name))
	}

	// Speech column width — fill remaining space inside the border+padding
	speechWidth := width - petColWidth - 6
	if speechWidth < 20 {
		speechWidth = 20
	}

	// Build speech lines (word-wrapped, no truncation)
	var speechLines []string
	switch p.phase {
	case phaseThinking:
		cursor := cursorFrames[(tick/3)%len(cursorFrames)]
		speechLines = append(speechLines,
			thinkStyle.Render("waiting for tokens")+lipgloss.NewStyle().Foreground(t.Accent).Render(cursor),
		)

	case phaseAnswering:
		cursor := cursorFrames[(tick/3)%len(cursorFrames)]
		live := p.liveTokens.String()
		if live != "" {
			// Show a rolling window of the last speechWidth chars of live tokens.
			runes := []rune(live)
			if len(runes) > speechWidth-2 {
				runes = runes[len(runes)-(speechWidth-2):]
			}
			display := string(runes)
			speechLines = append(speechLines,
				thinkStyle.Render(display)+lipgloss.NewStyle().Foreground(t.Accent).Render(cursor),
			)
		} else {
			speechLines = append(speechLines, idleStyle.Render(p.msg))
		}

	default: // phaseIdle
		if len(p.answer) > 0 {
			rendered := p.renderMarkdown(strings.Join(p.answer, "\n"), speechWidth, t)
			for _, wl := range strings.Split(rendered, "\n") {
				speechLines = append(speechLines, wl)
			}
		} else {
			speechLines = append(speechLines, idleStyle.Render(p.msg))
		}
	}

	// Pad both columns to the same height
	totalHeight := len(petCol)
	if len(speechLines) > totalHeight {
		totalHeight = len(speechLines)
	}
	for len(petCol) < totalHeight {
		petCol = append(petCol, "")
	}
	for len(speechLines) < totalHeight {
		speechLines = append(speechLines, "")
	}

	colStyle := lipgloss.NewStyle().Width(petColWidth)
	var rows []string
	for i := 0; i < totalHeight; i++ {
		left := colStyle.Render(petCol[i])
		rows = append(rows, left+"  "+speechLines[i])
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderMuted).
		Padding(0, 1).
		Width(width - 2).
		Render(strings.Join(rows, "\n"))
}

func renderPetArt(art string, t *theme.Theme) string {
	if t == nil {
		return art
	}
	outline := lipgloss.NewStyle().Foreground(t.Accent)
	eye := lipgloss.NewStyle().Foreground(t.UserFg).Bold(true)
	shade := lipgloss.NewStyle().Foreground(t.BorderMuted)
	dark := lipgloss.NewStyle().Foreground(t.Muted)

	var b strings.Builder
	for _, r := range art {
		switch r {
		case 'o', 'O', '0':
			b.WriteString(eye.Render(string(r)))
		case '.', ':', ';':
			b.WriteString(shade.Render(string(r)))
		case '\'', '"', '`':
			b.WriteString(shade.Render(string(r)))
		case '^', '-', '_', '=':
			b.WriteString(dark.Render(string(r)))
		case '\n':
			b.WriteRune(r)
		case ' ':
			b.WriteRune(r)
		default:
			b.WriteString(outline.Render(string(r)))
		}
	}
	return b.String()
}

func (p *PetBubble) renderMarkdown(src string, width int, t *theme.Theme) string {
	style := "dark"
	if t != nil && t.Name == "light" {
		style = "light"
	}
	if width < 20 {
		width = 20
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(style),
		glamour.WithWordWrap(width),
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

// wordWrap splits text into lines of at most maxWidth runes, breaking on spaces.
func wordWrap(text string, maxWidth int) []string {
	if maxWidth < 1 {
		maxWidth = 1
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	var cur strings.Builder
	for _, w := range words {
		if cur.Len() == 0 {
			cur.WriteString(w)
		} else if cur.Len()+1+len(w) <= maxWidth {
			cur.WriteByte(' ')
			cur.WriteString(w)
		} else {
			lines = append(lines, cur.String())
			cur.Reset()
			cur.WriteString(w)
		}
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return lines
}
