package debug

func ProcessAlerts(
	event DebugEvent,
) {

	if event.Type == "HARDWARE" &&
		event.Level == ERROR {

		Render(
			DebugEvent{
				ID:        0,
				Level:     ERROR,
				Type:      "ALERT",
				Message:   "Critical hardware issue detected",
				Timestamp: event.Timestamp,
			},
		)
	}
}
