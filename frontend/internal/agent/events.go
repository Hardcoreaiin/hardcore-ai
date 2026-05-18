package agent

import "github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"

type Event interface{ isEvent() }

type LineEvent struct{ Text string }
type ThinkEvent struct{ Text string }
type ToolStartEvent struct {
	Name string
	Args []any
}
type ToolResultEvent struct {
	Name   string
	Result string
}
type ArtifactEvent struct {
	From     string
	Artifact tools.Artifact
}
type StepLimitEvent struct{ Limit int }
type DoneEvent struct{}
type ErrorEvent struct{ Err error }

// TurnEndEvent is emitted once per call to Session.Send after the model
// finishes (whether via Done, StepLimit, or Error). The TUI uses it to
// know when input should be re-enabled.
type TurnEndEvent struct{}

// UserMessageEvent is emitted at the start of a turn so subscribers can
// render the user's own input in chat without coordinating with main.
type UserMessageEvent struct{ Text string }

// ToolCallBubbleEvent is emitted alongside ToolStartEvent so the TUI can
// create a dedicated per-call bubble immediately (before the result arrives).
// It carries no extra data — the TUI pairs it with ToolStartEvent by ordering.
type ToolCallBubbleEvent struct{}

// TodoEvent is emitted when the model outputs a TODO: list so the TUI can
// render a structured checklist bubble.
type TodoEvent struct {
	Items []string
}

// AskEvent is emitted when the model outputs an ASK: questionnaire. The TUI
// renders a choice bubble and feeds the user's answer back as the next message.
// Options never includes "Other" — the TUI always appends that automatically.
type AskEvent struct {
	Question string
	Options  []string
}

func (LineEvent) isEvent()            {}
func (ThinkEvent) isEvent()           {}
func (ToolStartEvent) isEvent()       {}
func (ToolResultEvent) isEvent()      {}
func (ArtifactEvent) isEvent()        {}
func (StepLimitEvent) isEvent()       {}
func (DoneEvent) isEvent()            {}
func (ErrorEvent) isEvent()           {}
func (TurnEndEvent) isEvent()         {}
func (UserMessageEvent) isEvent()     {}
func (TodoEvent) isEvent()            {}
func (AskEvent) isEvent()             {}
func (ToolCallBubbleEvent) isEvent()  {}
