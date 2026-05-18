package bubbles

import (
	"fmt"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// Welcome renders the Claude-CLI-style splash: a rounded box with the
// app version, a tamagotchi-ish bot, the cwd, and a tips column.
type Welcome struct {
	Version string
	User    string
	Cwd     string
	Pet     string // ASCII art for the tamagotchi
	Tips    []string
	News    []string
	theme   *theme.Theme
}

func NewWelcome(t *theme.Theme, version, user, cwd, pet string) *Welcome {
	return &Welcome{
		Version: version,
		User:    user,
		Cwd:     cwd,
		Pet:     pet,
		Tips: []string{
			"ask me to read or edit any file in this directory",
			"type /help to see commands",
		},
		News: []string{
			"per-directory settings live in .agent_settings/",
			"type / in the input to browse commands",
		},
		theme: t,
	}
}

func (w *Welcome) SetTheme(t *theme.Theme) { w.theme = t }
func (w *Welcome) SetPet(art string)       { w.Pet = art }

func (w *Welcome) View(width int) string {
	if width < 40 {
		width = 40
	}
	t := w.theme

	header := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).
		Render(fmt.Sprintf("hardcore-ai %s", w.Version))

	bot := lipgloss.NewStyle().Foreground(t.Accent).Render(w.Pet)
	greet := lipgloss.NewStyle().Foreground(t.Text).Bold(true).
		Render(fmt.Sprintf("welcome back %s!", w.User))
	cwd := lipgloss.NewStyle().Foreground(t.Muted).
		Render(w.Cwd)

	left := lipgloss.JoinVertical(lipgloss.Center, greet, bot, cwd)

	tipsHeader := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).
		Render("tips for getting started")
	tipLines := make([]string, 0, len(w.Tips))
	for _, tip := range w.Tips {
		tipLines = append(tipLines,
			lipgloss.NewStyle().Foreground(t.Text).Render("• "+tip))
	}
	newsHeader := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).
		Render("what's new")
	newsLines := make([]string, 0, len(w.News))
	for _, n := range w.News {
		newsLines = append(newsLines,
			lipgloss.NewStyle().Foreground(t.Text).Render("• "+n))
	}
	right := lipgloss.JoinVertical(lipgloss.Left,
		tipsHeader,
		strings.Join(tipLines, "\n"),
		"",
		newsHeader,
		strings.Join(newsLines, "\n"),
	)

	inner := width - 6
	leftW := inner / 3
	rightW := inner - leftW - 2

	leftBlock := lipgloss.NewStyle().Width(leftW).Align(lipgloss.Center).Render(left)
	rightBlock := lipgloss.NewStyle().Width(rightW).Render(right)
	sep := lipgloss.NewStyle().Foreground(t.BorderMuted).Render(strings.Repeat("│\n", lipgloss.Height(leftBlock)))

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, " "+sep+" ", rightBlock)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Accent).
		Padding(1, 2).
		Width(width - 2)

	titled := lipgloss.JoinVertical(lipgloss.Left, header, "", body)
	return box.Render(titled)
}
