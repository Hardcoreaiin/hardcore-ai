package bubbles

import (
	"fmt"
	"os"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/toolchain"
	"github.com/charmbracelet/lipgloss"
)

// BuildStatusBubble shows the active chip/arch target, build status, and emulation status.
type BuildStatusBubble struct {
	target       string
	build        buildStatus
	emulate      buildStatus
	buildMsg     string
	emuMsg       string
	project      string
	installTool  toolchain.Tool
	installStage string
	installDone  int64
	installTotal int64
	installErr   error
	theme        *theme.Theme
}

func NewBuildStatusBubble(t *theme.Theme) *BuildStatusBubble {
	return &BuildStatusBubble{theme: t}
}

func (b *BuildStatusBubble) Title() string           { return "build" }
func (b *BuildStatusBubble) SetTheme(t *theme.Theme) { b.theme = t }
func (b *BuildStatusBubble) HasContent() bool        { return b.target != "" || b.project != "" }

func (b *BuildStatusBubble) SetTarget(target string)   { b.target = target }
func (b *BuildStatusBubble) SetProject(project string) { b.project = project }
func (b *BuildStatusBubble) BuildStart()               { b.build = buildRunning; b.buildMsg = "" }
func (b *BuildStatusBubble) BuildOK(msg string)        { b.build = buildOK; b.buildMsg = msg }
func (b *BuildStatusBubble) BuildFail(msg string)      { b.build = buildFailed; b.buildMsg = msg }
func (b *BuildStatusBubble) EmulateStart()             { b.emulate = buildRunning; b.emuMsg = "" }
func (b *BuildStatusBubble) EmulateOK(msg string)      { b.emulate = buildOK; b.emuMsg = msg }
func (b *BuildStatusBubble) EmulateFail(msg string)    { b.emulate = buildFailed; b.emuMsg = msg }

func (b *BuildStatusBubble) ToolchainEvent(ev toolchain.Event) {
	b.installTool = ev.Tool
	b.installStage = ev.Stage
	b.installDone = ev.Downloaded
	b.installTotal = ev.Total
	b.installErr = ev.Err
}

func (b *BuildStatusBubble) Handle(ev agent.Event) bool {
	if _, ok := ev.(agent.UserMessageEvent); ok {
		b.target = ""
		b.project = ""
		b.build = buildNone
		b.emulate = buildNone
		b.buildMsg = ""
		b.emuMsg = ""
		b.installTool = ""
		b.installStage = ""
		b.installDone = 0
		b.installTotal = 0
		b.installErr = nil
		return true
	}
	return false
}

func (b *BuildStatusBubble) View(width int) string {
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
			return okStyle.Render("✓ " + short)
		case buildFailed:
			short := truncate(strings.ReplaceAll(msg, "\n", " "), 60)
			return failStyle.Render("✗ " + short)
		default:
			return labelStyle.Render("—")
		}
	}

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("embedded") + "\n")
	if b.project != "" {
		home, _ := os.UserHomeDir()
		wsPath := home + "/.hardcoreai/workspace/" + b.project
		sb.WriteString(labelStyle.Render("project ") + valStyle.Render(b.project) + "\n")
		sb.WriteString(labelStyle.Render("path    ") + valStyle.Render(wsPath) + "\n")
	}
	if b.target != "" {
		sb.WriteString(labelStyle.Render("arch    ") + valStyle.Render(b.target) + "\n")
	}
	sb.WriteString(labelStyle.Render("build   ") + statusStr(b.build, b.buildMsg) + "\n")
	sb.WriteString(labelStyle.Render("emulate ") + statusStr(b.emulate, b.emuMsg))
	if b.installStage != "" {
		sb.WriteString("\n")
		status := b.installStage
		if b.installErr != nil {
			status = failStyle.Render("error: " + truncate(b.installErr.Error(), 60))
		} else if b.installStage == "download" {
			status = runStyle.Render(downloadStatus(b.installDone, b.installTotal))
		} else if b.installStage == "extract" {
			status = runStyle.Render("extracting…")
		} else if b.installStage == "ready" {
			status = okStyle.Render("ready")
		}
		sb.WriteString(labelStyle.Render("toolchain ") + valStyle.Render(string(b.installTool)) + " " + status)
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderMuted).
		Padding(0, 1).
		Width(width - 2).
		Render(sb.String())
}

func downloadStatus(done, total int64) string {
	if total <= 0 {
		return fmt.Sprintf("downloading %.1f MB…", float64(done)/(1024*1024))
	}
	pct := float64(done) * 100 / float64(total)
	return fmt.Sprintf("downloading %.1f/%.1f MB (%.0f%%)…",
		float64(done)/(1024*1024),
		float64(total)/(1024*1024),
		pct,
	)
}
