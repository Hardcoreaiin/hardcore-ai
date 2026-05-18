package stm32

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

// targetFlags maps a short target string to gcc -mcpu / -mfpu flags.
var targetFlags = map[string][]string{
	"stm32f1": {"-mcpu=cortex-m3", "-mthumb"},
	"stm32f4": {"-mcpu=cortex-m4", "-mthumb", "-mfpu=fpv4-sp-d16", "-mfloat-abi=hard"},
	"stm32f7": {"-mcpu=cortex-m7", "-mthumb", "-mfpu=fpv5-d16", "-mfloat-abi=hard"},
	"stm32h7": {"-mcpu=cortex-m7", "-mthumb", "-mfpu=fpv5-d16", "-mfloat-abi=hard"},
	"stm32l4": {"-mcpu=cortex-m4", "-mthumb", "-mfpu=fpv4-sp-d16", "-mfloat-abi=hard"},
	"stm32g0": {"-mcpu=cortex-m0plus", "-mthumb"},
}

type compileArgs struct {
	Target string `tool:"target" desc:"chip family: stm32f1 stm32f4 stm32f7 stm32h7 stm32l4 stm32g0"`
	Entry  string `tool:"entry"  desc:"entry C file relative to workspace, e.g. main.c"`
}

func RegisterCompile(r *tools.Registry, mgr *toolchain.Manager) {
	tools.Register(r, "stm32_compile",
		"Compile C source files in the STM32 workspace into an ELF binary. Returns gcc output (errors/warnings). The ELF is written to workspace/build/<target>.elf.",
		func(ctx context.Context, a compileArgs) (string, []tools.Artifact, error) {
			flags, ok := targetFlags[strings.ToLower(a.Target)]
			if !ok {
				families := make([]string, 0, len(targetFlags))
				for k := range targetFlags {
					families = append(families, k)
				}
				return "", nil, fmt.Errorf("unknown target %q — supported: %s", a.Target, strings.Join(families, ", "))
			}

			if err := mgr.Ensure(ctx, toolchain.ToolGCC); err != nil {
				return "", nil, fmt.Errorf("toolchain not ready: %w", err)
			}
			gcc, err := mgr.BinPath(toolchain.ToolGCC, "arm-none-eabi-gcc")
			if err != nil {
				return "", nil, err
			}

			root, err := WorkspaceDir()
			if err != nil {
				return "", nil, err
			}

			buildDir := filepath.Join(root, "build")
			elfPath := filepath.Join(buildDir, a.Target+".elf")

			// Collect all .c files under the workspace (excluding build/)
			var sources []string
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

			if len(sources) == 0 {
				return "no .c files found in workspace", nil, nil
			}

			args := append(flags,
				"-nostdlib",
				"-ffreestanding",
				"-O2",
				"-Wall",
				"-Wextra",
				"-I", root,
				"-o", elfPath,
			)
			args = append(args, sources...)

			cmd := exec.CommandContext(ctx, gcc, args...)
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
				return "COMPILE FAILED:\n" + output, nil, nil
			}

			result := fmt.Sprintf("OK: compiled %d file(s) → build/%s.elf", len(sources), a.Target)
			if output != "" {
				result += "\nWarnings:\n" + output
			}
			return result, []tools.Artifact{{Type: "elf_path", Payload: elfPath}}, nil
		})
}
