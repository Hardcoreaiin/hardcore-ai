package llm

import "context"

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Message struct {
	Role    Role
	Content string
}

// Line is one buffered line from the model's output stream.
// Done=true marks the final sentinel; if there was a trailing partial line
// when the stream ended, it is delivered as a Line with Done=false followed
// by a Line with Done=true.
// Token carries the raw sub-line chunk as it arrived; non-empty only on
// intermediate (Done=false) deliveries where Text is not yet a full line.
type Line struct {
	Text  string
	Token string // raw chunk, may not end on a newline boundary
	Done  bool
}

type Client interface {
	Stream(ctx context.Context, msgs []Message) (<-chan Line, error)
}
