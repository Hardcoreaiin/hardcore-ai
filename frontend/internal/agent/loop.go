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

// Run drives the THINK/CALL loop until the model emits a plain answer (no
// CALL line) or the step limit is hit. Events are emitted on the returned
// channel and the channel is closed when the loop ends.
func (l *Loop) Run(ctx context.Context, userPrompt string) <-chan Event {
	out := make(chan Event, 32)

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

		msgs := []llm.Message{
			{Role: llm.RoleSystem, Content: l.sysp},
			{Role: llm.RoleUser, Content: userPrompt},
		}

		for step := 0; step < l.cfg.MaxToolSteps; step++ {
			lines, err := l.llm.Stream(ctx, msgs)
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
					if !emit(LineEvent{Text: line.Text}) {
						return
					}
				default:
					if !emit(LineEvent{Text: line.Text}) {
						return
					}
				}
			}

			assistantRaw := rawBuf.String()

			if pendingCall == nil {
				emit(DoneEvent{})
				return
			}

			pc := *pendingCall
			if pc.Err != nil {
				resultStr := "ERROR: " + pc.Err.Error()
				emit(ToolResultEvent{Name: pc.FuncName, Result: resultStr})
				msgs = append(msgs,
					llm.Message{Role: llm.RoleAssistant, Content: assistantRaw},
					llm.Message{Role: llm.RoleUser, Content: "Tool result: " + resultStr},
				)
				continue
			}

			tool, ok := l.reg.Get(pc.FuncName)
			if !ok {
				resultStr := fmt.Sprintf("ERROR: unknown tool %q", pc.FuncName)
				emit(ToolResultEvent{Name: pc.FuncName, Result: resultStr})
				msgs = append(msgs,
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

			msgs = append(msgs,
				llm.Message{Role: llm.RoleAssistant, Content: assistantRaw},
				llm.Message{Role: llm.RoleUser, Content: "Tool result: " + resultStr},
			)
		}

		emit(StepLimitEvent{Limit: l.cfg.MaxToolSteps})
	}()

	return out
}
