package embedded

import (
	"context"
	"fmt"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

type cdArgs struct {
	Path string `tool:"path" desc:"directory to switch into: absolute, ~-relative, or relative to the current directory (.. is allowed)"`
}

func RegisterCD(r *tools.Registry) {
	tools.Register(r, "cd",
		"Change the active directory. Accepts an absolute path, a ~-relative path, or a path relative to the current directory (.. allowed). The directory must already exist — use workspace_init to create one. Returns the new active directory.",
		func(_ context.Context, a cdArgs) (string, []tools.Artifact, error) {
			path := strings.TrimSpace(a.Path)
			if path == "" {
				return "", nil, fmt.Errorf("path must not be empty")
			}
			dir, err := ChangeDir(path)
			if err != nil {
				return "", nil, err
			}
			return fmt.Sprintf("active directory: %s", dir), []tools.Artifact{{Type: "project_dir", Payload: dir}}, nil
		})
}
