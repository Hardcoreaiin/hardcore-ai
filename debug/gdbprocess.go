package debug

import (
	"bufio"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var (
	gdbCmd    *exec.Cmd
	gdbInput  io.WriteCloser
	gdbOutput *bufio.Scanner

	gdbMutex sync.Mutex
)

// =====================================================
// CONNECT
// =====================================================

func ConnectGDB() {

	gdbPath := "C:\\Users\\KI\\.platformio\\packages\\toolchain-gccarmnoneeabi\\bin\\arm-none-eabi-gdb.exe"

	gdbCmd = exec.Command(gdbPath)

	stdin, err := gdbCmd.StdinPipe()
	if err != nil {
		Log(ERROR, "GDB", err.Error())
		return
	}

	stdout, err := gdbCmd.StdoutPipe()
	if err != nil {
		Log(ERROR, "GDB", err.Error())
		return
	}

	stderr, err := gdbCmd.StderrPipe()
	if err != nil {
		Log(ERROR, "GDB", err.Error())
		return
	}

	gdbInput = stdin
	gdbOutput = bufio.NewScanner(stdout)

	go StreamGDBErrors(stderr)

	err = gdbCmd.Start()
	if err != nil {
		Log(ERROR, "GDB", err.Error())
		return
	}

	Log(INFO, "GDB", "GDB started")

	// Start async reader
	StartGDBReader()

	time.Sleep(time.Millisecond * 500)

	// Setup target
	SendGDBCommand("set pagination off")

	SendGDBCommand(
		"file C:/Users/KI/Documents/PlatformIO/Projects/stm32_uart_debug/.pio/build/bluepill_f103c8/firmware.elf",
	)

	SendGDBCommand(
		"target extended-remote localhost:3333",
	)

	time.Sleep(time.Second)

	SendGDBCommand("monitor halt")

	time.Sleep(time.Millisecond * 500)

	SendGDBCommand("info registers")

	Log(INFO, "GDB", "Debugger ready")
}

// =====================================================
// ASYNC READER
// =====================================================

func StartGDBReader() {

	go func() {

		for gdbOutput.Scan() {

			line := strings.TrimSpace(
				gdbOutput.Text(),
			)

			if line == "" {
				continue
			}

			ParseGDBLine(line)
		}

		Log(ERROR, "GDB", "Reader stopped")
	}()
}

// =====================================================
// LINE PARSER
// =====================================================

func ParseGDBLine(line string) {

	Log(DEBUG, "GDB_OUT", line)

	lower := strings.ToLower(line)

	// =========================================
	// CPU STATE DETECTION FIRST
	// =========================================

	if strings.Contains(lower, "continuing") {

		SetCPUState("RUNNING")

		Log(INFO, "CPU_STATE", "RUNNING")
	}

	if strings.Contains(lower, "breakpoint") {

		SetCPUState("BREAKPOINT")

		Log(INFO, "BREAKPOINT", line)
	}

	if strings.Contains(lower, "target halted") {

		SetCPUState("HALTED")

		Log(INFO, "CPU_STATE", "HALTED")
	}

	// =========================================
	// NOW skip pure prompts
	// =========================================

	if line == "(gdb)" {
		return
	}

	// =========================================
	// REGISTERS
	// =========================================

	if strings.Contains(line, "0x") {

		ParseRegisterDump(line)
	}
}

// =====================================================
// REGISTER PARSER
// =====================================================

func ParseRegisterDump(raw string) {

	lines := strings.Split(raw, "\n")

	for _, line := range lines {

		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "(gdb)") {
			continue
		}

		var reg string
		var val string

		// FORMAT:
		// pc 0x08000260
		// PC : 0x08000260

		if strings.Contains(line, ":") {

			parts := strings.Split(line, ":")

			if len(parts) >= 2 {

				reg = strings.TrimSpace(parts[0])

				right := strings.TrimSpace(parts[1])

				fields := strings.Fields(right)

				if len(fields) > 0 {
					val = fields[0]
				}
			}

		} else {

			fields := strings.Fields(line)

			if len(fields) >= 2 {

				reg = fields[0]
				val = fields[1]
			}
		}

		if reg == "" || val == "" {
			continue
		}

		if !strings.HasPrefix(val, "0x") {
			continue
		}

		regLower := strings.ToLower(reg)

		switch regLower {

		case "pc":

			if UpdateRegister(&CPUState.PC, val) {

				Log(INFO, "REGISTER", "PC "+val)
			}

		case "sp":

			if UpdateRegister(&CPUState.SP, val) {

				Log(INFO, "REGISTER", "SP "+val)
			}

		case "lr":

			if UpdateRegister(&CPUState.LR, val) {

				Log(INFO, "REGISTER", "LR "+val)
			}

		case "xpsr":

			if UpdateRegister(&CPUState.XPSR, val) {

				Log(INFO, "REGISTER", "XPSR "+val)
			}
		}
	}
}

// =====================================================
// COMMAND SENDER
// =====================================================

func SendGDBCommand(cmd string) {

	gdbMutex.Lock()
	defer gdbMutex.Unlock()

	if gdbInput == nil {
		return
	}

	if cmd == "" {
		return
	}

	_, err := gdbInput.Write(
		[]byte(cmd + "\n"),
	)

	if err != nil {

		Log(ERROR, "GDB_CMD", err.Error())
		return
	}

	Log(DEBUG, "GDB_CMD", cmd)
}

// =====================================================
// INTERRUPT
// =====================================================

func InterruptCPU() {

	gdbMutex.Lock()
	defer gdbMutex.Unlock()

	if gdbInput == nil {
		return
	}

	_, err := gdbInput.Write([]byte{3})

	if err != nil {

		Log(ERROR, "CPU", err.Error())
		return
	}
	SetCPUState("HALTED")

	Log(INFO, "CPU", "Interrupt sent")
}

// =====================================================
// CPU CONTROL
// =====================================================

func HaltCPU() {

	Log(INFO, "CPU", "Halting CPU")

	InterruptCPU()

	time.Sleep(time.Millisecond * 500)

	SendGDBCommand("monitor halt")

	time.Sleep(time.Millisecond * 300)

	SendGDBCommand("info registers")
}

func ContinueCPU() {

	Log(INFO, "CPU", "Continuing CPU")

	SendGDBCommand("continue")
}

func StepCPU() {

	Log(INFO, "CPU", "Single step")

	SendGDBCommand("stepi")

	time.Sleep(time.Millisecond * 300)

	SendGDBCommand("info registers")
}

// =====================================================
// STATE
// =====================================================

func SetCPUState(state string) {

	if CPUState.State == state {
		return
	}

	old := CPUState.State

	CPUState.State = state

	Log(
		INFO,
		"CPU_STATE",
		old+" -> "+state,
	)
}

// =====================================================
// STDERR
// =====================================================

func StreamGDBErrors(
	pipe interface{ Read([]byte) (int, error) },
) {

	scanner := bufio.NewScanner(pipe)

	for scanner.Scan() {

		line := strings.TrimSpace(
			scanner.Text(),
		)

		if line == "" {
			continue
		}

		level := INFO

		if strings.Contains(
			strings.ToLower(line),
			"error",
		) {

			level = ERROR
		}

		Log(level, "GDB_ERR", line)
	}
}
