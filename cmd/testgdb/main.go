package main

import (
	"fmt"

	"hardcore-debug/debug"
)

func main() {

	gdb, err := debug.NewGDBMI(
		"C:\\Users\\KI\\Documents\\PlatformIO\\Projects\\stm32_uart_debug\\.pio\\build\\bluepill_f103c8\\firmware.elf",
	)

	if err != nil {
		panic(err)
	}

	defer gdb.Close()

	err = gdb.Connect(
		"127.0.0.1",
		3333,
	)

	if err != nil {
		panic(err)
	}

	fmt.Println("GDB Connected")

	regs, err := gdb.ReadRegisters()
	if err != nil {
		panic(err)
	}

	fmt.Println("\nREGISTERS:")

	for _, line := range regs {
		fmt.Println(line)
	}

	bt, err := gdb.Backtrace()
	if err != nil {
		panic(err)
	}

	fmt.Println("\nBACKTRACE:")

	for _, line := range bt {
		fmt.Println(line)
	}

	frames := debug.ParseBacktrace(bt)

	fmt.Println("\nPARSED FRAMES:")

	for _, f := range frames {

		fmt.Printf(
			"#%d %s -> %s:%d\n",
			f.Level,
			f.Function,
			f.File,
			f.Line,
		)
	}
}
