package debug

import (
	"strings"
)

func ParseHardwareLog(raw string) {

	level := INFO
	eventType := "HARDWARE"
	message := raw

	// Example:
	// [ERROR][TEMP] Sensor Failed

	if strings.Contains(raw, "[ERROR]") {
		level = ERROR
	}

	if strings.Contains(raw, "[WARNING]") {
		level = WARNING
	}

	if strings.Contains(raw, "[DEBUG]") {
		level = DEBUG
	}

	if strings.Contains(raw, "halted due to breakpoint") {

		SetCPUState("BREAKPOINT")

		Log(
			INFO,
			"BREAKPOINT",
			"Breakpoint hit",
		)

	} else if strings.Contains(raw, "halted due to") {

		SetCPUState("HALTED")

	} else if strings.Contains(raw, "resumed") {

		SetCPUState("RUNNING")
	}

	Log(
		level,
		eventType,
		message,
	)

}
