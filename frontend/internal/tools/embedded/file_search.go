package embedded

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

type fileSearchArgs struct {
	Query string `tool:"query" desc:"search term to fuzzy-match against file names and file contents"`
}

func RegisterFileSearch(r *tools.Registry) {
	tools.Register(r, "file_search",
		"Fuzzy search over file names and file contents in the current project. Returns matching file paths and the lines that matched.",
		func(_ context.Context, a fileSearchArgs) (string, []tools.Artifact, error) {
			query := strings.ToLower(strings.TrimSpace(a.Query))
			if query == "" {
				return "", nil, fmt.Errorf("query must not be empty")
			}
			root, err := WorkspaceDir()
			if err != nil {
				return "", nil, err
			}
			var results []string
			err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return err
				}
				rel, _ := filepath.Rel(root, path)
				relSlash := filepath.ToSlash(rel)

				if strings.Contains(strings.ToLower(relSlash), query) {
					results = append(results, fmt.Sprintf("[name match] %s", relSlash))
					return nil
				}

				if info.Size() > 512*1024 {
					return nil
				}
				data, err := os.ReadFile(path)
				if err != nil {
					return nil
				}
				lines := strings.Split(string(data), "\n")
				for i, line := range lines {
					if strings.Contains(strings.ToLower(line), query) {
						results = append(results, fmt.Sprintf("%s:%d: %s", relSlash, i+1, strings.TrimSpace(line)))
					}
				}
				return nil
			})
			if err != nil {
				return "", nil, err
			}
			if len(results) == 0 {
				return fmt.Sprintf("no matches for %q", a.Query), nil, nil
			}
			return strings.Join(results, "\n"), nil, nil
		})
}
