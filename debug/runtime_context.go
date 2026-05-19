package debug

type RuntimeContext struct {
	Registers *Registers

	Stack []StackFrame

	Fault *FaultReport

	RecentLogs []string

	CurrentFunction string

	CurrentFile string

	CurrentLine int
}

func BuildRuntimeContext(
	regs *Registers,
	stack []StackFrame,
	fault *FaultReport,
	logs []string,
) *RuntimeContext {

	ctx := &RuntimeContext{
		Registers:  regs,
		Stack:      stack,
		Fault:      fault,
		RecentLogs: logs,
	}

	if len(stack) > 0 {

		ctx.CurrentFunction =
			stack[0].Function

		ctx.CurrentFile =
			stack[0].File

		ctx.CurrentLine =
			stack[0].Line
	}

	return ctx
}
