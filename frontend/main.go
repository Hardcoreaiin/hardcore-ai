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
	accent     = lipgloss.Color("#00A7C7")
	muted      = lipgloss.Color("#7F8490")
	text       = lipgloss.Color("#C9D1D9")
	success    = lipgloss.Color("#7BD88F")
	codeFg     = lipgloss.Color("#E6EDF3")
	codeBg     = lipgloss.Color("#22272E")
	titleStyle = lipgloss.NewStyle().Foreground(success).Bold(true)
	mutedStyle = lipgloss.NewStyle().Foreground(muted)
	userStyle  = lipgloss.NewStyle().Foreground(accent).Bold(true)
	aiStyle    = lipgloss.NewStyle().Foreground(text)
	codeStyle  = lipgloss.NewStyle().Foreground(codeFg).Background(codeBg).Padding(0, 1)
)

func initialModel() model {
	return model{
		messages: []message{
			{role: "ai", body: "**hardcore-ai** ready. Type `/help` for commands."},
		},
	}
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
	width := max(40, m.width)
	height := max(10, m.height)
	bodyHeight := max(1, height-4)

	lines := []string{
		titleStyle.Render("> hardcore-ai") + mutedStyle.Render("  simple hardware assistant"),
		"",
	}

	for _, msg := range m.messages {
		prefix := userStyle.Render("›")
		if msg.role == "ai" {
			prefix = mutedStyle.Render("·")
		}

		rendered := renderMarkdown(msg.body, width-4)
		for i, line := range strings.Split(rendered, "\n") {
			if strings.TrimSpace(line) == "" {
				lines = append(lines, "")
				continue
			}
			if i == 0 {
				lines = append(lines, prefix+" "+line)
				continue
			}
			lines = append(lines, "  "+line)
		}
		lines = append(lines, "")
	}

	lines = visibleLines(lines, bodyHeight, m.scroll)
	lines = append(lines, m.inputView(width))
	lines = append(lines, mutedStyle.Render("enter send  pgup/pgdown scroll  esc quit"))

	return strings.Join(lines, "\n")
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

	line := userStyle.Render("› ") + aiStyle.Render(before) + lipgloss.NewStyle().
		Foreground(lipgloss.Color("#101417")).
		Background(accent).
		Render(cursorText) + aiStyle.Render(after)

	if strings.TrimSpace(m.input) == "" {
		line += mutedStyle.Render(" ask hardcore-ai")
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
	program := tea.NewProgram(initialModel())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to run hardcore-ai: %v\n", err)
		os.Exit(1)
	}
}
