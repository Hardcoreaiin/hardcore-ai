package embedded

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/toolchain"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

// archProfile holds the compiler binary name and flags for a CPU architecture.
type archProfile struct {
	compiler string
	flags    []string
	tool     toolchain.Tool
}

// knownArches maps arch identifiers to compiler profiles.
var knownArches = map[string]archProfile{
	// ARM Cortex-M (via arm-none-eabi-gcc)
	"cortex-m0":     {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m0", "-mthumb"}, tool: toolchain.ToolGCC},
	"cortex-m0plus": {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m0plus", "-mthumb"}, tool: toolchain.ToolGCC},
	"cortex-m3":     {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m3", "-mthumb"}, tool: toolchain.ToolGCC},
	"cortex-m4":     {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m4", "-mthumb"}, tool: toolchain.ToolGCC},
	"cortex-m4f":    {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m4", "-mthumb", "-mfpu=fpv4-sp-d16", "-mfloat-abi=hard"}, tool: toolchain.ToolGCC},
	"cortex-m7":     {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m7", "-mthumb"}, tool: toolchain.ToolGCC},
	"cortex-m7f":    {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m7", "-mthumb", "-mfpu=fpv5-d16", "-mfloat-abi=hard"}, tool: toolchain.ToolGCC},
	"cortex-m33":    {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m33", "-mthumb"}, tool: toolchain.ToolGCC},
	// STM32 chip families (convenience aliases)
	"stm32f0": {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m0", "-mthumb"}, tool: toolchain.ToolGCC},
	"stm32f1": {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m3", "-mthumb"}, tool: toolchain.ToolGCC},
	"stm32f2": {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m3", "-mthumb"}, tool: toolchain.ToolGCC},
	"stm32f3": {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m4", "-mthumb", "-mfpu=fpv4-sp-d16", "-mfloat-abi=hard"}, tool: toolchain.ToolGCC},
	"stm32f4": {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m4", "-mthumb", "-mfpu=fpv4-sp-d16", "-mfloat-abi=hard"}, tool: toolchain.ToolGCC},
	"stm32f7": {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m7", "-mthumb", "-mfpu=fpv5-d16", "-mfloat-abi=hard"}, tool: toolchain.ToolGCC},
	"stm32h7": {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m7", "-mthumb", "-mfpu=fpv5-d16", "-mfloat-abi=hard"}, tool: toolchain.ToolGCC},
	"stm32l4": {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m4", "-mthumb", "-mfpu=fpv4-sp-d16", "-mfloat-abi=hard"}, tool: toolchain.ToolGCC},
	"stm32g0": {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m0plus", "-mthumb"}, tool: toolchain.ToolGCC},
	"stm32g4": {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m4", "-mthumb", "-mfpu=fpv4-sp-d16", "-mfloat-abi=hard"}, tool: toolchain.ToolGCC},
	"stm32l0": {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m0plus", "-mthumb"}, tool: toolchain.ToolGCC},
	"stm32l1": {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m3", "-mthumb"}, tool: toolchain.ToolGCC},
	"stm32wb": {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m4", "-mthumb", "-mfpu=fpv4-sp-d16", "-mfloat-abi=hard"}, tool: toolchain.ToolGCC},
	// Nordic nRF
	"nrf51":  {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m0", "-mthumb"}, tool: toolchain.ToolGCC},
	"nrf52":  {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m4", "-mthumb", "-mfpu=fpv4-sp-d16", "-mfloat-abi=hard"}, tool: toolchain.ToolGCC},
	"nrf5340": {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m33", "-mthumb"}, tool: toolchain.ToolGCC},
	// RP2040
	"rp2040": {compiler: "arm-none-eabi-gcc", flags: []string{"-mcpu=cortex-m0plus", "-mthumb"}, tool: toolchain.ToolGCC},
	// ESP32 (Xtensa — must be in PATH)
	"esp32": {compiler: "xtensa-esp32-elf-gcc", flags: []string{"-mlongcalls"}, tool: ""},
	// RISC-V (must be in PATH)
	"riscv32": {compiler: "riscv32-unknown-elf-gcc", flags: []string{"-march=rv32imac", "-mabi=ilp32"}, tool: ""},
	"riscv64": {compiler: "riscv64-unknown-elf-gcc", flags: []string{"-march=rv64imac", "-mabi=lp64"}, tool: ""},
}

func archList() string {
	keys := make([]string, 0, len(knownArches))
	for k := range knownArches {
		keys = append(keys, k)
	}
	return strings.Join(keys, ", ")
}

type buildArgs struct {
	Arch  string `tool:"arch"  desc:"target architecture or chip family, e.g. cortex-m4f, rp2040, nrf52, riscv32"`
	Entry string `tool:"entry" desc:"entry C file relative to project root, e.g. main.c (leave blank to compile all .c files)"`
}

func RegisterBuild(r *tools.Registry, mgr *toolchain.Manager) {
	tools.Register(r, "build",
		"Compile C source files in the current project into an ELF binary for any supported embedded target. "+
			"The arch argument selects compiler flags automatically. Returns compiler output (errors/warnings). "+
			"Supported: "+archList(),
		func(ctx context.Context, a buildArgs) (string, []tools.Artifact, error) {
			arch := strings.ToLower(strings.TrimSpace(a.Arch))
			profile, ok := knownArches[arch]
			if !ok {
				return "", nil, fmt.Errorf("unknown arch %q — supported: %s", a.Arch, archList())
			}

			compiler := profile.compiler
			if profile.tool != "" {
				ready, err := mgr.EnsureAsync(profile.tool)
				if err != nil {
					return "", nil, fmt.Errorf("toolchain install failed: %w", err)
				}
				if !ready {
					return "toolchain is downloading in the background — try again in a moment", nil, nil
				}
				compiler, err = mgr.BinPath(profile.tool, profile.compiler)
				if err != nil {
					return "", nil, err
				}
			}

			root, err := WorkspaceDir()
			if err != nil {
				return "", nil, err
			}

			buildDir := filepath.Join(root, "build")
			if err := os.MkdirAll(buildDir, 0o755); err != nil {
				return "", nil, err
			}
			elfPath := filepath.Join(buildDir, arch+".elf")

			var sources []string
			if strings.TrimSpace(a.Entry) != "" {
				sources = []string{filepath.Join(root, filepath.FromSlash(a.Entry))}
			} else {
				_ = filepath.Walk(root, func(path string, info os.FileInfo, werr error) error {
					if werr != nil || info.IsDir() {
						return werr
					}
					rel, _ := filepath.Rel(root, path)
					if strings.HasPrefix(filepath.ToSlash(rel), "build/") {
						return nil
					}
					if strings.HasSuffix(path, ".c") {
						sources = append(sources, path)
					}
					return nil
				})
			}

			if len(sources) == 0 {
				return "no .c files found in project", nil, nil
			}

			args := append(profile.flags,
				"-nostdlib",
				"-ffreestanding",
				"-O2",
				"-Wall",
				"-Wextra",
				"-I", root,
				"-o", elfPath,
			)
			args = append(args, sources...)

			cmd := exec.CommandContext(ctx, compiler, args...)
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
				return "BUILD FAILED:\n" + output, nil, nil
			}

			result := fmt.Sprintf("OK: compiled %d file(s) → build/%s.elf", len(sources), arch)
			if output != "" {
				result += "\nWarnings:\n" + output
			}
			return result, []tools.Artifact{{Type: "elf_path", Payload: elfPath}}, nil
		})
}
