package debug

func Cleanup() {

	Log(
		INFO,
		"SYSTEM",
		"Shutting down debugger",
	)

	if gdbCmd != nil &&
		gdbCmd.Process != nil {

		gdbCmd.Process.Kill()
	}

	if openocdCmd != nil &&
		openocdCmd.Process != nil {

		openocdCmd.Process.Kill()
	}
}
