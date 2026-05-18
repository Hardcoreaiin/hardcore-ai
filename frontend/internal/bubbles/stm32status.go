package bubbles

type buildStatus int

const (
	buildNone buildStatus = iota
	buildRunning
	buildOK
	buildFailed
)
