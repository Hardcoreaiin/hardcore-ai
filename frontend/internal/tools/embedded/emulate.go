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

// qemuMachine maps arch/chip family to qemu-system-arm -machine values.
var qemuMachine = map[string]string{
	// STM32 families
	"stm32f0": "STM32-P103",
	"stm32f1": "STM32-P103",
	"stm32f2": "STM32-P103",
	"stm32f3": "STM32F4-Discovery",
	"stm32f4": "STM32F4-Discovery",
	"stm32f7": "NUCLEO-F746ZG",
	"stm32h7": "NUCLEO-F746ZG",
	"stm32l0": "STM32-P103",
	"stm32l1": "STM32-P103",
	"stm32l4": "STM32F4-Discovery",
	"stm32g0": "STM32-P103",
	"stm32g4": "STM32F4-Discovery",
	"stm32wb": "STM32F4-Discovery",
	// Cortex-M generic
	"cortex-m0":     "STM32-P103",
	"cortex-m0plus": "STM32-P103",
	"cortex-m3":     "STM32-P103",
	"cortex-m4":     "STM32F4-Discovery",
	"cortex-m4f":    "STM32F4-Discovery",
	"cortex-m7":     "NUCLEO-F746ZG",
	"cortex-m7f":    "NUCLEO-F746ZG",
	"cortex-m33":    "NUCLEO-F746ZG",
	// RP2040
	"rp2040": "raspi2b",
	// nRF
	"nrf51":  "microbit",
	"nrf52":  "STM32F4-Discovery",
	"nrf5340": "NUCLEO-F746ZG",
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
		"Run a compiled ELF in QEMU emulation and return UART/semihosting output. "+
			"Must call build first. Supported archs: stm32f1–h7, cortex-m0/3/4/7, rp2040, nrf51/52.",
		func(ctx context.Context, a emulateArgs) (string, []tools.Artifact, error) {
			arch := strings.ToLower(strings.TrimSpace(a.Arch))
			machine, ok := qemuMachine[arch]
			if !ok {
				return "", nil, fmt.Errorf("emulation not supported for arch %q — supported: %s",
					arch, func() string {
						var keys []string
						for k := range qemuMachine {
							keys = append(keys, k)
						}
						return strings.Join(keys, ", ")
					}())
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

			cmd := exec.CommandContext(runCtx, qemu, args...)
			cmd.Dir = root
			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out

			err = cmd.Run()
			output := strings.TrimSpace(out.String())

			if runCtx.Err() == context.DeadlineExceeded {
				if output == "" {
					return fmt.Sprintf("(emulation ran %dms — no output captured)", ms), nil, nil
				}
				return fmt.Sprintf("output (%dms):\n%s", ms, output), nil, nil
			}

			if err != nil {
				if output != "" {
					// If QEMU rejected the machine type, tell the LLM which
					// machines this build of QEMU actually supports.
					if strings.Contains(output, "unsupported machine type") {
						supported := qemuSupportedMachines(qemu)
						hint := ""
						if supported != "" {
							hint = "\nSupported machines on this host:\n" + supported
						}
						return "QEMU error:\n" + output + hint, nil, nil
					}
					return "QEMU error:\n" + output, nil, nil
				}
				return "QEMU error: " + err.Error(), nil, nil
			}

			if output == "" {
				return "(emulation exited cleanly — no output)", nil, nil
			}
			return "output:\n" + output, nil, nil
		})
}
