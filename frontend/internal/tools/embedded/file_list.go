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
				if err != nil {
					return err
				}
				relRaw, _ := filepath.Rel(root, path)
				rel := filepath.ToSlash(relRaw)
				// Skip build output and the generated firmware runtime — they
				// are not the user's source and must not be edited by hand.
				if rel == "build" || rel == runtimeDirName {
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
				if info.IsDir() {
					return nil
				}
				files = append(files, rel)
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
