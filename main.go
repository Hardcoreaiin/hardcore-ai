package main

import (
	"hardcore-debug/debug"
	"time"
)

func main() {

	// -----------------------------------
	// INITIALIZE STREAMING ENGINE
	// -----------------------------------

	debug.InitEventStream()

	debug.StartEventListener()

	debug.InitStats()

	// -----------------------------------
	// NORMAL EVENTS
	// -----------------------------------

	debug.Log(debug.INFO, "SYSTEM", "Agent Started")

	time.Sleep(time.Millisecond * 500)

	debug.Log(debug.DEBUG, "THINKING", "Analyzing task")

	time.Sleep(time.Millisecond * 500)

	debug.Log(debug.INFO, "TOOL", "Calling web search")

	time.Sleep(time.Millisecond * 500)

	debug.Log(debug.WARNING, "NETWORK", "Slow response detected")

	time.Sleep(time.Millisecond * 500)

	debug.Log(debug.ERROR, "TOOL", "Tool timeout")

	time.Sleep(time.Millisecond * 500)

	debug.Log(debug.INFO, "SYSTEM", "Retry successful")

	time.Sleep(time.Millisecond * 500)

	// -----------------------------------
	// SIMULATE EVENT BURST
	// -----------------------------------

	for i := 0; i < 50; i++ {

		debug.Log(
			debug.INFO,
			"NETWORK",
			"Incoming packet",
		)

	}

	// -----------------------------------
	// FINAL EVENT
	// -----------------------------------

	debug.Log(debug.INFO, "SYSTEM", "Task completed")

	// -----------------------------------
	// SAVE SESSION
	// -----------------------------------

	err := debug.SaveLogs()

	if err != nil {
		panic(err)
	}

	// -----------------------------------
	// PRINT TELEMETRY STATS
	// -----------------------------------

	debug.PrintStats()

	// Allow goroutine to finish rendering
	time.Sleep(time.Second * 2)
}
