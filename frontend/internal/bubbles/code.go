package bubbles

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// codeMode controls how the CodeBubble renders its content.
type codeMode int

const (
	codeModeView codeMode = iota // plain view with line numbers
	codeModeDiff                 // diff-style: all lines shown as additions (+)
	codeModeFence                // LLM-generated snippet with language label
)

// CodeBubble shows the most recently written file with line numbers,
// or a diff view for newly written files, or a code fence snippet.
// Persists across turns — only reset explicitly via Reset().
type CodeBubble struct {
	path    string
	content string
	lang    string // for fence mode
	mode    codeMode
	theme   *theme.Theme
}

func NewCodeBubble(t *theme.Theme) *CodeBubble {
	return &CodeBubble{theme: t}
}

func (b *CodeBubble) Title() string           { return "code" }
func (b *CodeBubble) SetTheme(t *theme.Theme) { b.theme = t }
func (b *CodeBubble) HasContent() bool        { return b.content != "" }

func (b *CodeBubble) Update(path, content string) {
	b.path = path
	b.content = content
	b.mode = codeModeDiff
}

func (b *CodeBubble) UpdateFence(lang, content string) {
	b.path = ""
	b.lang = lang
	b.content = content
	b.mode = codeModeFence
}

func (b *CodeBubble) Reset() {
	b.path = ""
	b.content = ""
	b.lang = ""
	b.mode = codeModeView
}

func (b *CodeBubble) Handle(_ agent.Event) bool { return false }

func (b *CodeBubble) View(width int) string {
	t := b.theme
	titleStyle := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	fileStyle := lipgloss.NewStyle().Foreground(t.Muted).Italic(true)
	lineNumStyle := lipgloss.NewStyle().Foreground(t.Muted)
	codeStyle := lipgloss.NewStyle().Foreground(t.Text)
	addStyle := lipgloss.NewStyle().Foreground(t.Accent)
	langStyle := lipgloss.NewStyle().Foreground(t.WarnFg).Italic(true)

	innerWidth := width - 6
	if innerWidth < 20 {
		innerWidth = 20
	}

	lines := strings.Split(b.content, "\n")

	maxLines := 50
	truncated := len(lines) > maxLines
	if truncated {
		lines = lines[:maxLines]
	}

	var sb strings.Builder

	switch b.mode {
	case codeModeFence:
		label := b.lang
		if label == "" {
			label = "code"
		}
		sb.WriteString(titleStyle.Render("snippet") + "  " + langStyle.Render(label) + "\n")
		for i, line := range lines {
			num := lineNumStyle.Render(fmt.Sprintf("%3d │ ", i+1))
			if len(line) > innerWidth {
				line = line[:innerWidth-1] + "…"
			}
			sb.WriteString(num + codeStyle.Render(line))
			if i < len(lines)-1 {
				sb.WriteByte('\n')
			}
		}

	case codeModeDiff:
		base := filepath.Base(b.path)
		dir := filepath.Dir(b.path)
		header := titleStyle.Render("new file")
		if dir != "" && dir != "." {
			header += "  " + fileStyle.Render(dir+"/"+base)
		} else {
			header += "  " + fileStyle.Render(base)
		}
		sb.WriteString(header + "\n")
		for i, line := range lines {
			prefix := addStyle.Render(fmt.Sprintf("%3d + ", i+1))
			if len(line) > innerWidth {
				line = line[:innerWidth-1] + "…"
			}
			sb.WriteString(prefix + addStyle.Render(line))
			if i < len(lines)-1 {
				sb.WriteByte('\n')
			}
		}

	default:
		base := filepath.Base(b.path)
		dir := filepath.Dir(b.path)
		header := titleStyle.Render("code")
		if dir != "" && dir != "." {
			header += "  " + fileStyle.Render(dir+"/"+base)
		} else {
			header += "  " + fileStyle.Render(base)
		}
		sb.WriteString(header + "\n")
		for i, line := range lines {
			num := lineNumStyle.Render(fmt.Sprintf("%3d │ ", i+1))
			if len(line) > innerWidth {
				line = line[:innerWidth-1] + "…"
			}
			sb.WriteString(num + codeStyle.Render(line))
			if i < len(lines)-1 {
				sb.WriteByte('\n')
			}
		}
	}

	if truncated {
		total := len(strings.Split(b.content, "\n"))
		sb.WriteString("\n" + lineNumStyle.Render(fmt.Sprintf("    │ … (%d more lines)", total-maxLines)))
	}

	borderColor := t.ToolFg
	if b.mode == codeModeDiff {
		borderColor = t.Accent
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width - 2).
		Render(sb.String())
}
