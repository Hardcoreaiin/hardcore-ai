package embedded

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

type fileWriteArgs struct {
	Path    string `tool:"path"    desc:"relative path inside the project, e.g. main.c or src/gpio.h"`
	Content string `tool:"content" desc:"full file content to write"`
}

func RegisterFileWrite(r *tools.Registry) {
	tools.Register(r, "file_write",
		"Write (or overwrite) a file in the current project. Use relative paths like 'main.c' or 'src/led.c'. Always write complete file content — never partial.",
		func(_ context.Context, a fileWriteArgs) (string, []tools.Artifact, error) {
			if strings.Contains(a.Path, "..") {
				return "", nil, fmt.Errorf("path must not contain '..'")
			}
			root, err := WorkspaceDir()
			if err != nil {
				return "", nil, err
			}
			full := filepath.Join(root, filepath.FromSlash(a.Path))
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				return "", nil, err
			}
			if err := os.WriteFile(full, []byte(a.Content), 0o644); err != nil {
				return "", nil, err
			}
			return fmt.Sprintf("wrote %d bytes → %s", len(a.Content), a.Path), nil, nil
		})
}
