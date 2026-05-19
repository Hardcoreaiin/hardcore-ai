package main

import "hardcore-debug/debug"

func main() {

	report := debug.DecodeFault(
		0x00000002,
		0x40000000,
		0x20000000,
		0x0,
	)

	report.Print()
}
