package main

import (
	"fmt"

	"hardcore-debug/debug"
)

func main() {

	regs := &debug.Registers{
		PC: 0x080018e0,
		SP: 0x20004ff0,
		LR: 0x080018e9,
	}

	stack := []debug.StackFrame{
		{
			Level:    0,
			Function: "my_test_function",
			File:     "src/main.cpp",
			Line:     5,
		},
		{
			Level:    1,
			Function: "setup",
			File:     "src/main.cpp",
			Line:     12,
		},
	}

	fault := debug.DecodeFault(
		0x00000002,
		0x40000000,
		0x20000000,
		0x0,
	)

	logs := []string{
		"UART RX overflow",
		"DMA timeout detected",
	}

	ctx := debug.BuildRuntimeContext(
		regs,
		stack,
		fault,
		logs,
	)

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
		"\nRECENT LOGS:\n",
	)

	for _, l := range ctx.RecentLogs {
		fmt.Printf("- %s\n", l)
	}
}
