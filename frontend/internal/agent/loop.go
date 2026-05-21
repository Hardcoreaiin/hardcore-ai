package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/llm"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

const DefaultMaxToolSteps = 10

// historyResultMaxLen is the max bytes of a generic tool result kept in
// message history. The TUI already shows the full output; the LLM only needs
// enough to understand what happened. Large build logs would otherwise
// balloon every subsequent LLM request and spike RAM.
const historyResultMaxLen = 2000

// fileReadMaxLines caps a file_read result by LINES, not bytes. The model must
// see a file in full to edit it — file_edit anchors on exact lines, and a
// byte cap would slice a line in half and break every anchor. Truncating on a
// line boundary keeps anchors intact. 2000 lines covers any realistic
// firmware file while still bounding context/RAM.
const fileReadMaxLines = 2000

// historyRawMaxLen caps how many bytes of raw assistant text are stored back
// into history per step (includes THINK lines which can be very long).
const historyRawMaxLen = 4000

// maxHistoryMessages is the total number of messages (excluding the system
// prompt) kept in history. Older message pairs are dropped when exceeded so
// the context window and RAM stay bounded across long sessions.
const maxHistoryMessages = 40

// maxRawBufBytes caps the live per-step assistant buffer. A model that streams
// without stopping (or emits a pathologically long response) would otherwise
// grow this builder unbounded and exhaust RAM. Once exceeded, further line
// text is dropped from the buffer — the step still completes on the prefix.
const maxRawBufBytes = 256 * 1024

type Config struct {
	MaxToolSteps int
}

// Retriever supplies background reference context for a user prompt. The agent
// queries it automatically at the start of each turn — the model never invokes
// it and never sees a "nothing found" result. Implementations must return
// ok=false (not an error, not an empty string with ok=true) when there is no
// relevant context, so the loop can simply inject nothing.
type Retriever interface {
	Retrieve(ctx context.Context, query string) (context string, ok bool)
}

type Loop struct {
	llm  llm.Client
	reg  *tools.Registry
	cfg  Config
	sysp string
	ret  Retriever
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

// SetRetriever installs the background reference retriever. Safe to call before
// any turn runs; pass nil to disable automatic retrieval.
func (l *Loop) SetRetriever(r Retriever) { l.ret = r }

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

// trimHistory keeps the message slice bounded. It preserves the system prompt
// (always msgs[0]) and the most recent maxHistoryMessages messages, dropping
// the oldest pairs first. Non-destructive when under the limit.
func trimHistory(msgs *[]llm.Message) {
	if len(*msgs) <= 1+maxHistoryMessages {
		return
	}
	// Keep msgs[0] (system) + the last maxHistoryMessages entries.
	tail := (*msgs)[len(*msgs)-maxHistoryMessages:]
	newMsgs := make([]llm.Message, 1, 1+len(tail))
	newMsgs[0] = (*msgs)[0] // system prompt
	newMsgs = append(newMsgs, tail...)
	*msgs = newMsgs
}

// truncResult trims a tool-result string to historyResultMaxLen so large build
// logs don't bloat the message history that gets re-sent to the LLM every step.
func truncResult(s string) string {
	if len(s) <= historyResultMaxLen {
		return s
	}
	// Keep the tail — errors are usually at the end of build output.
	return "[…truncated…]\n" + s[len(s)-historyResultMaxLen:]
}

// truncFileRead trims a file_read result on a LINE boundary. Unlike
// truncResult it never slices mid-line, so the lines the model copies into a
// file_edit before-block stay byte-exact. Only files past fileReadMaxLines are
// touched — realistic firmware files pass through whole.
func truncFileRead(s string) string {
	if s == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= fileReadMaxLines {
		return s
	}
	kept := strings.Join(lines[:fileReadMaxLines], "\n")
	omitted := len(lines) - fileReadMaxLines
	return kept + fmt.Sprintf("\n[…%d more line(s) not shown — read a specific range if you need them…]", omitted)
}

// resultForHistory picks the right truncation for a tool result based on which
// tool produced it. file_read keeps whole source files (line-capped); every
// other tool gets the small byte cap suited to status/log output.
func resultForHistory(toolName, result string) string {
	if toolName == "file_read" {
		return truncFileRead(result)
	}
	return truncResult(result)
}

// truncRaw trims the raw assistant buffer stored in history to historyRawMaxLen.
func truncRaw(s string) string {
	if len(s) <= historyRawMaxLen {
		return s
	}
	return s[:historyRawMaxLen] + "\n[…truncated…]"
}

// runTurn executes the THINK/CALL state machine against the given message
// history, appending assistant/tool turns to msgs in place. Returns true if
// it already emitted TurnEndEvent (ASK suspension), false otherwise.
func (l *Loop) runTurn(ctx context.Context, msgs *[]llm.Message, emit func(Event) bool) (emittedEnd bool) {
	// nudgedThisTurn ensures the "you forgot to CALL" prod fires at most once
	// per turn, so a model that refuses to call a tool can't spin forever.
	nudgedThisTurn := false
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
		var inFence bool // true while inside a ``` code fence
		var runaway bool // set when the model exceeds maxRawBufBytes

		for line := range lines {
			if line.Done {
				break
			}
			// Runaway guard: a model that never stops streaming would flood the
			// event channel and grow RAM unbounded. Once the buffer is full,
			// drain the rest of the stream silently and proceed on the prefix.
			if rawBuf.Len() >= maxRawBufBytes {
				runaway = true
				for range lines {
				}
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

			// A line inside a ``` fence is code content (a file body, or a
			// file_edit before/after block) — never a directive. Parsing it
			// would let firmware like `delay(500);` be misread as a tool CALL
			// and clobber the real pendingCall. Track the fence toggle here,
			// before classification, and skip parsing for fenced lines.
			if strings.HasPrefix(strings.TrimSpace(line.Text), "```") {
				inFence = !inFence
				continue
			}
			if inFence {
				continue
			}

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
				// Plain prose line (fenced lines were already filtered out
				// above). Fence content is rendered via CodeFenceEvent, not
				// as LineEvents, so the pet/chat view stays clean.
				pendingLines = append(pendingLines, LineEvent{Text: line.Text})
			}
			if pendingAsk != nil {
				break
			}
		}

		assistantRaw := rawBuf.String()

		// Runaway response: the model exceeded the buffer cap without ever
		// emitting a usable CALL/ASK boundary. Continuing would re-stream the
		// same runaway output every step. Surface an error and end the turn.
		if runaway && pendingCall == nil && pendingAsk == nil {
			emit(ErrorEvent{Err: fmt.Errorf("model response exceeded %d bytes without a tool call — stopping", maxRawBufBytes)})
			*msgs = append(*msgs, llm.Message{Role: llm.RoleAssistant, Content: truncRaw(assistantRaw)})
			return
		}

		// ASK: suspend the turn. Append the partial assistant message so
		// history is intact, emit the event, then return — the next call
		// to session.Send (with the user's answer) continues the work.
		if pendingAsk != nil {
			*msgs = append(*msgs, llm.Message{Role: llm.RoleAssistant, Content: assistantRaw})
			emit(*pendingAsk)
			emit(TurnEndEvent{})
			return true
		}

		// If per-line parsing missed the CALL (e.g. multi-line argument like
		// file_write content), scan the full buffer now.
		if pendingCall == nil {
			if pl := ParseFullText(assistantRaw); pl.Kind == LineCall {
				pendingCall = &pl
			}
		}

		// Scan full buffer for code fences (``` ... ```) and emit them as
		// snippet bubbles — UNLESS this response is a file_write or file_edit
		// call, whose fences are the tool's payload (file body / edit pairs),
		// not standalone snippets. file_write renders a diff via ToolStart;
		// file_edit renders one via its file_diff artifact.
		fenceIsToolPayload := pendingCall != nil &&
			(pendingCall.FuncName == "file_write" || pendingCall.FuncName == "file_edit")
		if !fenceIsToolPayload {
			emitCodeFences(assistantRaw, emit)
		}

		// Heredoc convention: file_write may be called with only a path, with
		// the file body supplied as a fenced ``` block right after the CALL.
		// This keeps source code out of the arg tokenizer entirely. Pair the
		// path-only call with the first fence body found in the response.
		if pendingCall != nil && pendingCall.FuncName == "file_write" && len(pendingCall.Tokens) == 1 {
			if body, ok := firstFenceBody(assistantRaw); ok {
				pendingCall.Tokens = append(pendingCall.Tokens, body)
			}
		}

		// file_edit("path") is followed by paired before/after ``` blocks.
		// Hand the whole post-CALL text to the tool as the edits argument;
		// editmatch.ParseEdits pulls the fence pairs out of it. This keeps the
		// before/after code (with quotes, parens, newlines) out of the tokenizer.
		if pendingCall != nil && pendingCall.FuncName == "file_edit" && len(pendingCall.Tokens) == 1 {
			if edits, ok := fencesAfterCall(assistantRaw, pendingCall.Text); ok {
				pendingCall.Tokens = append(pendingCall.Tokens, edits)
			}
		}

		if pendingCall == nil {
			upper := strings.ToUpper(assistantRaw)
			// If the model only emitted a TODO (plan) with no tool call, prod it
			// to start executing instead of stopping.
			if strings.Contains(upper, "TODO:") {
				*msgs = append(*msgs,
					llm.Message{Role: llm.RoleAssistant, Content: truncRaw(assistantRaw)},
					llm.Message{Role: llm.RoleUser, Content: "Good plan. Now execute it — start with the first step immediately using a CALL."},
				)
				continue
			}
			// The model signalled intent to act (THINK:) but emitted no CALL —
			// a common failure of weak/quantized models that "narrate" the
			// action instead of invoking it. Prod it once to actually call a
			// tool rather than silently ending the turn. nudgedThisTurn guards
			// against an infinite prod loop if the model keeps refusing.
			if strings.Contains(upper, "THINK:") && !nudgedThisTurn {
				nudgedThisTurn = true
				*msgs = append(*msgs,
					llm.Message{Role: llm.RoleAssistant, Content: truncRaw(assistantRaw)},
					llm.Message{Role: llm.RoleUser, Content: "You described what to do but did not emit a tool call. Respond with exactly one line in the form: CALL toolname(\"arg\"). Do not narrate — call the tool now."},
				)
				continue
			}
			for _, ev := range pendingLines {
				if !emit(ev) {
					return
				}
			}
			*msgs = append(*msgs, llm.Message{Role: llm.RoleAssistant, Content: truncRaw(assistantRaw)})
			emit(DoneEvent{})
			return false
		}

		pc := *pendingCall
		if pc.Err != nil {
			resultStr := "ERROR: " + pc.Err.Error()
			emit(ToolResultEvent{Name: pc.FuncName, Result: resultStr})
			*msgs = append(*msgs,
				llm.Message{Role: llm.RoleAssistant, Content: truncRaw(assistantRaw)},
				llm.Message{Role: llm.RoleUser, Content: "Tool result: " + truncResult(resultStr)},
			)
			trimHistory(msgs)
			continue
		}

		tool, ok := l.reg.Get(pc.FuncName)
		if !ok {
			resultStr := fmt.Sprintf("ERROR: unknown tool %q", pc.FuncName)
			emit(ToolResultEvent{Name: pc.FuncName, Result: resultStr})
			*msgs = append(*msgs,
				llm.Message{Role: llm.RoleAssistant, Content: truncRaw(assistantRaw)},
				llm.Message{Role: llm.RoleUser, Content: "Tool result: " + truncResult(resultStr)},
			)
			trimHistory(msgs)
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
			llm.Message{Role: llm.RoleAssistant, Content: truncRaw(assistantRaw)},
			llm.Message{Role: llm.RoleUser, Content: "Tool result: " + resultForHistory(pc.FuncName, resultStr)},
		)
		trimHistory(msgs)
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

		// Automatic reference retrieval: query the retriever for this prompt
		// and, if anything relevant comes back, inject it as a reference
		// message right before the model runs. The model never asks for this
		// and never learns when retrieval found nothing.
		if s.loop.ret != nil {
			if refCtx, ok := s.loop.ret.Retrieve(ctx, userPrompt); ok {
				s.msgs = append(s.msgs, llm.Message{
					Role:    llm.RoleUser,
					Content: "[Reference documentation — use if relevant, do not mention this block]\n" + refCtx,
				})
			}
		}

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

// ToolNames returns the names of all registered tools, for user-facing
// completion and direct tool invocation.
func (s *Session) ToolNames() []string {
	return s.loop.reg.Names()
}

// RunTool executes a registered tool directly (bypassing the LLM). It returns
// the tool's result string, any artifacts, and an error. Used by the TUI to
// let the user invoke tools without the agent.
func (s *Session) RunTool(ctx context.Context, name string, tokens []any) (string, []tools.Artifact, error) {
	tool, ok := s.loop.reg.Get(name)
	if !ok {
		return "", nil, fmt.Errorf("unknown tool %q", name)
	}
	return tool.Execute(ctx, tokens)
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

// firstFenceBody returns the raw body of the first ``` fenced block in text,
// used as the file content for a path-only file_write call. Unlike
// emitCodeFences it does NOT trim interior whitespace — source code is written
// verbatim — but it drops the single newline before the closing fence and
// guarantees a trailing newline so written files end cleanly.
func firstFenceBody(text string) (string, bool) {
	start := strings.Index(text, "```")
	if start == -1 {
		return "", false
	}
	rest := text[start+3:]
	nl := strings.IndexByte(rest, '\n')
	if nl == -1 {
		return "", false
	}
	body := rest[nl+1:]
	end := strings.Index(body, "```")
	if end == -1 {
		return "", false
	}
	content := body[:end]
	content = strings.TrimRight(content, "\n")
	if content == "" {
		return "", false
	}
	return content + "\n", true
}

// fencesAfterCall returns the slice of text that follows the file_edit CALL
// line, which holds the before/after ``` fence pairs. callText is the matched
// call expression (e.g. `file_edit("main.c")`); everything after its first
// occurrence is the edits payload. Returns false if no ``` block follows.
func fencesAfterCall(text, callText string) (string, bool) {
	idx := strings.Index(text, callText)
	if idx == -1 {
		return "", false
	}
	after := text[idx+len(callText):]
	if !strings.Contains(after, "```") {
		return "", false
	}
	return after, true
}

// SwapClient replaces the underlying LLM client for future turns.
func (l *Loop) SwapClient(c llm.Client) { l.llm = c }

func (s *Session) SwapClient(c llm.Client) { s.loop.SwapClient(c) }
