package bubbles

import (
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

type buildStatus int

const (
	buildNone buildStatus = iota
	buildRunning
	buildOK
	buildFailed
)

// STM32StatusBubble is a persistent card that shows the active target,
// compile status, and emulation status. The TUI updates it via Set* methods.
type STM32StatusBubble struct {
	target   string
	compile  buildStatus
	emulate  buildStatus
	compMsg  string
	emuMsg   string
	theme    *theme.Theme
}

func NewSTM32StatusBubble(t *theme.Theme) *STM32StatusBubble {
	return &STM32StatusBubble{theme: t}
}

func (b *STM32StatusBubble) SetTheme(t *theme.Theme) { b.theme = t }
func (b *STM32StatusBubble) HasContent() bool         { return b.target != "" }

func (b *STM32StatusBubble) SetTarget(target string)  { b.target = target }
func (b *STM32StatusBubble) CompileStart()             { b.compile = buildRunning; b.compMsg = "" }
func (b *STM32StatusBubble) CompileOK(msg string)      { b.compile = buildOK; b.compMsg = msg }
func (b *STM32StatusBubble) CompileFail(msg string)    { b.compile = buildFailed; b.compMsg = msg }
func (b *STM32StatusBubble) EmulateStart()             { b.emulate = buildRunning; b.emuMsg = "" }
func (b *STM32StatusBubble) EmulateOK(msg string)      { b.emulate = buildOK; b.emuMsg = msg }
func (b *STM32StatusBubble) EmulateFail(msg string)    { b.emulate = buildFailed; b.emuMsg = msg }

func (b *STM32StatusBubble) Handle(ev agent.Event) bool {
	if _, ok := ev.(agent.UserMessageEvent); ok {
		b.target = ""
		b.compile = buildNone
		b.emulate = buildNone
		b.compMsg = ""
		b.emuMsg = ""
		return true
	}
	return false
}

func (b *STM32StatusBubble) View(width int) string {
	t := b.theme
	titleStyle := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(t.Muted)
	valStyle := lipgloss.NewStyle().Foreground(t.Text)
	okStyle := lipgloss.NewStyle().Foreground(t.Accent)
	failStyle := lipgloss.NewStyle().Foreground(t.ErrorFg).Bold(true)
	runStyle := lipgloss.NewStyle().Foreground(t.WarnFg)

	statusStr := func(s buildStatus, msg string) string {
		switch s {
		case buildRunning:
			return runStyle.Render("⠋ running…")
		case buildOK:
			short := truncate(strings.ReplaceAll(msg, "\n", " "), 60)
			return okStyle.Render("✓ "+short)
		case buildFailed:
			short := truncate(strings.ReplaceAll(msg, "\n", " "), 60)
			return failStyle.Render("✗ "+short)
		default:
			return labelStyle.Render("—")
		}
	}

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("stm32") + "\n")
	sb.WriteString(labelStyle.Render("target  ") + valStyle.Render(b.target) + "\n")
	sb.WriteString(labelStyle.Render("compile ") + statusStr(b.compile, b.compMsg) + "\n")
	sb.WriteString(labelStyle.Render("emulate ") + statusStr(b.emulate, b.emuMsg))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderMuted).
		Padding(0, 1).
		Width(width - 2).
		Render(sb.String())
}
