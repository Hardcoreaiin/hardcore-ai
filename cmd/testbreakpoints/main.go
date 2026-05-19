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
	if err != nil {
		panic(err)
	}

	defer dbg.Disconnect()

	fmt.Println("Connected")

	err = dbg.Reset()
	if err != nil {
		panic(err)
	}

	err = dbg.Halt()
	if err != nil {
		panic(err)
	}

	regs, err := dbg.ReadRegisters()
	if err != nil {
		panic(err)
	}

	fmt.Printf(
		"Current PC: 0x%08X\n",
		regs.PC,
	)

	bpm := debug.NewBreakpointManager(dbg)

	bpm.StartMonitor()

	_, err = bpm.SetBreakpoint(regs.PC)
	if err != nil {
		panic(err)
	}

	fmt.Println("Breakpoint Set")

	go func() {

		for event := range bpm.Events() {

			fmt.Printf(
				"\nEVENT => BP=%d ADDR=0x%08X\n",
				event.BreakpointID,
				event.Address,
			)
		}
	}()

	err = dbg.Continue()
	if err != nil {
		panic(err)
	}

	for {
		time.Sleep(time.Second)
	}
}
