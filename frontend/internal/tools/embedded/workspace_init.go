package embedded

import (
	"context"
	"fmt"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

type workspaceInitArgs struct {
	Name string `tool:"name" desc:"project name (directory will be created under the workspace root)"`
}

func RegisterWorkspaceInit(r *tools.Registry) {
	tools.Register(r, "workspace_init",
		"Create or switch to a named project directory. Call this first when starting a new firmware project or when the user names a project. Returns the project path.",
		func(_ context.Context, a workspaceInitArgs) (string, []tools.Artifact, error) {
			name := strings.TrimSpace(a.Name)
			if name == "" {
				return "", nil, fmt.Errorf("project name must not be empty")
			}
			if strings.ContainsAny(name, "/\\..") {
				return "", nil, fmt.Errorf("project name must not contain path separators or '..'")
			}
			dir, err := SetWorkspaceProject(name)
			if err != nil {
				return "", nil, err
			}
			return fmt.Sprintf("project: %q  path: %s", name, dir), []tools.Artifact{{Type: "project_dir", Payload: dir}}, nil
		})
}
