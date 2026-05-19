package debug

func SetBreakpoint(location string) {

	SendGDBCommand(
		"break " + location,
	)

	Log(
		INFO,
		"BREAKPOINT",
		"Breakpoint set at "+location,
	)
}

func RemoveBreakpoint(id string) {

	SendGDBCommand(
		"delete " + id,
	)

	Log(
		INFO,
		"BREAKPOINT",
		"Breakpoint deleted "+id,
	)
}

func ListBreakpoints() {

	SendGDBCommand(
		"info breakpoints",
	)
}
