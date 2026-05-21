package agent

import (
	"strings"
	"testing"
)

// TestTruncFileReadKeepsWholeSmallFile confirms a normal firmware file passes
// through untouched — the bug that caused endless re-reads was a 2 KB byte cap
// chopping the head off real source files.
func TestTruncFileReadKeepsWholeSmallFile(t *testing.T) {
	// ~300 lines of plausible source, well over the old 2000-byte cap.
	var b strings.Builder
	for i := 0; i < 300; i++ {
		b.WriteString("    GPIOA->ODR |= (1 << 5); /* toggle the status LED pin */\n")
	}
	src := b.String()
	if len(src) <= 2000 {
		t.Fatal("test setup: source should exceed the old byte cap")
	}
	got := truncFileRead(src)
	if got != src {
		t.Errorf("a 300-line file was altered by truncFileRead; want it whole")
	}
}

// TestTruncFileReadLineCap confirms an oversized file is cut on a line
// boundary (never mid-line, which would break a file_edit anchor) and the
// model is told lines were omitted.
func TestTruncFileReadLineCap(t *testing.T) {
	var b strings.Builder
	for i := 0; i < fileReadMaxLines+50; i++ {
		b.WriteString("x\n")
	}
	got := truncFileRead(b.String())
	lines := strings.Split(got, "\n")
	// fileReadMaxLines content lines + the notice line (+ trailing empty).
	if len(lines) > fileReadMaxLines+2 {
		t.Errorf("got %d lines, want <= %d", len(lines), fileReadMaxLines+2)
	}
	if !strings.Contains(got, "more line(s) not shown") {
		t.Error("oversized file should carry an omission notice")
	}
	for _, ln := range lines {
		if ln != "x" && ln != "" && !strings.Contains(ln, "not shown") {
			t.Errorf("unexpected partial/garbled line: %q", ln)
		}
	}
}

// TestResultForHistoryRouting confirms file_read uses the line cap while other
// tools keep the small byte cap.
func TestResultForHistoryRouting(t *testing.T) {
	big := strings.Repeat("a", 5000)
	if r := resultForHistory("build", big); !strings.Contains(r, "truncated") {
		t.Error("build result should be byte-truncated")
	}
	// 5000 one-char lines: under the line cap, so file_read keeps it whole.
	manyLines := strings.Repeat("a\n", 100)
	if r := resultForHistory("file_read", manyLines); r != manyLines {
		t.Error("a 100-line file_read result should pass through whole")
	}
}
