package embedded

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

type fileReadArgs struct {
	Path string `tool:"path" desc:"relative path inside the project to read"`
}

func RegisterFileRead(r *tools.Registry) {
	tools.Register(r, "file_read",
		"Read a file from the current project. Returns full file content.",
		func(_ context.Context, a fileReadArgs) (string, []tools.Artifact, error) {
			if strings.Contains(a.Path, "..") {
				return "", nil, fmt.Errorf("path must not contain '..'")
			}
			root, err := WorkspaceDir()
			if err != nil {
				return "", nil, err
			}
			data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(a.Path)))
			if err != nil {
				return "", nil, err
			}
			return string(data), nil, nil
		})
}
