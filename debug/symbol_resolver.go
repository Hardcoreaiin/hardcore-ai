package debug

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

type SymbolInfo struct {
	Address  uint32
	Function string
	File     string
	Line     string
}

type SymbolResolver struct {
	elfPath string
}

func NewSymbolResolver(
	elf string,
) *SymbolResolver {

	return &SymbolResolver{
		elfPath: elf,
	}
}

func (s *SymbolResolver) Resolve(
	addr uint32,
) (*SymbolInfo, error) {

	cmd := exec.Command(
		"C:\\Users\\KI\\.platformio\\packages\\toolchain-gccarmnoneeabi@1.70201.0\\bin\\arm-none-eabi-addr2line.exe",
		"-f",
		"-C",
		"-e",
		s.elfPath,
		fmt.Sprintf(
			"0x%08X",
			addr,
		),
	)

	out, err := cmd.CombinedOutput()

	if err != nil {

		return nil, fmt.Errorf(
			"addr2line failed: %v\nOUTPUT:\n%s",
			err,
			string(out),
		)
	}

	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(
		strings.NewReader(
			string(out),
		),
	)

	lines := []string{}

	for scanner.Scan() {

		line := strings.TrimSpace(
			scanner.Text(),
		)

		if line == "" {
			continue
		}

		if strings.Contains(
			line,
			"Dwarf Error",
		) {
			continue
		}

		lines = append(
			lines,
			line,
		)
	}

	info := &SymbolInfo{
		Address: addr,
	}

	if len(lines) >= 1 {
		info.Function = lines[0]
	}

	if len(lines) >= 2 {
		info.Line = lines[1]
	}

	return info, nil
}
