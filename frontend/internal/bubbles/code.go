package bubbles

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
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
	path      string
	content   string
	lang      string // for fence mode
	mode      codeMode
	theme     *theme.Theme
	diffLines []diffLine
	scrollTop int // first visible line index (0-based)
}

type diffLine struct {
	op   int // 0=equal, 1=insert, -1=delete
	text string
}

func computeSimpleDiff(oldContent, newContent string) []diffLine {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")
	if oldContent == "" {
		var out []diffLine
		for _, l := range newLines {
			out = append(out, diffLine{1, l})
		}
		return out
	}
	start := 0
	for start < len(oldLines) && start < len(newLines) && oldLines[start] == newLines[start] {
		start++
	}
	endOld := len(oldLines) - 1
	endNew := len(newLines) - 1
	for endOld >= start && endNew >= start && oldLines[endOld] == newLines[endNew] {
		endOld--
		endNew--
	}
	var out []diffLine
	for i := 0; i < start; i++ {
		out = append(out, diffLine{0, oldLines[i]})
	}
	for i := start; i <= endOld; i++ {
		out = append(out, diffLine{-1, oldLines[i]})
	}
	for i := start; i <= endNew; i++ {
		out = append(out, diffLine{1, newLines[i]})
	}
	for i := endNew + 1; i < len(newLines); i++ {
		out = append(out, diffLine{0, newLines[i]})
	}
	return out
}

func NewCodeBubble(t *theme.Theme) *CodeBubble {
	return &CodeBubble{theme: t}
}

func (b *CodeBubble) Title() string           { return "code" }
func (b *CodeBubble) SetTheme(t *theme.Theme) { b.theme = t }
func (b *CodeBubble) HasContent() bool        { return b.content != "" }
func (b *CodeBubble) Content() string         { return b.content }

func (b *CodeBubble) Update(path, content string) {
	b.path = path
	b.content = content
	b.mode = codeModeDiff
	b.diffLines = computeSimpleDiff("", content)
}

func (b *CodeBubble) UpdateDiff(path, content, oldContent string) {
	b.path = path
	b.content = content
	b.mode = codeModeDiff
	b.diffLines = computeSimpleDiff(oldContent, content)
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

// visibleLines is how many code lines the bubble shows at once before scrolling.
const visibleLines = 25

// Scroll handles keyboard scrolling when the bubble is focused via tab.
// Implements the scrollable interface so MasonryManager.UpdateFocused works.
func (b *CodeBubble) Scroll(msg tea.Msg) {
	lines := b.displayLines()
	total := len(lines)
	maxTop := total - visibleLines
	if maxTop < 0 {
		maxTop = 0
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if b.scrollTop > 0 {
				b.scrollTop--
			}
		case "down", "j":
			if b.scrollTop < maxTop {
				b.scrollTop++
			}
		case "pgup":
			b.scrollTop -= visibleLines / 2
			if b.scrollTop < 0 {
				b.scrollTop = 0
			}
		case "pgdown":
			b.scrollTop += visibleLines / 2
			if b.scrollTop > maxTop {
				b.scrollTop = maxTop
			}
		case "home", "g":
			b.scrollTop = 0
		case "end", "G":
			b.scrollTop = maxTop
		}
	}
}

// displayLines returns the slice of renderable lines based on current mode.
func (b *CodeBubble) displayLines() []string {
	switch b.mode {
	case codeModeDiff:
		out := make([]string, len(b.diffLines))
		for i, dl := range b.diffLines {
			out[i] = dl.text
		}
		return out
	default:
		return strings.Split(b.content, "\n")
	}
}

func (b *CodeBubble) View(width int) string {
	t := b.theme
	titleStyle := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	fileStyle := lipgloss.NewStyle().Foreground(t.Muted).Italic(true)
	lineNumStyle := lipgloss.NewStyle().Foreground(t.Muted)
	codeStyle := lipgloss.NewStyle().Foreground(t.Text)
	addStyle := lipgloss.NewStyle().Foreground(t.Accent)
	delStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("167"))
	langStyle := lipgloss.NewStyle().Foreground(t.WarnFg).Italic(true)
	scrollStyle := lipgloss.NewStyle().Foreground(t.Muted).Italic(true)

	innerWidth := width - 6
	if innerWidth < 20 {
		innerWidth = 20
	}

	truncLine := func(s string) string {
		if len(s) > innerWidth {
			return s[:innerWidth-1] + "…"
		}
		return s
	}

	var sb strings.Builder

	switch b.mode {
	case codeModeFence:
		label := b.lang
		if label == "" {
			label = "code"
		}
		allLines := strings.Split(b.content, "\n")
		total := len(allLines)
		// Clamp scrollTop.
		maxTop := total - visibleLines
		if maxTop < 0 {
			maxTop = 0
		}
		if b.scrollTop > maxTop {
			b.scrollTop = maxTop
		}
		end := b.scrollTop + visibleLines
		if end > total {
			end = total
		}
		visible := allLines[b.scrollTop:end]

		sb.WriteString(titleStyle.Render("snippet") + "  " + langStyle.Render(label) + "\n")
		if b.scrollTop > 0 {
			sb.WriteString(scrollStyle.Render(fmt.Sprintf("  ↑ %d above (↑/↓ scroll)", b.scrollTop)) + "\n")
		}
		for i, line := range visible {
			absLine := b.scrollTop + i + 1
			num := lineNumStyle.Render(fmt.Sprintf("%3d │ ", absLine))
			sb.WriteString(num + codeStyle.Render(truncLine(line)))
			if i < len(visible)-1 {
				sb.WriteByte('\n')
			}
		}
		below := total - end
		if below > 0 {
			sb.WriteString("\n" + scrollStyle.Render(fmt.Sprintf("  ↓ %d below (↑/↓ scroll)", below)))
		}

	case codeModeDiff:
		base := filepath.Base(b.path)
		dir := filepath.Dir(b.path)
		header := titleStyle.Render("diff")
		if dir != "" && dir != "." {
			header += "  " + fileStyle.Render(dir+"/"+base)
		} else {
			header += "  " + fileStyle.Render(base)
		}
		sb.WriteString(header + "\n")

		total := len(b.diffLines)
		maxTop := total - visibleLines
		if maxTop < 0 {
			maxTop = 0
		}
		if b.scrollTop > maxTop {
			b.scrollTop = maxTop
		}
		end := b.scrollTop + visibleLines
		if end > total {
			end = total
		}
		visible := b.diffLines[b.scrollTop:end]

		if b.scrollTop > 0 {
			sb.WriteString(scrollStyle.Render(fmt.Sprintf("  ↑ %d above (↑/↓ scroll)", b.scrollTop)) + "\n")
		}
		for i, dLine := range visible {
			absLine := b.scrollTop + i + 1
			line := truncLine(dLine.text)
			var prefix, rendered string
			if dLine.op == 1 {
				prefix = addStyle.Render(fmt.Sprintf("%3d + ", absLine))
				rendered = addStyle.Render(line)
			} else if dLine.op == -1 {
				prefix = delStyle.Render("    - ")
				rendered = delStyle.Render(line)
			} else {
				prefix = lineNumStyle.Render(fmt.Sprintf("%3d │ ", absLine))
				rendered = codeStyle.Render(line)
			}
			sb.WriteString(prefix + rendered)
			if i < len(visible)-1 {
				sb.WriteByte('\n')
			}
		}
		below := total - end
		if below > 0 {
			sb.WriteString("\n" + scrollStyle.Render(fmt.Sprintf("  ↓ %d below (↑/↓ scroll)", below)))
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

		allLines := strings.Split(b.content, "\n")
		total := len(allLines)
		maxTop := total - visibleLines
		if maxTop < 0 {
			maxTop = 0
		}
		if b.scrollTop > maxTop {
			b.scrollTop = maxTop
		}
		end := b.scrollTop + visibleLines
		if end > total {
			end = total
		}
		visible := allLines[b.scrollTop:end]

		if b.scrollTop > 0 {
			sb.WriteString(scrollStyle.Render(fmt.Sprintf("  ↑ %d above (↑/↓ scroll)", b.scrollTop)) + "\n")
		}
		for i, line := range visible {
			absLine := b.scrollTop + i + 1
			num := lineNumStyle.Render(fmt.Sprintf("%3d │ ", absLine))
			sb.WriteString(num + codeStyle.Render(truncLine(line)))
			if i < len(visible)-1 {
				sb.WriteByte('\n')
			}
		}
		below := total - end
		if below > 0 {
			sb.WriteString("\n" + scrollStyle.Render(fmt.Sprintf("  ↓ %d below (↑/↓ scroll)", below)))
		}
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

