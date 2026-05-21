package embedded

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

// ConfirmFunc is asked to approve a shell command before it runs. It receives
// the command string and the directory it will run in, and returns true to
// allow execution. The TUI installs one of these so the user gets a per-command
// accept/reject prompt; when none is installed, commands are denied.
type ConfirmFunc func(command, dir string) bool

var (
	confirmMu sync.RWMutex
	confirmFn ConfirmFunc
)

// SetBashConfirmFunc installs the confirmation gate for the bash tool.
func SetBashConfirmFunc(fn ConfirmFunc) {
	confirmMu.Lock()
	confirmFn = fn
	confirmMu.Unlock()
}

func getConfirmFn() ConfirmFunc {
	confirmMu.RLock()
	defer confirmMu.RUnlock()
	return confirmFn
}

type bashArgs struct {
	Command string `tool:"command" desc:"the shell command to run in the current directory"`
}

func RegisterBash(r *tools.Registry) {
	tools.Register(r, "bash",
		"Run a shell command in the current directory. The user is asked to approve each command before it runs — a rejected command returns an error. Use this for git, ls, mkdir, running scripts, package managers, and any task not covered by a dedicated tool.",
		func(ctx context.Context, a bashArgs) (string, []tools.Artifact, error) {
			cmdStr := strings.TrimSpace(a.Command)
			if cmdStr == "" {
				return "", nil, fmt.Errorf("command must not be empty")
			}
			dir, err := WorkspaceDir()
			if err != nil {
				return "", nil, err
			}

			fn := getConfirmFn()
			if fn == nil {
				return "", nil, fmt.Errorf("bash command rejected: no confirmation handler available")
			}
			if !fn(cmdStr, dir) {
				return "", nil, fmt.Errorf("bash command rejected by user")
			}

			runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()

			cmd := exec.CommandContext(runCtx, "sh", "-c", cmdStr)
			cmd.Dir = dir
			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out
			runErr := cmd.Run()
			output := strings.TrimSpace(out.String())

			if runErr != nil {
				if output == "" {
					output = runErr.Error()
				}
				return fmt.Sprintf("ERROR: command exited with error: %v\n%s", runErr, output), nil, nil
			}
			if output == "" {
				output = "(no output)"
			}
			return output, nil, nil
		})
}
