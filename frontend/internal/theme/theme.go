// Package theme defines the color palettes the TUI can render with.
//
// A Theme is just a bag of lipgloss colors. Bubbles read from the active
// theme rather than hardcoding ANSI codes, so swapping themes is a single
// pointer change in the root model.
package theme

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Name string

	// Foregrounds.
	Text    lipgloss.Color
	Muted   lipgloss.Color
	Accent  lipgloss.Color
	UserFg  lipgloss.Color
	AsstFg  lipgloss.Color
	ToolFg  lipgloss.Color
	ErrorFg lipgloss.Color
	WarnFg  lipgloss.Color

	// Borders.
	Border        lipgloss.Color
	BorderPending lipgloss.Color
	BorderMuted   lipgloss.Color

	// Backgrounds (optional — empty string means no fill).
	Bg lipgloss.Color
}

var (
	Dark = Theme{
		Name:          "dark",
		Text:          "252",
		Muted:         "244",
		Accent:        "208",
		UserFg:        "39",
		AsstFg:        "213",
		ToolFg:        "36",
		ErrorFg:       "196",
		WarnFg:        "214",
		Border:        "63",
		BorderPending: "214",
		BorderMuted:   "244",
	}

	Light = Theme{
		Name:          "light",
		Text:          "236",
		Muted:         "242",
		Accent:        "27",
		UserFg:        "21",
		AsstFg:        "127",
		ToolFg:        "29",
		ErrorFg:       "124",
		WarnFg:        "130",
		Border:        "27",
		BorderPending: "130",
		BorderMuted:   "248",
	}

	Dracula = Theme{
		Name:          "dracula",
		Text:          "253",
		Muted:         "103",
		Accent:        "212",
		UserFg:        "117",
		AsstFg:        "212",
		ToolFg:        "84",
		ErrorFg:       "203",
		WarnFg:        "215",
		Border:        "141",
		BorderPending: "215",
		BorderMuted:   "103",
	}

	Solarized = Theme{
		Name:          "solarized",
		Text:          "230",
		Muted:         "245",
		Accent:        "136",
		UserFg:        "33",
		AsstFg:        "125",
		ToolFg:        "37",
		ErrorFg:       "160",
		WarnFg:        "136",
		Border:        "37",
		BorderPending: "136",
		BorderMuted:   "240",
	}

	Monochrome = Theme{
		Name:          "monochrome",
		Text:          "255",
		Muted:         "245",
		Accent:        "250",
		UserFg:        "255",
		AsstFg:        "250",
		ToolFg:        "248",
		ErrorFg:       "245",
		WarnFg:        "250",
		Border:        "250",
		BorderPending: "245",
		BorderMuted:   "240",
	}
)

// All returns the registered themes in display order.
func All() []Theme {
	return []Theme{Dark, Light, Dracula, Solarized, Monochrome}
}

// ByName looks up a theme. Falls back to Dark if name is unknown.
func ByName(name string) Theme {
	for _, t := range All() {
		if t.Name == name {
			return t
		}
	}
	return Dark
}
