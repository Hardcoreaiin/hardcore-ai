package embedded

import (
	"context"
	"os"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

type fileReadArgs struct {
	Path string `tool:"path" desc:"file to read: absolute, ~-relative, or relative to the current directory"`
}

func RegisterFileRead(r *tools.Registry) {
	tools.Register(r, "file_read",
		"Read a file. The path may be absolute, ~-relative, or relative to the current directory. Returns full file content.",
		func(_ context.Context, a fileReadArgs) (string, []tools.Artifact, error) {
			full, err := ResolvePath(a.Path)
			if err != nil {
				return "", nil, err
			}
			data, err := os.ReadFile(full)
			if err != nil {
				return "", nil, err
			}
			return string(data), nil, nil
		})
}
