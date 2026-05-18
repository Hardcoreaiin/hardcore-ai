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
	art      string
	name     string
	msg      string
	thinking []string
	answer   []string
	phase    petPhase
	theme    *theme.Theme
}

type petPhase int

const (
	phaseIdle petPhase = iota
	phaseThinking
	phaseAnswering
)

// spinnerFrames is used during thinking.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

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
func (p *PetBubble) SetArt(art string)        { p.art = art }
func (p *PetBubble) SetName(name string)      { p.name = name }

func (p *PetBubble) Handle(ev agent.Event) {
	switch e := ev.(type) {
	case agent.UserMessageEvent:
		_ = e
		p.thinking = nil
		p.answer = nil
		p.phase = phaseThinking
		p.msg = ""
	case agent.ThinkEvent:
		if p.phase == phaseThinking {
			p.thinking = append(p.thinking, StripLatex(e.Text))
		}
	case agent.LineEvent:
		p.phase = phaseAnswering
		p.thinking = nil
		line := strings.TrimSpace(StripLatex(e.Text))
		if line != "" {
			p.answer = append(p.answer, line)
		}
	case agent.ToolStartEvent:
		p.phase = phaseAnswering
		p.thinking = nil
		p.answer = nil
		p.msg = "running " + e.Name + "…"
	case agent.ToolResultEvent:
		p.msg = "done " + e.Name + "."
	case agent.TurnEndEvent:
		p.phase = phaseIdle
		p.thinking = nil
		if len(p.answer) == 0 {
			p.msg = "ready."
		}
	case agent.ErrorEvent:
		p.phase = phaseIdle
		p.thinking = nil
		p.answer = nil
		p.msg = "something went wrong."
	}
}

// View renders the pet bubble. tick drives animations (increment externally on each clock tick).
func (p *PetBubble) View(width, tick int) string {
	t := p.theme
	petStyle := lipgloss.NewStyle().Foreground(t.Accent)
	nameStyle := lipgloss.NewStyle().Foreground(t.Muted).Italic(true)
	thinkStyle := lipgloss.NewStyle().Foreground(t.Muted).Italic(true)
	idleStyle := lipgloss.NewStyle().Foreground(t.Muted).Italic(true)
	spinStyle := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)

	// Build pet column: art + name
	artLines := strings.Split(p.art, "\n")
	petWidth := 0
	for _, l := range artLines {
		if len(l) > petWidth {
			petWidth = len(l)
		}
	}
	petColWidth := petWidth + 2

	petCol := append([]string{}, artLines...)
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
		spinner := spinnerFrames[tick%len(spinnerFrames)]
		if len(p.thinking) == 0 {
			speechLines = append(speechLines, spinStyle.Render(spinner)+" "+thinkStyle.Render("thinking…"))
		} else {
			// Show last think sentence, word-wrapped
			last := strings.TrimSpace(p.thinking[len(p.thinking)-1])
			wrapped := wordWrap(last, speechWidth-2)
			speechLines = append(speechLines, spinStyle.Render(spinner)+" "+thinkStyle.Render(wrapped[0]))
			for _, wl := range wrapped[1:] {
				speechLines = append(speechLines, "  "+thinkStyle.Render(wl))
			}
		}

	case phaseAnswering:
		if len(p.answer) > 0 {
			cursor := cursorFrames[(tick/3)%len(cursorFrames)]
			rendered := p.renderMarkdown(strings.Join(p.answer, "\n"), speechWidth, t)
			mdLines := strings.Split(rendered, "\n")
			for i, wl := range mdLines {
				if i == len(mdLines)-1 {
					speechLines = append(speechLines, wl+lipgloss.NewStyle().Foreground(t.Accent).Render(cursor))
				} else {
					speechLines = append(speechLines, wl)
				}
			}
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
		left := colStyle.Render(petStyle.Render(petCol[i]))
		rows = append(rows, left+"  "+speechLines[i])
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderMuted).
		Padding(0, 1).
		Width(width - 2).
		Render(strings.Join(rows, "\n"))
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
