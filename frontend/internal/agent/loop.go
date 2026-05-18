package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/llm"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

const DefaultMaxToolSteps = 10

type Config struct {
	MaxToolSteps int
}

type Loop struct {
	llm  llm.Client
	reg  *tools.Registry
	cfg  Config
	sysp string
}

func New(client llm.Client, reg *tools.Registry, cfg Config) *Loop {
	if cfg.MaxToolSteps == 0 {
		cfg.MaxToolSteps = DefaultMaxToolSteps
	}
	return &Loop{
		llm:  client,
		reg:  reg,
		cfg:  cfg,
		sysp: BuildSystemPrompt(reg),
	}
}

// Run drives a single-shot conversation: system + user, run THINK/CALL loop
// until DoneEvent or step cap. Kept for backward compatibility and headless
// one-off scripts. For interactive use see NewSession.
func (l *Loop) Run(ctx context.Context, userPrompt string) <-chan Event {
	out := make(chan Event, 32)
	go func() {
		defer close(out)
		msgs := []llm.Message{
			{Role: llm.RoleSystem, Content: l.sysp},
			{Role: llm.RoleUser, Content: userPrompt},
		}
		emit := func(e Event) bool {
			select {
			case out <- e:
				return true
			case <-ctx.Done():
				return false
			}
		}
		l.runTurn(ctx, &msgs, emit)
	}()
	return out
}

// runTurn executes the THINK/CALL state machine against the given message
// history, appending assistant/tool turns to msgs in place. Returns when
// the model emits a final answer, hits the step cap, or errors out.
func (l *Loop) runTurn(ctx context.Context, msgs *[]llm.Message, emit func(Event) bool) {
	for step := 0; step < l.cfg.MaxToolSteps; step++ {
		lines, err := l.llm.Stream(ctx, *msgs)
		if err != nil {
			emit(ErrorEvent{Err: err})
			return
		}

		var rawBuf strings.Builder
		var pendingCall *ParsedLine

		for line := range lines {
			if line.Done {
				break
			}
			if rawBuf.Len() > 0 {
				rawBuf.WriteByte('\n')
			}
			rawBuf.WriteString(line.Text)

			pl := ParseLine(line.Text)
			switch pl.Kind {
			case LineThink:
				if !emit(ThinkEvent{Text: pl.Text}) {
					return
				}
			case LineCall:
				p := pl
				pendingCall = &p
				// Intentionally don't emit a LineEvent: CALL syntax is
				// internal and must never reach the chat view.
			default:
				if !emit(LineEvent{Text: line.Text}) {
					return
				}
			}
		}

		assistantRaw := rawBuf.String()

		if pendingCall == nil {
			*msgs = append(*msgs, llm.Message{Role: llm.RoleAssistant, Content: assistantRaw})
			emit(DoneEvent{})
			return
		}

		pc := *pendingCall
		if pc.Err != nil {
			resultStr := "ERROR: " + pc.Err.Error()
			emit(ToolResultEvent{Name: pc.FuncName, Result: resultStr})
			*msgs = append(*msgs,
				llm.Message{Role: llm.RoleAssistant, Content: assistantRaw},
				llm.Message{Role: llm.RoleUser, Content: "Tool result: " + resultStr},
			)
			continue
		}

		tool, ok := l.reg.Get(pc.FuncName)
		if !ok {
			resultStr := fmt.Sprintf("ERROR: unknown tool %q", pc.FuncName)
			emit(ToolResultEvent{Name: pc.FuncName, Result: resultStr})
			*msgs = append(*msgs,
				llm.Message{Role: llm.RoleAssistant, Content: assistantRaw},
				llm.Message{Role: llm.RoleUser, Content: "Tool result: " + resultStr},
			)
			continue
		}

		emit(ToolStartEvent{Name: pc.FuncName, Args: pc.Tokens})

		result, artifacts, terr := tool.Execute(ctx, pc.Tokens)
		var resultStr string
		if terr != nil {
			resultStr = "ERROR: " + terr.Error()
		} else {
			resultStr = result
		}
		emit(ToolResultEvent{Name: pc.FuncName, Result: resultStr})
		for _, a := range artifacts {
			if !emit(ArtifactEvent{From: pc.FuncName, Artifact: a}) {
				return
			}
		}

		*msgs = append(*msgs,
			llm.Message{Role: llm.RoleAssistant, Content: assistantRaw},
			llm.Message{Role: llm.RoleUser, Content: "Tool result: " + resultStr},
		)
	}

	emit(StepLimitEvent{Limit: l.cfg.MaxToolSteps})
}

// Session is a multi-turn conversation. Message history accumulates across
// Send calls so the model has full context of prior turns.
type Session struct {
	loop *Loop
	msgs []llm.Message
}

func (l *Loop) NewSession() *Session {
	return &Session{
		loop: l,
		msgs: []llm.Message{{Role: llm.RoleSystem, Content: l.sysp}},
	}
}

// Send adds a user message and runs one full THINK/CALL turn. Events are
// emitted on the returned channel, terminating with TurnEndEvent and then
// channel close. Safe to call sequentially; do not call concurrently.
func (s *Session) Send(ctx context.Context, userPrompt string) <-chan Event {
	out := make(chan Event, 32)
	s.msgs = append(s.msgs, llm.Message{Role: llm.RoleUser, Content: userPrompt})

	go func() {
		defer close(out)
		emit := func(e Event) bool {
			select {
			case out <- e:
				return true
			case <-ctx.Done():
				return false
			}
		}
		emit(UserMessageEvent{Text: userPrompt})
		s.loop.runTurn(ctx, &s.msgs, emit)
		emit(TurnEndEvent{})
	}()
	return out
}

// Reset clears all history except the system prompt.
func (s *Session) Reset() {
	s.msgs = s.msgs[:1]
}

// SwapClient replaces the underlying LLM client for future turns.
func (l *Loop) SwapClient(c llm.Client) { l.llm = c }

func (s *Session) SwapClient(c llm.Client) { s.loop.SwapClient(c) }
