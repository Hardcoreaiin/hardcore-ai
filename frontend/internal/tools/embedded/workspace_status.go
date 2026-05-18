package embedded

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

type workspaceStatusArgs struct{}

func RegisterWorkspaceStatus(r *tools.Registry) {
	tools.Register(r, "workspace_status",
		"Show the current project directory and list all existing projects in the workspace root. Use this at the start of a conversation to check what already exists.",
		func(_ context.Context, _ workspaceStatusArgs) (string, []tools.Artifact, error) {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", nil, err
			}
			root := filepath.Join(home, ".hardcoreai", "workspace")
			if err := os.MkdirAll(root, 0o755); err != nil {
				return "", nil, err
			}
			entries, err := os.ReadDir(root)
			if err != nil {
				return "", nil, err
			}
			var projects []string
			for _, e := range entries {
				if e.IsDir() {
					projects = append(projects, e.Name())
				}
			}
			current := CurrentProjectName()
			if len(projects) == 0 {
				return fmt.Sprintf("current project: %s  (no projects yet)", current), nil, nil
			}
			return fmt.Sprintf("current project: %s\nall projects: %s", current, strings.Join(projects, ", ")), nil, nil
		})
}
