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

func (LineEvent) isEvent()       {}
func (ThinkEvent) isEvent()      {}
func (ToolStartEvent) isEvent()  {}
func (ToolResultEvent) isEvent() {}
func (ArtifactEvent) isEvent()   {}
func (StepLimitEvent) isEvent()  {}
func (DoneEvent) isEvent()       {}
func (ErrorEvent) isEvent()      {}
