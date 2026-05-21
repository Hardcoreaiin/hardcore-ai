package embedded

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/toolchain"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

// qemuMachineFor returns the verified QEMU -machine value for an arch. It is
// sourced from runtimeProfiles (runtime.go) so the machine the firmware is
// LINKED for and the machine it is EMULATED on can never disagree. Every value
// is a machine that exists in QEMU 9.x — unlike the old hand-typed map, which
// was full of names from an abandoned QEMU fork that no longer exist.
func qemuMachineFor(arch string) (string, bool) {
	p, ok := RuntimeProfileFor(arch)
	if !ok {
		return "", false
	}
	return p.QEMUMachine, true
}

// emulatableArches lists arches emulation supports, for error messages.
func emulatableArches() string {
	keys := make([]string, 0, len(runtimeProfiles))
	for k := range runtimeProfiles {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}

type emulateArgs struct {
	Arch      string `tool:"arch"       desc:"target architecture or chip family used during build, e.g. stm32f4, cortex-m4f, rp2040"`
	Binary    string `tool:"binary"     desc:"ELF file path relative to project root (default: build/<arch>.elf)"`
	TimeoutMs int    `tool:"timeout_ms" desc:"how long to run emulation in milliseconds (default 3000, max 30000)"`
}

// qemuSupportedMachines runs qemu-system-arm -machine help and returns the
// output so the LLM can pick a valid machine type.
func qemuSupportedMachines(qemuBin string) string {
	cmd := exec.Command(qemuBin, "-machine", "help")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	_ = cmd.Run()
	return strings.TrimSpace(out.String())
}

func RegisterEmulate(r *tools.Registry, mgr *toolchain.Manager) {
	tools.Register(r, "emulate",
		"Run a compiled ELF in QEMU and return its semihosting log output (everything "+
			"printed with the LOG() macro from hcai.h). Build the SAME arch first. "+
			"Supported archs: STM32 f0/f1/f3/f4/g0/g4/l0/l1/l4/wb, cortex-m3/m4/m4f/m7/m7f/m33.",
		func(ctx context.Context, a emulateArgs) (string, []tools.Artifact, error) {
			arch := strings.ToLower(strings.TrimSpace(a.Arch))
			machine, ok := qemuMachineFor(arch)
			if !ok {
				return "", nil, fmt.Errorf("emulation not supported for arch %q — supported: %s",
					arch, emulatableArches())
			}

			ready, err := mgr.EnsureAsync(toolchain.ToolQEMU)
			if err != nil {
				return "", nil, fmt.Errorf("QEMU install failed: %w", err)
			}
			if !ready {
				return "QEMU is downloading in the background — try again in a moment", nil, nil
			}
			qemu, err := mgr.BinPath(toolchain.ToolQEMU, "qemu-system-arm")
			if err != nil {
				return "", nil, err
			}

			root, err := WorkspaceDir()
			if err != nil {
				return "", nil, err
			}

			binary := strings.TrimSpace(a.Binary)
			if binary == "" {
				binary = filepath.Join(root, "build", arch+".elf")
			} else {
				binary = filepath.Join(root, filepath.FromSlash(binary))
			}

			if _, err := os.Stat(binary); err != nil {
				return fmt.Sprintf("ERROR: binary not found at %s — run build first", binary), nil, nil
			}

			ms := a.TimeoutMs
			if ms <= 0 {
				ms = 3000
			}
			if ms > 30000 {
				ms = 30000
			}

			runCtx, cancel := context.WithTimeout(ctx, time.Duration(ms)*time.Millisecond)
			defer cancel()

			args := []string{
				"-machine", machine,
				"-kernel", binary,
				"-nographic",
				"-semihosting",
				"-semihosting-config", "enable=on,target=native",
			}

			// QEMU routes semihosting output to stderr, and its own
			// diagnostics there too — so both streams are merged for capture.
			// A real QEMU failure (bad machine, unloadable kernel) is
			// identified by the "qemu-system-arm:" diagnostic prefix; firmware
			// semihosting output never carries that, even when the firmware
			// ends via SYS_EXIT (which makes QEMU return a non-zero code).
			cmd := exec.CommandContext(runCtx, qemu, args...)
			cmd.Dir = root
			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out

			err = cmd.Run()
			raw := strings.TrimSpace(out.String())

			// Separate QEMU's own diagnostic lines from firmware log lines.
			var diagLines, fwLines []string
			for _, ln := range strings.Split(raw, "\n") {
				if strings.HasPrefix(ln, "qemu-system-arm:") ||
					strings.HasPrefix(ln, "Use -machine help") {
					diagLines = append(diagLines, ln)
				} else if ln != "" {
					fwLines = append(fwLines, ln)
				}
			}
			fwOutput := strings.Join(fwLines, "\n")
			qemuMsg := strings.Join(diagLines, "\n")

			if qemuMsg != "" {
				if strings.Contains(qemuMsg, "unsupported machine type") {
					supported := qemuSupportedMachines(qemu)
					hint := ""
					if supported != "" {
						hint = "\nSupported machines on this host:\n" + supported
					}
					return "QEMU error:\n" + qemuMsg + hint, nil, nil
				}
				return "QEMU error:\n" + qemuMsg, nil, nil
			}

			// Timed out — firmware kept running (an infinite loop with no
			// hcai_exit). That is normal for blink-style firmware; whatever
			// it logged before the timeout is the result.
			if runCtx.Err() == context.DeadlineExceeded {
				if fwOutput == "" {
					return fmt.Sprintf("(emulation ran %dms with no LOG output — add LOG() calls, "+
						"and #include \"hcai.h\")", ms), nil, nil
				}
				return fmt.Sprintf("output (ran %dms, still looping):\n%s", ms, fwOutput), nil, nil
			}

			// Clean run: firmware reached the end of main() / called hcai_exit.
			if fwOutput == "" {
				return "(emulation finished with no LOG output — add LOG() calls, " +
					"and #include \"hcai.h\")", nil, nil
			}
			_ = err // non-zero exit from semihosting SYS_EXIT is expected
			return "output:\n" + fwOutput, nil, nil
		})
}
