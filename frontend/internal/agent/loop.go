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
		_ = l.runTurn(ctx, &msgs, emit)
	}()
	return out
}

// runTurn executes the THINK/CALL state machine against the given message
// history, appending assistant/tool turns to msgs in place. Returns true if
// it already emitted TurnEndEvent (ASK suspension), false otherwise.
func (l *Loop) runTurn(ctx context.Context, msgs *[]llm.Message, emit func(Event) bool) (emittedEnd bool) {
	for step := 0; step < l.cfg.MaxToolSteps; step++ {
		lines, err := l.llm.Stream(ctx, *msgs)
		if err != nil {
			emit(ErrorEvent{Err: err})
			return
		}

		var rawBuf strings.Builder
		var pendingCall *ParsedLine
		var pendingAsk *AskEvent
		var pendingLines []LineEvent

		for line := range lines {
			if line.Done {
				break
			}
			// Raw token (sub-line chunk) — emit for the stream ticker, skip parsing.
			if line.Token != "" && line.Text == "" {
				emit(TokenEvent{Text: line.Token})
				continue
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
			case LineTodo:
				items := make([]string, len(pl.Tokens))
				for i, t := range pl.Tokens {
					items[i], _ = t.(string)
				}
				if !emit(TodoEvent{Items: items}) {
					return
				}
			case LineAsk:
				parts := make([]string, len(pl.Tokens))
				for i, t := range pl.Tokens {
					parts[i], _ = t.(string)
				}
				question := ""
				options := parts
				if len(parts) > 0 {
					question = parts[0]
					options = parts[1:]
				}
				ev := AskEvent{Question: question, Options: options}
				pendingAsk = &ev
				// Stop processing this stream immediately — the turn is
				// suspended until the user answers. Drain the channel so
				// the HTTP goroutine can exit cleanly.
				for range lines {
				}
				break
			default:
				pendingLines = append(pendingLines, LineEvent{Text: line.Text})
			}
			if pendingAsk != nil {
				break
			}
		}

		assistantRaw := rawBuf.String()

		// ASK: suspend the turn. Append the partial assistant message so
		// history is intact, emit the event, then return — the next call
		// to session.Send (with the user's answer) continues the work.
		if pendingAsk != nil {
			*msgs = append(*msgs, llm.Message{Role: llm.RoleAssistant, Content: assistantRaw})
			emit(*pendingAsk)
			emit(TurnEndEvent{})
			return true
		}

		// Scan full buffer for code fences (``` ... ```) and emit them.
		emitCodeFences(assistantRaw, emit)

		// If per-line parsing missed the CALL (e.g. multi-line argument like
		// file_write content), scan the full buffer now.
		if pendingCall == nil {
			if pl := ParseFullText(assistantRaw); pl.Kind == LineCall {
				pendingCall = &pl
			}
		}

		if pendingCall == nil {
			for _, ev := range pendingLines {
				if !emit(ev) {
					return
				}
			}
			// If the model only emitted a TODO (plan) with no tool call, prod it
			// to start executing instead of stopping.
			if strings.Contains(strings.ToUpper(assistantRaw), "TODO:") {
				*msgs = append(*msgs,
					llm.Message{Role: llm.RoleAssistant, Content: assistantRaw},
					llm.Message{Role: llm.RoleUser, Content: "Good plan. Now execute it — start with the first step immediately using a CALL."},
				)
				continue
			}
			*msgs = append(*msgs, llm.Message{Role: llm.RoleAssistant, Content: assistantRaw})
			emit(DoneEvent{})
			return false
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
	return false
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
		if alreadyEnded := s.loop.runTurn(ctx, &s.msgs, emit); !alreadyEnded {
			emit(TurnEndEvent{})
		}
	}()
	return out
}

// Reset clears all history except the system prompt.
func (s *Session) Reset() {
	s.msgs = s.msgs[:1]
}

// emitCodeFences scans text for ``` fenced code blocks and emits a
// CodeFenceEvent for each one found.
func emitCodeFences(text string, emit func(Event) bool) {
	for {
		start := strings.Index(text, "```")
		if start == -1 {
			return
		}
		rest := text[start+3:]
		// Extract optional language identifier (up to newline).
		nl := strings.IndexByte(rest, '\n')
		if nl == -1 {
			return
		}
		lang := strings.TrimSpace(rest[:nl])
		body := rest[nl+1:]
		end := strings.Index(body, "```")
		if end == -1 {
			return
		}
		content := strings.TrimSpace(body[:end])
		if content != "" {
			emit(CodeFenceEvent{Lang: lang, Content: content})
		}
		text = body[end+3:]
	}
}

// SwapClient replaces the underlying LLM client for future turns.
func (l *Loop) SwapClient(c llm.Client) { l.llm = c }

func (s *Session) SwapClient(c llm.Client) { s.loop.SwapClient(c) }
