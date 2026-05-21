package editmatch

import (
	"strings"
	"testing"
)

// ── Sample firmware with three real, common STM32 build errors ──────────────
//
//   line 12: missing semicolon                  → "expected ';' before ..."
//   line 18: wrong register name (ODR typo)      → "'GPIOA_ODR' undeclared"
//   line 22: assignment vs comparison in if      → "-Werror suggest-parentheses"
const buggyMain = `#include <stdint.h>

#define LOG(fmt, ...) printf("[LOG] " fmt "\n", ##__VA_ARGS__)
#define GPIOA ((volatile uint32_t *)0x40020000)

void delay(volatile uint32_t n) {
    while (n--) { }
}

int main(void) {
    LOG("boot");
    uint32_t mode = 0
    GPIOA[0] |= (1 << 10);

    int ready = 0;
    while (1) {
        LOG("tick");
        GPIOA_ODR |= (1 << 5);
        delay(500);

        ready = check();
        if (ready = 1) {
            LOG("ready");
        }
    }
    return 0;
}
`

// scenario is one build-error fix expressed in the paired-fence format.
type scenario struct {
	name       string
	resp       string // what the model emits: CALL file_edit + before/after fences
	wantInFile string // substring that must appear in the fixed file
	wantGone   string // substring that must NOT remain
}

var scenarios = []scenario{
	{
		name: "missing semicolon",
		resp: "CALL file_edit(\"main.c\")\n" +
			"```c\n" +
			"    LOG(\"boot\");\n" +
			"    uint32_t mode = 0\n" +
			"    GPIOA[0] |= (1 << 10);\n" +
			"```\n" +
			"```c\n" +
			"    LOG(\"boot\");\n" +
			"    uint32_t mode = 0;\n" +
			"    GPIOA[0] |= (1 << 10);\n" +
			"```",
		wantInFile: "uint32_t mode = 0;",
		wantGone:   "uint32_t mode = 0\n",
	},
	{
		name: "undeclared register typo",
		resp: "CALL file_edit(\"main.c\")\n" +
			"```c\n" +
			"        LOG(\"tick\");\n" +
			"        GPIOA_ODR |= (1 << 5);\n" +
			"        delay(500);\n" +
			"```\n" +
			"```c\n" +
			"        LOG(\"tick\");\n" +
			"        GPIOA[5] |= (1 << 5);\n" +
			"        delay(500);\n" +
			"```",
		wantInFile: "GPIOA[5] |= (1 << 5);",
		wantGone:   "GPIOA_ODR",
	},
	{
		name: "assignment in if condition",
		resp: "CALL file_edit(\"main.c\")\n" +
			"```c\n" +
			"        ready = check();\n" +
			"        if (ready = 1) {\n" +
			"            LOG(\"ready\");\n" +
			"```\n" +
			"```c\n" +
			"        ready = check();\n" +
			"        if (ready == 1) {\n" +
			"            LOG(\"ready\");\n" +
			"```",
		wantInFile: "if (ready == 1) {",
		wantGone:   "if (ready = 1) {",
	},
}

// TestScenariosFix is the core check: each real build error is located and
// corrected, leaving the buggy span gone.
func TestScenariosFix(t *testing.T) {
	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			edits, err := ParseEdits(sc.resp)
			if err != nil {
				t.Fatalf("ParseEdits: %v", err)
			}
			fixed, results := ApplyAll(buggyMain, edits)
			for _, r := range results {
				if r.Err != nil {
					t.Fatalf("ApplyEdit: %v", r.Err)
				}
			}
			if !strings.Contains(fixed, sc.wantInFile) {
				t.Errorf("fixed file missing %q", sc.wantInFile)
			}
			if strings.Contains(fixed, sc.wantGone) {
				t.Errorf("buggy span %q still present", sc.wantGone)
			}
			t.Logf("%s: applied at lines %d-%d", sc.name, results[0].StartLine, results[0].EndLine)
		})
	}
}

// TestMultiSiteOneTurn checks that several edit sites in a single response are
// all applied — the key win over whole-file regeneration.
func TestMultiSiteOneTurn(t *testing.T) {
	resp := "CALL file_edit(\"main.c\")\n" +
		"```c\n" +
		"    LOG(\"boot\");\n" +
		"    uint32_t mode = 0\n" +
		"    GPIOA[0] |= (1 << 10);\n" +
		"```\n" +
		"```c\n" +
		"    LOG(\"boot\");\n" +
		"    uint32_t mode = 0;\n" +
		"    GPIOA[0] |= (1 << 10);\n" +
		"```\n" +
		"```c\n" +
		"        ready = check();\n" +
		"        if (ready = 1) {\n" +
		"            LOG(\"ready\");\n" +
		"```\n" +
		"```c\n" +
		"        ready = check();\n" +
		"        if (ready == 1) {\n" +
		"            LOG(\"ready\");\n" +
		"```"
	edits, err := ParseEdits(resp)
	if err != nil {
		t.Fatalf("ParseEdits: %v", err)
	}
	if len(edits) != 2 {
		t.Fatalf("want 2 edits, got %d", len(edits))
	}
	fixed, results := ApplyAll(buggyMain, edits)
	for i, r := range results {
		if r.Err != nil {
			t.Fatalf("edit %d: %v", i, r.Err)
		}
	}
	if !strings.Contains(fixed, "uint32_t mode = 0;") || !strings.Contains(fixed, "if (ready == 1) {") {
		t.Error("not all sites fixed")
	}
}

// TestOddFenceCountRejected confirms a forgotten after-block fails loudly.
func TestOddFenceCountRejected(t *testing.T) {
	resp := "CALL file_edit(\"main.c\")\n```c\nfoo\n```"
	if _, err := ParseEdits(resp); err == nil {
		t.Fatal("expected odd-fence-count error, got none")
	}
}

// TestAmbiguousAnchorRejected confirms the safety contract: an anchor that
// matches multiple places is refused rather than editing the wrong line.
func TestAmbiguousAnchorRejected(t *testing.T) {
	src := "    x = 1;\n    x = 1;\n"
	_, r := ApplyEdit(src, Edit{Old: "    x = 1;", New: "    x = 2;"})
	if r.Err == nil || !strings.Contains(r.Err.Error(), "ambiguous") {
		t.Errorf("expected ambiguity error, got %v", r.Err)
	}
}

// TestMissingAnchorRejected confirms a stale before-block is refused.
func TestMissingAnchorRejected(t *testing.T) {
	_, r := ApplyEdit(buggyMain, Edit{Old: "this text is not in the file", New: "x"})
	if r.Err == nil || !strings.Contains(r.Err.Error(), "not found") {
		t.Errorf("expected not-found error, got %v", r.Err)
	}
}

// TestNoOpRejected confirms identical before/after is refused.
func TestNoOpRejected(t *testing.T) {
	_, r := ApplyEdit(buggyMain, Edit{Old: "    LOG(\"boot\");", New: "    LOG(\"boot\");"})
	if r.Err == nil {
		t.Error("expected no-op error for identical before/after")
	}
}
