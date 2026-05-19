package debug

func ProcessAnalytics(
	event DebugEvent,
) {

	switch event.Level {

	case ERROR:

		Log(
			WARNING,
			"ANALYTICS",
			"Error spike detected",
		)

	}
}
