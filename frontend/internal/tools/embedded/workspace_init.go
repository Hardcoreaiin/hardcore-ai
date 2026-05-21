package embedded

import (
	"context"
	"fmt"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

type workspaceInitArgs struct {
	Name string `tool:"name" desc:"project directory: an absolute path, a ~-relative path, or a name/path relative to the current directory"`
}

func RegisterWorkspaceInit(r *tools.Registry) {
	tools.Register(r, "workspace_init",
		"Create or switch to a project directory and make it the active directory. The argument may be an absolute path (e.g. /home/user/proj), a ~-relative path, or a name relative to the current directory. The directory is created if missing. Returns the absolute project path.",
		func(_ context.Context, a workspaceInitArgs) (string, []tools.Artifact, error) {
			name := strings.TrimSpace(a.Name)
			if name == "" {
				return "", nil, fmt.Errorf("project name or path must not be empty")
			}
			dir, err := SetWorkspaceProject(name)
			if err != nil {
				return "", nil, err
			}
			return fmt.Sprintf("active project: %s", dir), []tools.Artifact{{Type: "project_dir", Payload: dir}}, nil
		})
}
