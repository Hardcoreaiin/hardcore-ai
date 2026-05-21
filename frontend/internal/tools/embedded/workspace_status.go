package embedded

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

type workspaceStatusArgs struct{}

func RegisterWorkspaceStatus(r *tools.Registry) {
	tools.Register(r, "workspace_status",
		"Show the current active directory and list its immediate subdirectories. Use this at the start of a conversation to see where you are and what project directories exist.",
		func(_ context.Context, _ workspaceStatusArgs) (string, []tools.Artifact, error) {
			dir, err := WorkspaceDir()
			if err != nil {
				return "", nil, err
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				return "", nil, err
			}
			var subdirs []string
			for _, e := range entries {
				if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
					subdirs = append(subdirs, e.Name())
				}
			}
			if len(subdirs) == 0 {
				return fmt.Sprintf("current directory: %s\n(no subdirectories)", dir), nil, nil
			}
			return fmt.Sprintf("current directory: %s\nsubdirectories: %s", dir, strings.Join(subdirs, ", ")), nil, nil
		})
}
