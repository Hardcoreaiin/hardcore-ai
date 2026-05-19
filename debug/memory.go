package debug

import (
	"fmt"
	"regexp"
	"strconv"
)

type MemoryBlock struct {
	Address uint32
	Bytes   []byte
}

var memRegex = regexp.MustCompile(
	`memory=\[\{begin="0x([0-9a-fA-F]+)",offset="0x0",end="0x([0-9a-fA-F]+)",contents="([0-9a-fA-F]+)"`,
)

func (g *GDBMI) ReadMemory(
	addr uint32,
	size int,
) (*MemoryBlock, error) {

	cmd := fmt.Sprintf(
		"-data-read-memory-bytes 0x%08X %d",
		addr,
		size,
	)

	err := g.Send(cmd)
	if err != nil {
		return nil, err
	}

	lines, err := g.Read()
	if err != nil {
		return nil, err
	}

	for _, line := range lines {

		m := memRegex.FindStringSubmatch(
			line,
		)

		if len(m) == 4 {

			data := m[3]

			bytes := []byte{}

			for i := 0; i < len(data); i += 2 {

				v, _ := strconv.ParseUint(
					data[i:i+2],
					16,
					8,
				)

				bytes = append(
					bytes,
					byte(v),
				)
			}

			return &MemoryBlock{
				Address: addr,
				Bytes:   bytes,
			}, nil
		}
	}

	return nil, fmt.Errorf(
		"memory parse failed",
	)
}
