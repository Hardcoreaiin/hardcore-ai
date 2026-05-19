package main

import (
	"fmt"
	"hardcore-debug/debug"
	"time"
)

func main() {

	dbg := debug.New()

	debug.StartOpenOCD()

	err := dbg.Connect()
	err = dbg.Connect()
	if err != nil {
		panic(err)
	}

	err = dbg.Continue()
	if err != nil {
		panic(err)
	}

	time.Sleep(2 * time.Second)

	err = dbg.Halt()
	if err != nil {
		panic(err)
	}
	defer dbg.Disconnect()

	err = dbg.Halt()
	if err != nil {
		panic(err)
	}

	regs, err := dbg.ReadRegisters()
	if err != nil {
		panic(err)
	}

	fmt.Printf(
		"LIVE PC: 0x%08X\n",
		regs.PC,
	)

	resolver := debug.NewSymbolResolver(
		"C:\\Users\\KI\\Documents\\PlatformIO\\Projects\\stm32_uart_debug\\.pio\\build\\bluepill_f103c8\\firmware.elf",
	)

	info, err := resolver.Resolve(
		regs.PC,
	)

	if err != nil {
		panic(err)
	}

	fmt.Printf(
		"Function : %s\n",
		info.Function,
	)

	fmt.Printf(
		"Location : %s\n",
		info.Line,
	)
}
