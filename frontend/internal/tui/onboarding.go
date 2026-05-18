package tui

import (
	"fmt"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/pets"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

type onboardStep int

const (
	stepTheme   onboardStep = iota
	stepPet                 // pick species
	stepPetName             // name the pet
	stepTrust
)

type onboarding struct {
	step    onboardStep
	themes  []theme.Theme
	themeIx int

	petIx   int    // index into pets.All()
	petName string // user-typed nickname (input buffer)

	trustChoice int
	cwd         string
}

func newOnboarding(cwd string) *onboarding {
	return &onboarding{
		step:    stepTheme,
		themes:  theme.All(),
		themeIx: 0,
		petIx:   0,
		petName: pets.All()[0].Name,
		cwd:     cwd,
	}
}

func (o *onboarding) currentTheme() theme.Theme { return o.themes[o.themeIx] }
func (o *onboarding) currentPet() pets.Pet      { return pets.All()[o.petIx] }

// handleKey advances the onboarding state machine.
// Returns (done, trusted, pickedTheme, pickedPetName, pickedPetSpecies).
func (o *onboarding) handleKey(key string) (done bool, trusted bool, picked theme.Theme, petName string, petSpecies string) {
	switch o.step {
	case stepTheme:
		switch key {
		case "left", "up", "shift+tab":
			o.themeIx = (o.themeIx - 1 + len(o.themes)) % len(o.themes)
		case "right", "down", "tab":
			o.themeIx = (o.themeIx + 1) % len(o.themes)
		case "enter":
			o.step = stepPet
		}

	case stepPet:
		allPets := pets.All()
		switch key {
		case "left", "up", "shift+tab":
			o.petIx = (o.petIx - 1 + len(allPets)) % len(allPets)
			o.petName = allPets[o.petIx].Name
		case "right", "down", "tab":
			o.petIx = (o.petIx + 1) % len(allPets)
			o.petName = allPets[o.petIx].Name
		case "enter":
			o.step = stepPetName
		case "esc":
			o.step = stepTheme
		}

	case stepPetName:
		switch key {
		case "enter":
			if strings.TrimSpace(o.petName) == "" {
				o.petName = o.currentPet().Name
			}
			o.step = stepTrust
		case "esc":
			o.step = stepPet
		case "backspace":
			if len(o.petName) > 0 {
				o.petName = o.petName[:len(o.petName)-1]
			}
		default:
			// Accept printable single chars
			if len(key) == 1 && key[0] >= 32 && key[0] < 127 {
				o.petName += key
			}
		}

	case stepTrust:
		switch key {
		case "left", "right", "tab", "shift+tab":
			o.trustChoice = 1 - o.trustChoice
		case "y", "Y":
			o.trustChoice = 0
			return true, true, o.currentTheme(), o.petName, o.currentPet().Name
		case "n", "N":
			o.trustChoice = 1
			return true, false, o.currentTheme(), o.petName, o.currentPet().Name
		case "enter":
			return true, o.trustChoice == 0, o.currentTheme(), o.petName, o.currentPet().Name
		case "esc":
			o.step = stepPetName
		}
	}
	return false, false, theme.Theme{}, "", ""
}

func (o *onboarding) view(width, height int) string {
	t := o.currentTheme()
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Accent).
		Padding(1, 2).
		Width(width - 4)

	title := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	muted := lipgloss.NewStyle().Foreground(t.Muted)
	body := lipgloss.NewStyle().Foreground(t.Text)

	var inner string
	switch o.step {
	case stepTheme:
		header := title.Render("step 1 of 4 — pick a theme")
		var swatches []string
		for i, th := range o.themes {
			label := th.Name
			if i == o.themeIx {
				label = lipgloss.NewStyle().Foreground(th.Accent).Bold(true).Render("▸ " + label)
			} else {
				label = body.Render("  " + label)
			}
			swatches = append(swatches, label+"  "+swatch(th))
		}
		list := strings.Join(swatches, "\n")
		preview := previewBlock(t, width-12)
		hint := muted.Render("← → to browse · enter to confirm")
		inner = lipgloss.JoinVertical(lipgloss.Left, header, "", list, "", preview, "", hint)

	case stepPet:
		header := title.Render("step 2 of 4 — pick a pet")
		allPets := pets.All()
		p := allPets[o.petIx]
		petArt := lipgloss.NewStyle().Foreground(t.Accent).Render(p.Art)
		petName := lipgloss.NewStyle().Foreground(t.Muted).Italic(true).Render(p.Name)
		petBlock := lipgloss.JoinVertical(lipgloss.Center, petArt, petName)

		var labels []string
		for i, pp := range allPets {
			if i == o.petIx {
				labels = append(labels, lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("▸ "+pp.Name))
			} else {
				labels = append(labels, body.Render("  "+pp.Name))
			}
		}
		list := strings.Join(labels, "\n")
		hint := muted.Render("← → to browse · enter to confirm · esc back")
		inner = lipgloss.JoinVertical(lipgloss.Left,
			header, "",
			lipgloss.JoinHorizontal(lipgloss.Top, petBlock, "    ", list),
			"", hint)

	case stepPetName:
		header := title.Render("step 3 of 4 — name your pet")
		p := o.currentPet()
		petArt := lipgloss.NewStyle().Foreground(t.Accent).Render(p.Art)
		cursor := lipgloss.NewStyle().Foreground(t.Accent).Render("▌")
		nameDisplay := lipgloss.NewStyle().Foreground(t.Text).Bold(true).Render(o.petName) + cursor
		hint := muted.Render("type a name · enter to confirm · esc back")
		inner = lipgloss.JoinVertical(lipgloss.Left,
			header, "",
			lipgloss.NewStyle().Align(lipgloss.Center).Width(width-12).Render(petArt),
			"",
			body.Render("name: ")+nameDisplay,
			"", hint)

	case stepTrust:
		header := title.Render("step 4 of 4 — trust this directory?")
		path := lipgloss.NewStyle().Foreground(t.Accent).Render(o.cwd)
		explain := body.Render(
			"trusting a directory lets the agent read, write, and run\n" +
				"tools against files here. you can change this later by\n" +
				"editing .agent_settings/settings.json.")

		opts := []string{"trust this directory", "don't trust (read-only mode)"}
		var rendered []string
		for i, s := range opts {
			if i == o.trustChoice {
				rendered = append(rendered,
					lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("▸ "+s))
			} else {
				rendered = append(rendered, body.Render("  "+s))
			}
		}
		hint := muted.Render("y/n or ← → · enter to confirm · esc to go back")
		inner = lipgloss.JoinVertical(lipgloss.Left,
			header, "", path, "", explain, "", strings.Join(rendered, "\n"), "", hint)
	}

	return box.Render(inner)
}

func swatch(t theme.Theme) string {
	cells := []lipgloss.Color{t.Accent, t.UserFg, t.AsstFg, t.ToolFg, t.Border}
	var b strings.Builder
	for _, c := range cells {
		b.WriteString(lipgloss.NewStyle().Foreground(c).Render("██"))
	}
	return b.String()
}

func previewBlock(t theme.Theme, width int) string {
	if width < 30 {
		width = 30
	}
	user := lipgloss.NewStyle().Foreground(t.UserFg).Bold(true).Render("you: ")
	asst := lipgloss.NewStyle().Foreground(t.AsstFg).Bold(true).Render("ai:  ")
	tool := lipgloss.NewStyle().Foreground(t.ToolFg).Bold(true).Render("tool ")
	body := lipgloss.NewStyle().Foreground(t.Text)
	muted := lipgloss.NewStyle().Foreground(t.Muted).Italic(true)

	lines := []string{
		user + body.Render("hey, what does this code do?"),
		asst + body.Render("it greets the world."),
		muted.Render("(thinking: the function is straightforward)"),
		tool + body.Render("read(\"main.go\") → 42 lines"),
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(0, 1).
		Width(width).
		Render(fmt.Sprintf("preview\n\n%s", strings.Join(lines, "\n")))
}
