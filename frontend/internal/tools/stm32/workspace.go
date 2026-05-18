package stm32

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

// activeWorkspace holds the current session's working directory.
// A single agent session reuses one workspace; reset between tasks if needed.
var (
	wsMu      sync.Mutex
	wsRootDir string
)

// WorkspaceDir returns (creating if needed) the active workspace root.
func WorkspaceDir() (string, error) {
	wsMu.Lock()
	defer wsMu.Unlock()
	if wsRootDir != "" {
		if _, err := os.Stat(wsRootDir); err == nil {
			return wsRootDir, nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".hardcoreai", "workspace")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	wsRootDir = dir
	return dir, nil
}

type writeFileArgs struct {
	Path    string `tool:"path"    desc:"relative path inside workspace, e.g. main.c or src/led.c"`
	Content string `tool:"content" desc:"full file content to write"`
}

type readFileArgs struct {
	Path string `tool:"path" desc:"relative path inside workspace to read"`
}

type listFilesArgs struct {
	// no args — lists entire workspace
}

func RegisterWorkspace(r *tools.Registry) {
	tools.Register(r, "stm32_write_file",
		"Write (or overwrite) a file in the STM32 workspace. Use relative paths like 'main.c' or 'src/init.c'.",
		func(_ context.Context, a writeFileArgs) (string, []tools.Artifact, error) {
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
			return fmt.Sprintf("wrote %d bytes to %s", len(a.Content), a.Path), nil, nil
		})

	tools.Register(r, "stm32_read_file",
		"Read a file from the STM32 workspace.",
		func(_ context.Context, a readFileArgs) (string, []tools.Artifact, error) {
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

	tools.Register(r, "stm32_list_files",
		"List all files currently in the STM32 workspace.",
		func(_ context.Context, _ listFilesArgs) (string, []tools.Artifact, error) {
			root, err := WorkspaceDir()
			if err != nil {
				return "", nil, err
			}
			var files []string
			err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return err
				}
				rel, _ := filepath.Rel(root, path)
				files = append(files, filepath.ToSlash(rel))
				return nil
			})
			if err != nil {
				return "", nil, err
			}
			if len(files) == 0 {
				return "(workspace is empty)", nil, nil
			}
			return strings.Join(files, "\n"), nil, nil
		})
}
