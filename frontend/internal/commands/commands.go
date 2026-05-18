// Package commands is the registry for slash commands the TUI exposes
// in the input line. Each command has a name, a description, an optional
// dynamic completer for its first argument, and a Run function that the
// TUI invokes when the user submits "/<name> [args...]".
//
// The registry is intentionally decoupled from any TUI types — it
// returns a Result struct describing what changed, and the caller
// applies that result to its model.
package commands

import (
	"fmt"
	"sort"
	"strings"
)

// Result is what a command returns to the caller after running.
type Result struct {
	// Message is rendered as a system line in the chat. Empty = silent.
	Message string

	// State transitions the caller should perform. Zero values mean
	// "no change". The caller persists settings after applying.
	NewTheme   string
	NewPet     string
	NewPetName string
	NewTrusted *bool

	// NewProvider asks the caller to rebuild the LLM client for the given provider.
	NewProvider string

	// ResetSession asks the caller to start a fresh agent session.
	ResetSession bool
	// ClearVisual asks the caller to drop existing bubbles.
	ClearVisual bool
	// Quit asks the caller to exit the program.
	Quit bool

	// Err non-nil signals the user-facing error to render.
	Err error
}

// Command is one slash command.
type Command struct {
	Name        string
	Description string
	// ArgValues returns the set of completions for the first argument.
	// Nil means the command takes no completable args.
	ArgValues func() []string
	// Run executes the command. args is the rest of the input after the name.
	Run func(args []string) Result
}

// Registry holds the commands in a stable display order.
type Registry struct {
	cmds []*Command
}

func New() *Registry { return &Registry{} }

func (r *Registry) Register(c *Command) { r.cmds = append(r.cmds, c) }

// All returns commands sorted by name.
func (r *Registry) All() []*Command {
	out := make([]*Command, len(r.cmds))
	copy(out, r.cmds)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Lookup returns the command with the given name, or nil.
func (r *Registry) Lookup(name string) *Command {
	for _, c := range r.cmds {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// IsCommand reports whether input looks like a slash command.
func IsCommand(input string) bool {
	s := strings.TrimSpace(input)
	return strings.HasPrefix(s, "/") && len(s) > 1
}

// Parse splits "/name a b c" into (name, ["a","b","c"]).
func Parse(input string) (string, []string) {
	s := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(input), "/"))
	if s == "" {
		return "", nil
	}
	parts := strings.Fields(s)
	return parts[0], parts[1:]
}

// Run dispatches input to the matching command. Returns an error result
// if the command is unknown.
func (r *Registry) Run(input string) Result {
	name, args := Parse(input)
	c := r.Lookup(name)
	if c == nil {
		return Result{Err: fmt.Errorf("unknown command: /%s (try /help)", name)}
	}
	return c.Run(args)
}

// Suggestion describes one autocomplete row.
type Suggestion struct {
	// Value is the full text to insert into the input (without trailing space).
	Value string
	// Label is what to show in the popup ("/theme dracula").
	Label string
	// Detail is a one-line description (right-aligned in the popup).
	Detail string
}

// Suggest returns autocomplete options for the current input. When the
// input is just "/" or "/foo", it suggests command names. When it's
// "/foo bar", it asks the command for arg completions.
func (r *Registry) Suggest(input string) []Suggestion {
	trimmed := strings.TrimLeft(input, " ")
	if !strings.HasPrefix(trimmed, "/") {
		return nil
	}
	body := strings.TrimPrefix(trimmed, "/")

	// Decide whether we're completing the command name or its first arg.
	// We're on the name if there's no space yet in `body`.
	if !strings.Contains(body, " ") {
		prefix := strings.ToLower(body)
		var out []Suggestion
		for _, c := range r.All() {
			if strings.HasPrefix(c.Name, prefix) {
				out = append(out, Suggestion{
					Value:  "/" + c.Name,
					Label:  "/" + c.Name,
					Detail: c.Description,
				})
			}
		}
		return out
	}

	// Completing first argument.
	parts := strings.SplitN(body, " ", 2)
	name := parts[0]
	argPrefix := ""
	if len(parts) == 2 {
		argPrefix = strings.ToLower(parts[1])
	}
	c := r.Lookup(name)
	if c == nil || c.ArgValues == nil {
		return nil
	}
	var out []Suggestion
	for _, v := range c.ArgValues() {
		if strings.HasPrefix(strings.ToLower(v), argPrefix) {
			out = append(out, Suggestion{
				Value:  "/" + name + " " + v,
				Label:  v,
				Detail: "argument for /" + name,
			})
		}
	}
	return out
}
