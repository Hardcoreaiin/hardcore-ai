package debug

type State string

const (
	INIT         State = "INIT"
	THINKING     State = "THINKING"
	CALLING_TOOL State = "CALLING_TOOL"
	WAITING      State = "WAITING"
	DONE         State = "DONE"
	FAILED       State = "FAILED"
)
