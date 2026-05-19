package debug

func InitProcessors() {

	RegisterProcessor(
		func(event DebugEvent) {

			Render(event)
		},
	)

	RegisterProcessor(
		func(event DebugEvent) {

			ProcessAnalytics(event)
		},
	)

	RegisterProcessor(
		func(event DebugEvent) {

			ProcessAlerts(event)
		},
	)
}
