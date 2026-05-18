package embedded

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

type fileListArgs struct{}

func RegisterFileList(r *tools.Registry) {
	tools.Register(r, "file_list",
		"List all files in the current project directory, showing the directory tree.",
		func(_ context.Context, _ fileListArgs) (string, []tools.Artifact, error) {
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
				return "(project is empty)", nil, nil
			}
			return strings.Join(files, "\n"), nil, nil
		})
}
