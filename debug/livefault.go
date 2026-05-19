package debug

import "fmt"

func ReadFaultRegisters(
	gdb *GDBMI,
) (*FaultReport, error) {

	readWord := func(
		addr uint32,
	) uint32 {

		mem, err := gdb.ReadMemory(
			addr,
			4,
		)

		if err != nil || mem == nil {

			fmt.Printf(
				"[FAULT READ FAILED] 0x%08X\n",
				addr,
			)

			return 0
		}

		if len(mem.Bytes) < 4 {
			return 0
		}

		return uint32(mem.Bytes[0]) |
			uint32(mem.Bytes[1])<<8 |
			uint32(mem.Bytes[2])<<16 |
			uint32(mem.Bytes[3])<<24
	}

	cfsr := readWord(
		0xE000ED28,
	)

	hfsr := readWord(
		0xE000ED2C,
	)

	bfar := readWord(
		0xE000ED38,
	)

	mmfar := readWord(
		0xE000ED34,
	)

	return DecodeFault(
		cfsr,
		hfsr,
		bfar,
		mmfar,
	), nil
}
