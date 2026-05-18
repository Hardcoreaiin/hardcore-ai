package stm32

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/toolchain"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

// qemuMachine maps target families to qemu-system-arm -machine values.
// xpack-qemu-arm ships its own STM32-aware machine types.
var qemuMachine = map[string]string{
	"stm32f1": "STM32-P103",
	"stm32f4": "STM32F4-Discovery",
	"stm32f7": "NUCLEO-F746ZG",
	"stm32h7": "NUCLEO-F746ZG", // closest available; peripheral coverage varies
	"stm32l4": "STM32F4-Discovery",
	"stm32g0": "STM32-P103",
}

type emulateArgs struct {
	Target    string `tool:"target"     desc:"chip family used during compile: stm32f1 stm32f4 stm32f7 stm32h7 stm32l4 stm32g0"`
	TimeoutMs int    `tool:"timeout_ms" desc:"how long to run emulation in milliseconds (default 3000, max 30000)"`
}

func RegisterEmulate(r *tools.Registry, mgr *toolchain.Manager) {
	tools.Register(r, "stm32_emulate",
		"Run the compiled ELF in QEMU STM32 emulation and return UART/semihosting output. Must call stm32_compile first. The 'target' argument must be the same chip-family string passed to stm32_compile (e.g. stm32f1, stm32f4) — NOT a filename, project name, or ELF name.",
		func(ctx context.Context, a emulateArgs) (string, []tools.Artifact, error) {
			machine, ok := qemuMachine[strings.ToLower(a.Target)]
			if !ok {
				return "", nil, fmt.Errorf("unknown target %q for emulation", a.Target)
			}

			if err := mgr.Ensure(ctx, toolchain.ToolQEMU); err != nil {
				return "", nil, fmt.Errorf("QEMU not ready: %w", err)
			}
			qemu, err := mgr.BinPath(toolchain.ToolQEMU, "qemu-system-arm")
			if err != nil {
				return "", nil, err
			}

			root, err := WorkspaceDir()
			if err != nil {
				return "", nil, err
			}
			elfPath := filepath.Join(root, "build", a.Target+".elf")

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
				"-kernel", elfPath,
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

			// timeout is expected — firmware runs forever until we stop it
			if runCtx.Err() == context.DeadlineExceeded {
				if output == "" {
					return fmt.Sprintf("(emulation ran %dms, no UART output captured)", ms), nil, nil
				}
				return fmt.Sprintf("UART output (%dms):\n%s", ms, output), nil, nil
			}

			if err != nil {
				if output != "" {
					return "QEMU error:\n" + output, nil, nil
				}
				return "QEMU error: " + err.Error(), nil, nil
			}

			if output == "" {
				return "(emulation exited cleanly, no output)", nil, nil
			}
			return "Output:\n" + output, nil, nil
		})
}
