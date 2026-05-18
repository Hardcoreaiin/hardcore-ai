// Package bubbles holds the minimal bubble widgets for the TUI prototype.
//
// A Bubble is anything that can render a titled, bordered block given a
// width. Each bubble decides for itself whether a given event is relevant
// (via Handle); the TUI just fans every event to every bubble.
package bubbles

import "github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"

type Bubble interface {
	Title() string
	Handle(ev agent.Event) bool
	View(width int) string
}
