package agent

import (
	"strings"
	"testing"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/editmatch"
)

// TestFileEditFencePairing simulates the loop's wiring for file_edit: parse the
// CALL out of a full model response, then hand the post-CALL text to
// fencesAfterCall and confirm editmatch.ParseEdits recovers the before/after
// pairs. This is the path most likely to break if the parser or the pairing
// helper changes.
func TestFileEditFencePairing(t *testing.T) {
	SetKnownTools([]string{"file_edit", "file_write", "build"})

	resp := strings.Join([]string{
		"THINK: build failed at main.c:12 — adding the missing semicolon",
		`CALL file_edit("main.c")`,
		"```c",
		`    LOG("boot");`,
		"    uint32_t mode = 0",
		"    GPIOA[0] |= (1 << 10);",
		"```",
		"```c",
		`    LOG("boot");`,
		"    uint32_t mode = 0;",
		"    GPIOA[0] |= (1 << 10);",
		"```",
	}, "\n")

	pl := ParseFullText(resp)
	if pl.Kind != LineCall || pl.FuncName != "file_edit" {
		t.Fatalf("ParseFullText: got kind=%d name=%q, want file_edit call", pl.Kind, pl.FuncName)
	}
	if len(pl.Tokens) != 1 {
		t.Fatalf("want 1 token (path), got %d: %v", len(pl.Tokens), pl.Tokens)
	}

	edits, ok := fencesAfterCall(resp, pl.Text)
	if !ok {
		t.Fatal("fencesAfterCall found no fences after the CALL")
	}

	parsed, err := editmatch.ParseEdits(edits)
	if err != nil {
		t.Fatalf("ParseEdits: %v", err)
	}
	if len(parsed) != 1 {
		t.Fatalf("want 1 edit pair, got %d", len(parsed))
	}
	if !strings.Contains(parsed[0].Old, "uint32_t mode = 0\n") {
		t.Errorf("before block missing the buggy line: %q", parsed[0].Old)
	}
	if !strings.Contains(parsed[0].New, "uint32_t mode = 0;") {
		t.Errorf("after block missing the fix: %q", parsed[0].New)
	}
}

// TestFileEditMultiSitePairing confirms several before/after pairs in one
// response all survive the parse → pairing → ParseEdits round trip.
func TestFileEditMultiSitePairing(t *testing.T) {
	SetKnownTools([]string{"file_edit"})

	resp := strings.Join([]string{
		`CALL file_edit("main.c")`,
		"```c", "a", "bad1", "c", "```",
		"```c", "a", "fix1", "c", "```",
		"```c", "d", "bad2", "f", "```",
		"```c", "d", "fix2", "f", "```",
	}, "\n")

	pl := ParseFullText(resp)
	if pl.Kind != LineCall {
		t.Fatalf("expected a CALL, got kind=%d", pl.Kind)
	}
	edits, ok := fencesAfterCall(resp, pl.Text)
	if !ok {
		t.Fatal("no fences found")
	}
	parsed, err := editmatch.ParseEdits(edits)
	if err != nil {
		t.Fatalf("ParseEdits: %v", err)
	}
	if len(parsed) != 2 {
		t.Fatalf("want 2 edit pairs, got %d", len(parsed))
	}
}

// TestFencesAfterCallNoFences confirms a bare file_edit call with no fenced
// blocks reports false so the tool can return a helpful error.
func TestFencesAfterCallNoFences(t *testing.T) {
	if _, ok := fencesAfterCall(`CALL file_edit("main.c")`, `file_edit("main.c")`); ok {
		t.Error("expected ok=false when no fences follow the CALL")
	}
}
