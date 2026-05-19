package debug

import "fmt"

func DumpStack(
	gdb *GDBMI,
	sp uint32,
	size int,
) error {

	mem, err := gdb.ReadMemory(
		sp,
		size,
	)

	if err != nil {
		return err
	}

	fmt.Println("\nSTACK DUMP:")

	for i := 0; i < len(mem.Bytes); i += 4 {

		if i+3 >= len(mem.Bytes) {
			break
		}

		val :=
			uint32(mem.Bytes[i]) |
				uint32(mem.Bytes[i+1])<<8 |
				uint32(mem.Bytes[i+2])<<16 |
				uint32(mem.Bytes[i+3])<<24

		fmt.Printf(
			"0x%08X : 0x%08X\n",
			sp+uint32(i),
			val,
		)
	}

	return nil
}
