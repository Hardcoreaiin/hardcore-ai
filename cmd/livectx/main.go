package main

import (
	"fmt"
	"hardcore-debug/debug"
	"time"
)

func main() {

	debug.StartOpenOCD()

	time.Sleep(3 * time.Second)

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
	fmt.Println("GDB REMOTE CONNECTED")

	if err != nil {
		panic(err)
	}

	fmt.Println("CONNECTED TO GDB")

	// -----------------------------
	// REGISTERS
	// -----------------------------

	regLines, err := gdb.ReadRegisters()
	if err != nil {
		panic(err)
	}

	regs := debug.ParseRegisters(
		regLines,
	)

	fault, _ := debug.ReadFaultRegisters(
		gdb,
	)

	fault.Print()

	debug.DumpStack(
		gdb,
		regs.SP,
		64,
	)

	debug.DumpGPIOA(
		gdb,
	)

	fmt.Println("\nREGISTERS:")

	fmt.Printf("PC   : 0x%08X\n", regs.PC)
	fmt.Printf("SP   : 0x%08X\n", regs.SP)
	fmt.Printf("LR   : 0x%08X\n", regs.LR)
	fmt.Printf("XPSR : 0x%08X\n", regs.XPSR)

	// -----------------------------
	// BACKTRACE
	// -----------------------------

	btLines, err := gdb.Backtrace()
	if err != nil {
		panic(err)
	}

	fmt.Println("\nRAW BACKTRACE:")

	for _, line := range btLines {
		fmt.Println(line)
	}

	frames := debug.ParseBacktrace(
		btLines,
	)

	// -----------------------------
	// LOGS
	// -----------------------------

	logs := []string{}
	// -----------------------------
	// CONTEXT
	// -----------------------------

	ctx := debug.BuildRuntimeContext(
		regs,
		frames,
		fault,
		logs,
	)

	// -----------------------------
	// OUTPUT
	// -----------------------------

	fmt.Println("\n==============================")
	fmt.Println("LIVE RUNTIME CONTEXT")
	fmt.Println("==============================")

	fmt.Printf(
		"\nCURRENT FUNCTION : %s\n",
		ctx.CurrentFunction,
	)

	fmt.Printf(
		"LOCATION : %s:%d\n",
		ctx.CurrentFile,
		ctx.CurrentLine,
	)

	fmt.Printf(
		"\nFAULT : %s\n",
		ctx.Fault.FaultType,
	)

	fmt.Printf(
		"CAUSE : %s\n",
		ctx.Fault.ProbableCause,
	)

	fmt.Printf(
		"\nSTACK TRACE:\n",
	)

	for _, f := range ctx.Stack {

		fmt.Printf(
			"#%d %s -> %s:%d\n",
			f.Level,
			f.Function,
			f.File,
			f.Line,
		)
	}

	fmt.Printf(
		"\nRECENT LOGS:\n",
	)

	for _, l := range ctx.RecentLogs {
		fmt.Printf("- %s\n", l)
	}

}
