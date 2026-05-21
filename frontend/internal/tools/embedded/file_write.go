package embedded

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

type fileWriteArgs struct {
	Path    string `tool:"path"    desc:"file to write: absolute, ~-relative, or relative to the current directory, e.g. main.c or src/gpio.h"`
	Content string `tool:"content" desc:"file body — supplied as a fenced code block on the line(s) right after the CALL, not inline"`
}

func RegisterFileWrite(r *tools.Registry) {
	tools.Register(r, "file_write",
		"Write (or overwrite) a file. Call as: CALL file_write(\"path\") then put the COMPLETE file body in a fenced ``` code block on the following lines. Parent directories are created automatically.",
		func(_ context.Context, a fileWriteArgs) (string, []tools.Artifact, error) {
			if a.Content == "" {
				return "", nil, fmt.Errorf("no file content: put the file body in a fenced ``` block on the line after CALL file_write(%q)", a.Path)
			}
			full, err := ResolvePath(a.Path)
			if err != nil {
				return "", nil, err
			}
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				return "", nil, err
			}
			if err := os.WriteFile(full, []byte(a.Content), 0o644); err != nil {
				return "", nil, err
			}
			return fmt.Sprintf("wrote %d bytes → %s", len(a.Content), full), nil, nil
		})
}
