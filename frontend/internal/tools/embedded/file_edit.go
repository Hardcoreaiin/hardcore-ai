package embedded

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/editmatch"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

type fileEditArgs struct {
	Path string `tool:"path" desc:"file to edit (must already exist): absolute, ~-relative, or relative to the current directory"`
	// Edits is filled by the loop's two-fence pairing, not by the tokenizer —
	// the model supplies before/after code in fenced blocks after the CALL.
	Edits string `tool:"edits" desc:"DO NOT pass inline — supplied as paired fenced code blocks (before, then after) on the lines after the CALL"`
}

// FileDiff is the artifact payload emitted after a successful file_edit so the
// TUI can render a diff bubble without re-running the matcher.
type FileDiff struct {
	Path string
	Old  string // file content before the edits
	New  string // file content after the edits
}

func RegisterFileEdit(r *tools.Registry) {
	tools.Register(r, "file_edit",
		"Precisely edit an EXISTING file without rewriting it whole — the right tool for fixing "+
			"build errors. Call as: CALL file_edit(\"path\") then, for each place to change, supply "+
			"TWO fenced ``` blocks: the FIRST is the exact lines to find (the lines to change plus "+
			"ONE unchanged line above and below as context), the SECOND is those lines after your "+
			"fix. The context lines must match the file verbatim including indentation. For several "+
			"fixes, emit several before/after pairs. Prefer this over file_write for any change to a "+
			"file that already exists.",
		func(_ context.Context, a fileEditArgs) (string, []tools.Artifact, error) {
			full, err := ResolvePath(a.Path)
			if err != nil {
				return "", nil, err
			}
			if a.Edits == "" {
				return "", nil, fmt.Errorf("no edits: after CALL file_edit(%q) supply paired ``` blocks — a before block then an after block for each change", a.Path)
			}

			orig, err := os.ReadFile(full)
			if err != nil {
				if os.IsNotExist(err) {
					return "", nil, fmt.Errorf("file does not exist: %s — use file_write to create a new file", full)
				}
				return "", nil, err
			}

			edits, err := editmatch.ParseEdits(a.Edits)
			if err != nil {
				return "", nil, err
			}

			updated, results := editmatch.ApplyAll(string(orig), edits)

			// Report the first failure with the partial-progress count so the
			// model knows what landed and what to retry.
			applied := 0
			for i, res := range results {
				if res.Err != nil {
					return "", nil, fmt.Errorf("edit %d of %d failed: %w (%d earlier edit(s) applied but not saved)",
						i+1, len(edits), res.Err, applied)
				}
				applied++
			}

			if updated == string(orig) {
				return "no change: the edits produced identical content", nil, nil
			}

			if err := os.WriteFile(full, []byte(updated), 0o644); err != nil {
				return "", nil, err
			}

			var spans []string
			for _, res := range results {
				spans = append(spans, fmt.Sprintf("lines %d-%d", res.StartLine, res.EndLine))
			}
			msg := fmt.Sprintf("applied %d edit(s) → %s (%s)", applied, full, strings.Join(spans, ", "))

			return msg, []tools.Artifact{{
				Type:    "file_diff",
				Payload: FileDiff{Path: a.Path, Old: string(orig), New: updated},
			}}, nil
		})
}
