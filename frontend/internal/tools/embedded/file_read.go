package embedded

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

type fileReadArgs struct {
	Path string `tool:"path" desc:"file to read: absolute, ~-relative, or relative to the current directory"`
}

func RegisterFileRead(r *tools.Registry) {
	tools.Register(r, "file_read",
		"Read a file. The path may be absolute, ~-relative, or relative to the current directory. "+
			"Returns the file content with a 'N| ' line-number prefix on every line so build "+
			"errors (file:line) can be located. The 'N| ' prefix is display-only — when copying "+
			"lines into a file_edit before-block, copy ONLY the code, never the 'N| ' prefix.",
		func(_ context.Context, a fileReadArgs) (string, []tools.Artifact, error) {
			full, err := ResolvePath(a.Path)
			if err != nil {
				return "", nil, err
			}
			data, err := os.ReadFile(full)
			if err != nil {
				return "", nil, err
			}
			return numberLines(string(data)), nil, nil
		})
}

// numberLines prefixes each line with a right-aligned "N| " gutter, like
// `cat -n`. The width adapts to the line count. A trailing newline in the
// source does not produce a spurious empty numbered line.
func numberLines(content string) string {
	if content == "" {
		return ""
	}
	lines := strings.Split(content, "\n")
	// A file ending in "\n" splits to a final empty element — drop it so the
	// last real line is the last numbered line.
	if len(lines) > 1 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	width := len(fmt.Sprintf("%d", len(lines)))
	var b strings.Builder
	for i, ln := range lines {
		fmt.Fprintf(&b, "%*d| %s\n", width, i+1, ln)
	}
	return b.String()
}
