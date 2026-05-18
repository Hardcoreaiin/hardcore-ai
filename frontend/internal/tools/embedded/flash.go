package embedded

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

type flashArgs struct {
	Arch   string `tool:"arch"   desc:"chip architecture/family, same as used in build"`
	Binary string `tool:"binary" desc:"path to the compiled ELF or HEX file relative to project root (default: build/<arch>.elf)"`
	Port   string `tool:"port"   desc:"serial port or interface, e.g. /dev/ttyUSB0, /dev/ttyACM0, or 'auto'"`
}

func RegisterFlash(r *tools.Registry) {
	tools.Register(r, "flash",
		"Flash a compiled binary to a connected device. Uses OpenOCD for ARM targets. "+
			"Requires the device to be connected and openocd to be installed and in PATH.",
		func(ctx context.Context, a flashArgs) (string, []tools.Artifact, error) {
			arch := strings.ToLower(strings.TrimSpace(a.Arch))
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
				return "", nil, fmt.Errorf("binary not found: %s — run build first", binary)
			}

			var cmd *exec.Cmd
			switch {
			case isARMFamily(arch):
				port := strings.TrimSpace(a.Port)
				iface := "interface/stlink.cfg"
				if strings.Contains(port, "jlink") {
					iface = "interface/jlink.cfg"
				}
				target := openocdTarget(arch)
				cmd = exec.CommandContext(ctx, "openocd",
					"-f", iface,
					"-f", "target/"+target+".cfg",
					"-c", fmt.Sprintf("program %s verify reset exit", binary),
				)
			default:
				return "", nil, fmt.Errorf(
					"flash not yet automated for arch %q — please use the appropriate flash tool manually", arch)
			}

			cmd.Dir = root
			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out

			runErr := cmd.Run()
			output := strings.TrimSpace(out.String())
			if runErr != nil {
				if output == "" {
					output = runErr.Error()
				}
				return "FLASH FAILED:\n" + output, nil, nil
			}
			return "FLASH OK:\n" + output, nil, nil
		})
}

func isARMFamily(arch string) bool {
	return strings.HasPrefix(arch, "stm32") ||
		strings.HasPrefix(arch, "cortex-m") ||
		strings.HasPrefix(arch, "nrf") ||
		arch == "rp2040"
}

func openocdTarget(arch string) string {
	switch {
	case strings.HasPrefix(arch, "stm32f1") || arch == "stm32f2":
		return "stm32f1x"
	case strings.HasPrefix(arch, "stm32f3"):
		return "stm32f3x"
	case strings.HasPrefix(arch, "stm32f4"):
		return "stm32f4x"
	case strings.HasPrefix(arch, "stm32f7"):
		return "stm32f7x"
	case strings.HasPrefix(arch, "stm32h7"):
		return "stm32h7x"
	case strings.HasPrefix(arch, "stm32l"):
		return "stm32lx"
	case strings.HasPrefix(arch, "stm32g"):
		return "stm32g0x"
	case strings.HasPrefix(arch, "nrf51"):
		return "nrf51"
	case strings.HasPrefix(arch, "nrf52"):
		return "nrf52"
	case arch == "rp2040":
		return "rp2040"
	default:
		return arch
	}
}
