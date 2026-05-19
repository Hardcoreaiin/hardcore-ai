package debug

type Debugger interface {
	Connect() error
	Disconnect() error

	Halt() error
	Continue() error
	Step() error
	Reset() error

	ReadRegisters() (*Registers, error)

	ReadMemory(addr uint32, size uint32) ([]byte, error)
	WriteMemory(addr uint32, data []byte) error

	SetBreakpoint(addr uint32) error
	RemoveBreakpoint(addr uint32) error

	GetState() DebugState
}

type Registers struct {
	R0  uint32
	R1  uint32
	R2  uint32
	R3  uint32
	R4  uint32
	R5  uint32
	R6  uint32
	R7  uint32
	R8  uint32
	R9  uint32
	R10 uint32
	R11 uint32
	R12 uint32

	SP   uint32
	LR   uint32
	PC   uint32
	XPSR uint32
}

type DebugState string

const (
	StateRunning DebugState = "RUNNING"
	StateHalted  DebugState = "HALTED"
	StateUnknown DebugState = "UNKNOWN"
)
