package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type message struct {
	role string
	body string
}

type model struct {
	input    string
	cursor   int
	messages []message
	width    int
	height   int
	scroll   int
}

var (
	accent  = lipgloss.Color("#a855f7") // Green theme
	muted   = lipgloss.Color("#7F8490")
	text    = lipgloss.Color("#C9D1D9")
	success = lipgloss.Color("#7BD88F")
	codeFg  = lipgloss.Color("#E6EDF3")
	codeBg  = lipgloss.Color("#22272E")

	titleStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	mutedStyle = lipgloss.NewStyle().Foreground(muted)
	userStyle  = lipgloss.NewStyle().Foreground(accent).Bold(true)
	aiStyle    = lipgloss.NewStyle().Foreground(text)
	codeStyle  = lipgloss.NewStyle().Foreground(codeFg).Background(codeBg).Padding(0, 1)

	sidebarTitleStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
)

const asciiArt = `
   ▄▄       ▄▄▄▄
  ████      ████
   ▀▀       ████
            ████
  ▄▄▄▄      ████
  ████▄▄▄▄▄▄████
  ████▀▀▀▀▀▀▀▀▀▀
  ████          
  ████       ▄▄ 
  ████      ████
  ▀▀▀▀       ▀▀ 
`

func initialModel() model {
	return model{}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			m = m.submit()
			return m, nil
		case "pgup":
			m.scroll += max(4, m.height/2)
			return m, nil
		case "pgdown":
			m.scroll = max(0, m.scroll-max(4, m.height/2))
			return m, nil
		case "backspace", "ctrl+h":
			m = m.deleteBeforeCursor()
			return m, nil
		case "delete":
			m = m.deleteAtCursor()
			return m, nil
		case "left":
			m.cursor = max(0, m.cursor-1)
			return m, nil
		case "right":
			m.cursor = min(len([]rune(m.input)), m.cursor+1)
			return m, nil
		case "ctrl+a", "home":
			m.cursor = 0
			return m, nil
		case "ctrl+e", "end":
			m.cursor = len([]rune(m.input))
			return m, nil
		}

		if len(msg.Runes) > 0 {
			m = m.insert(string(msg.Runes))
			return m, nil
		}
	}

	return m, cmd
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	// Layout dimensions
	boxBorder := 2    // Left + Right borders
	promptHeight := 3 // 1 line prompt, 1 line padding, 1 line separator

	boxHeight := m.height - promptHeight - boxBorder
	if boxHeight < 5 {
		boxHeight = 5
	}
	boxWidth := m.width - boxBorder

	// Sidebar width (30%)
	sidebarWidth := (boxWidth * 3) / 10
	if sidebarWidth < 25 {
		sidebarWidth = 25
	}
	mainWidth := boxWidth - sidebarWidth

	// --- Left Pane (Main Content) ---
	var mainLines []string
	if len(m.messages) == 0 {
		// Welcome screen
		welcomeText := titleStyle.Render("Welcome to Hardcore AI!")

		artLines := strings.Split(strings.Trim(asciiArt, "\n\r"), "\n")

		maxArtWidth := 0
		for _, line := range artLines {
			w := lipgloss.Width(line)
			if w > maxArtWidth {
				maxArtWidth = w
			}
		}

		// Center vertically
		vPad := (boxHeight - len(artLines) - 2) / 2
		if vPad < 0 {
			vPad = 0
		}

		for i := 0; i < vPad; i++ {
			mainLines = append(mainLines, "")
		}

		// Add centered welcome text
		padText := (mainWidth - lipgloss.Width(welcomeText)) / 2
		if padText < 0 {
			padText = 0
		}
		mainLines = append(mainLines, strings.Repeat(" ", padText)+welcomeText)
		mainLines = append(mainLines, "")

		padArt := (mainWidth - maxArtWidth) / 2
		if padArt < 0 {
			padArt = 0
		}

		for _, line := range artLines {
			mainLines = append(mainLines, strings.Repeat(" ", padArt)+userStyle.Render(line))
		}
	} else {
		// Chat History
		for _, msg := range m.messages {
			prefix := userStyle.Render("›")
			if msg.role == "ai" {
				prefix = mutedStyle.Render("·")
			}
			rendered := renderMarkdown(msg.body, mainWidth-4)
			for i, line := range strings.Split(rendered, "\n") {
				if strings.TrimSpace(line) == "" {
					mainLines = append(mainLines, "")
					continue
				}
				if i == 0 {
					mainLines = append(mainLines, prefix+" "+line)
					continue
				}
				mainLines = append(mainLines, "  "+line)
			}
			mainLines = append(mainLines, "")
		}
		mainLines = visibleLines(mainLines, boxHeight, m.scroll)
	}

	// Pad main lines to exactly boxHeight
	for len(mainLines) < boxHeight {
		mainLines = append(mainLines, "")
	}
	mainPane := lipgloss.NewStyle().Width(mainWidth).Height(boxHeight).Render(strings.Join(mainLines, "\n"))

	// --- Right Pane (Sidebar) ---
	var sidebarLines []string
	sidebarLines = append(sidebarLines, sidebarTitleStyle.Render("Hardware Options"))
	sidebarLines = append(sidebarLines, mutedStyle.Render("Select your target device:"))
	sidebarLines = append(sidebarLines, "")
	sidebarLines = append(sidebarLines, "1. ST-Link V2")
	sidebarLines = append(sidebarLines, "2. ST-Link V3")
	sidebarLines = append(sidebarLines, "3. J-Link EDU")
	sidebarLines = append(sidebarLines, "4. STM32 Discovery")
	sidebarLines = append(sidebarLines, "5. OpenOCD Custom")
	sidebarLines = append(sidebarLines, "")
	sidebarLines = append(sidebarLines, sidebarTitleStyle.Render("Recent activity"))
	sidebarLines = append(sidebarLines, mutedStyle.Render("No recent hardware faults"))

	for len(sidebarLines) < boxHeight {
		sidebarLines = append(sidebarLines, "")
	}

	sidebarInnerWidth := sidebarWidth - 2
	if sidebarInnerWidth < 1 {
		sidebarInnerWidth = 1
	}

	sidebarPane := lipgloss.NewStyle().
		Width(sidebarInnerWidth).
		Height(boxHeight).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(accent).
		PaddingLeft(1).
		Render(strings.Join(sidebarLines, "\n"))

	// --- Combine Top Box ---
	topContent := lipgloss.JoinHorizontal(lipgloss.Top, mainPane, sidebarPane)

	// Create Top Border
	titleStr := " Hardcore AI v0.1.0 "
	titleLen := lipgloss.Width(titleStr)
	dashCount := boxWidth - titleLen - 1
	if dashCount < 0 {
		dashCount = 0
	}

	borderStyle := lipgloss.NewStyle().Foreground(accent)

	topBorder := borderStyle.Render("╭─") + titleStyle.Render(titleStr) + borderStyle.Render(strings.Repeat("─", dashCount)+"╮")
	bottomBorder := borderStyle.Render("╰" + strings.Repeat("─", boxWidth) + "╯")

	var finalBoxLines []string
	finalBoxLines = append(finalBoxLines, topBorder)
	for _, line := range strings.Split(topContent, "\n") {
		// Truncate line if it exceeds boxWidth to prevent layout breaking
		visualWidth := lipgloss.Width(line)
		if visualWidth > boxWidth {
			// Basic truncation (this is naive but prevents major layout breaks)
			// A true truncation would be complex with ANSI escapes, we assume lipgloss handled it well
		}

		// Fill space if lipgloss didn't expand fully
		if visualWidth < boxWidth {
			line += strings.Repeat(" ", boxWidth-visualWidth)
		}

		finalBoxLines = append(finalBoxLines, borderStyle.Render("│")+line+borderStyle.Render("│"))
	}
	finalBoxLines = append(finalBoxLines, bottomBorder)

	mainUI := strings.Join(finalBoxLines, "\n")

	// --- Bottom Box (Input) ---
	promptLine := m.inputView(m.width)

	return lipgloss.JoinVertical(lipgloss.Left, mainUI, "", promptLine)
}

func (m model) submit() model {
	value := strings.TrimSpace(m.input)
	if value == "" {
		return m
	}

	m.messages = append(m.messages, message{role: "user", body: value})
	m.input = ""
	m.cursor = 0
	m.scroll = 0

	switch value {
	case "/quit", "quit", "exit":
		m.messages = append(m.messages, message{role: "ai", body: "Use `esc` or `ctrl+c` to quit."})
	case "/clear", "clear":
		m.messages = nil
	case "/help", "help":
		m.messages = append(m.messages, message{role: "ai", body: "# Commands\n- `/help` show commands\n- `/clear` clear the session\n- `esc` quit\n\nMarkdown works: **bold**, `code`, headings, lists, quotes, and fenced code."})
	default:
		m.messages = append(m.messages, message{role: "ai", body: "Frontend scaffold is alive. Backend diagnostics are not wired yet."})
	}

	return m
}

func (m model) inputView(width int) string {
	runes := []rune(m.input)
	cursor := min(m.cursor, len(runes))
	before := string(runes[:cursor])
	after := string(runes[cursor:])
	cursorText := " "
	if cursor < len(runes) {
		cursorText = string(runes[cursor])
		after = string(runes[cursor+1:])
	}

	var line string
	if strings.TrimSpace(m.input) == "" {
		line = titleStyle.Render("> ") + mutedStyle.Render("Try \"how do I diagnose an ST-Link error?\"")
	} else {
		line = titleStyle.Render("> ") + aiStyle.Render(before) + lipgloss.NewStyle().
			Foreground(lipgloss.Color("#101417")).
			Background(accent).
			Render(cursorText) + aiStyle.Render(after)
	}

	return lipgloss.NewStyle().Width(max(1, width)).Render(line)
}

func (m model) insert(value string) model {
	runes := []rune(m.input)
	cursor := min(m.cursor, len(runes))
	next := append([]rune{}, runes[:cursor]...)
	next = append(next, []rune(value)...)
	next = append(next, runes[cursor:]...)
	m.input = string(next)
	m.cursor = cursor + len([]rune(value))
	return m
}

func (m model) deleteBeforeCursor() model {
	runes := []rune(m.input)
	if m.cursor == 0 || len(runes) == 0 {
		return m
	}

	cursor := min(m.cursor, len(runes))
	next := append([]rune{}, runes[:cursor-1]...)
	next = append(next, runes[cursor:]...)
	m.input = string(next)
	m.cursor = cursor - 1
	return m
}

func (m model) deleteAtCursor() model {
	runes := []rune(m.input)
	if m.cursor >= len(runes) {
		return m
	}

	cursor := min(m.cursor, len(runes))
	next := append([]rune{}, runes[:cursor]...)
	next = append(next, runes[cursor+1:]...)
	m.input = string(next)
	return m
}

func visibleLines(lines []string, height int, scroll int) []string {
	if len(lines) <= height {
		return lines
	}

	end := len(lines) - scroll
	if end > len(lines) {
		end = len(lines)
	}
	if end < height {
		end = height
	}

	return lines[end-height : end]
}

func renderMarkdown(markdown string, width int) string {
	var out []string
	inCode := false

	for _, raw := range strings.Split(markdown, "\n") {
		line := strings.TrimRight(raw, " ")
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			inCode = !inCode
			continue
		}

		if inCode {
			out = append(out, codeStyle.Width(max(1, width-2)).Render(line))
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, "# "):
			out = append(out, titleStyle.Render(strings.TrimPrefix(trimmed, "# ")))
		case strings.HasPrefix(trimmed, "## "):
			out = append(out, userStyle.Render(strings.TrimPrefix(trimmed, "## ")))
		case strings.HasPrefix(trimmed, "- "):
			out = append(out, mutedStyle.Render("• ")+inline(strings.TrimPrefix(trimmed, "- ")))
		case strings.HasPrefix(trimmed, "> "):
			out = append(out, mutedStyle.Render("│ ")+inline(strings.TrimPrefix(trimmed, "> ")))
		default:
			out = append(out, inline(trimmed))
		}
	}

	return strings.Join(out, "\n")
}

func inline(s string) string {
	s = renderPairs(s, "`", func(value string) string {
		return codeStyle.Render(value)
	})
	s = renderPairs(s, "**", func(value string) string {
		return lipgloss.NewStyle().Bold(true).Render(value)
	})
	return aiStyle.Render(s)
}

func renderPairs(s string, marker string, render func(string) string) string {
	var b strings.Builder

	for {
		start := strings.Index(s, marker)
		if start == -1 {
			b.WriteString(s)
			return b.String()
		}

		end := strings.Index(s[start+len(marker):], marker)
		if end == -1 {
			b.WriteString(s)
			return b.String()
		}

		b.WriteString(s[:start])
		contentStart := start + len(marker)
		contentEnd := contentStart + end
		b.WriteString(render(s[contentStart:contentEnd]))
		s = s[contentEnd+len(marker):]
	}
}

func main() {
	program := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to run hardcore-ai: %v\n", err)
		os.Exit(1)
	}
}
