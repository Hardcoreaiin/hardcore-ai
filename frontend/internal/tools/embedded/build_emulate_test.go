package embedded

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/toolchain"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

// toolchainsInstalled reports whether the arm-gcc + qemu xPacks are present.
// The build/emulate end-to-end test is skipped when they are not.
func toolchainsInstalled(t *testing.T) bool {
	home, _ := os.UserHomeDir()
	tc := filepath.Join(home, ".hardcoreai", "toolchain")
	gcc, _ := filepath.Glob(filepath.Join(tc, "xpack-arm-none-eabi-gcc-*"))
	qemu, _ := filepath.Glob(filepath.Join(tc, "xpack-qemu-arm-*"))
	return len(gcc) > 0 && len(qemu) > 0
}

// TestBuildThenEmulate drives the real build and emulate tools end to end: it
// writes a firmware file, builds it for stm32f4 (which auto-generates the
// runtime), then emulates it and asserts the LOG output appears. This is the
// regression guard for "emulation prints nothing".
func TestBuildThenEmulate(t *testing.T) {
	if !toolchainsInstalled(t) {
		t.Skip("arm-gcc / qemu xPacks not installed")
	}

	dir := t.TempDir()
	SetWorkspaceRoot(dir)
	if err := os.WriteFile(filepath.Join(dir, "main.c"), []byte(`#include "hcai.h"
int main(void) {
    LOG("[LOG] firmware up, answer=%d\n", 42);
    for (int i = 0; i < 3; i++) LOG("[LOG] loop %d\n", i);
    return 0;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	mgr, err := toolchain.DefaultManager()
	if err != nil {
		t.Fatalf("toolchain manager: %v", err)
	}

	reg := tools.NewRegistry()
	RegisterBuild(reg, mgr)
	RegisterEmulate(reg, mgr)

	build, _ := reg.Get("build")
	bres, _, berr := build.Execute(context.Background(), []any{"stm32f4", ""})
	if berr != nil {
		t.Fatalf("build error: %v", berr)
	}
	if !strings.HasPrefix(bres, "OK:") {
		t.Fatalf("build did not succeed:\n%s", bres)
	}
	t.Logf("build: %s", bres)

	// The runtime files must have been generated into the project.
	for _, f := range []string{"hcai.h", "hcai_startup.c", "hcai_semihost.c", "hcai_linker.ld"} {
		if _, err := os.Stat(filepath.Join(dir, runtimeDirName, f)); err != nil {
			t.Errorf("runtime file %s not generated: %v", f, err)
		}
	}

	emulate, _ := reg.Get("emulate")
	eres, _, eerr := emulate.Execute(context.Background(), []any{"stm32f4", "", int64(5000)})
	if eerr != nil {
		t.Fatalf("emulate error: %v", eerr)
	}
	t.Logf("emulate: %s", eres)

	for _, want := range []string{"firmware up, answer=42", "loop 0", "loop 1", "loop 2"} {
		if !strings.Contains(eres, want) {
			t.Errorf("emulation output missing %q\nfull output:\n%s", want, eres)
		}
	}
}

// TestRuntimeProfileMachinesAreReal is a cheap guard that every QEMU machine
// referenced by a runtime profile is non-empty and looks like a machine id
// (no spaces). It does not invoke QEMU.
func TestRuntimeProfileMachinesAreReal(t *testing.T) {
	for arch, p := range runtimeProfiles {
		if p.QEMUMachine == "" || strings.ContainsAny(p.QEMUMachine, " \t") {
			t.Errorf("%s: bad QEMU machine %q", arch, p.QEMUMachine)
		}
		if p.FlashOrigin == "" || p.RAMOrigin == "" {
			t.Errorf("%s: incomplete memory layout", arch)
		}
	}
}
