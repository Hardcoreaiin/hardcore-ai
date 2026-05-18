package bubbles

import (
	"fmt"

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
	if width < 48 {
		width = 48
	}
	boxW := width - 4
	if boxW > 104 {
		boxW = 104
	}
	if boxW < 44 {
		boxW = 44
	}
	t := w.theme

	header := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).
		Render(fmt.Sprintf("hardcore-ai %s", w.Version))

	bot := renderPetArt(w.Pet, t)
	greet := lipgloss.NewStyle().Foreground(t.Text).Bold(true).
		Render(fmt.Sprintf("welcome back %s!", w.User))
	cwd := lipgloss.NewStyle().Foreground(t.Muted).
		Render(truncateMiddle(w.Cwd, boxW/2))

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
		lipgloss.JoinVertical(lipgloss.Left, tipLines...),
		"",
		newsHeader,
		lipgloss.JoinVertical(lipgloss.Left, newsLines...),
	)

	inner := boxW - 6
	leftW := inner * 42 / 100
	if leftW < 28 {
		leftW = 28
	}
	rightW := inner - leftW - 3
	if rightW < 24 {
		rightW = 24
		leftW = inner - rightW - 3
	}

	leftBlock := lipgloss.NewStyle().Width(leftW).Align(lipgloss.Center).Render(left)
	rightBlock := lipgloss.NewStyle().
		Width(rightW).
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(t.BorderMuted).
		PaddingLeft(2).
		Render(right)

	body := lipgloss.JoinHorizontal(lipgloss.Center, leftBlock, "  ", rightBlock)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Accent).
		Padding(1, 2).
		Width(boxW)

	titled := lipgloss.JoinVertical(lipgloss.Left, header, "", body)
	return box.Render(titled)
}

func truncateMiddle(s string, max int) string {
	r := []rune(s)
	if max < 12 || len(r) <= max {
		return s
	}
	keep := max - 1
	left := keep / 2
	right := keep - left
	return string(r[:left]) + "…" + string(r[len(r)-right:])
}
