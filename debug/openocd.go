package debug

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strings"
)

var openocdCmd *exec.Cmd

type OpenOCDDebugger struct {
	conn  net.Conn
	state DebugState
}

func New() *OpenOCDDebugger {
	return &OpenOCDDebugger{
		state: StateUnknown,
	}
}

func StreamOpenOCDLogs(
	pipe io.ReadCloser,
) {

	scanner := bufio.NewScanner(pipe)

	for scanner.Scan() {

		line := scanner.Text()

		ParseHardwareLog(line)
	}
}

func StartOpenOCD() {

	openocdCmd = exec.Command(
		"C:\\Users\\KI\\.platformio\\packages\\tool-openocd\\bin\\openocd.exe",

		"-f",
		"C:\\Users\\KI\\.platformio\\packages\\tool-openocd\\openocd\\scripts\\interface\\stlink.cfg",

		"-f",
		"C:\\Users\\KI\\.platformio\\packages\\tool-openocd\\openocd\\scripts\\target\\stm32f1x.cfg",
	)

	stdout, _ := openocdCmd.StdoutPipe()
	stderr, _ := openocdCmd.StderrPipe()

	go StreamOpenOCDLogs(stdout)
	go StreamOpenOCDLogs(stderr)

	err := openocdCmd.Start()

	if err != nil {

		Log(
			ERROR,
			"OPENOCD",
			"Failed: "+err.Error(),
		)

		return
	}

	Log(
		INFO,
		"OPENOCD",
		"Started",
	)
}

func (o *OpenOCDDebugger) Connect() error {

	conn, err := net.Dial(
		"tcp",
		"127.0.0.1:4444",
	)

	if err != nil {
		return err
	}

	o.conn = conn

	return nil
}

func (o *OpenOCDDebugger) Disconnect() error {

	if o.conn != nil {
		return o.conn.Close()
	}

	return nil
}

func (o *OpenOCDDebugger) send(
	cmd string,
) (string, error) {

	if o.conn == nil {
		return "", fmt.Errorf("not connected")
	}

	_, err := fmt.Fprintf(
		o.conn,
		"%s\n",
		cmd,
	)

	if err != nil {
		return "", err
	}

	reader := bufio.NewReader(o.conn)

	response, err := reader.ReadString('\n')

	if err != nil {
		return "", err
	}

	return response, nil
}

func (o *OpenOCDDebugger) Halt() error {

	_, err := o.send("halt")

	if err != nil {
		return err
	}

	o.state = StateHalted

	return nil
}

func (o *OpenOCDDebugger) Continue() error {

	_, err := o.send("resume")

	if err != nil {
		return err
	}

	o.state = StateRunning

	return nil
}

func (o *OpenOCDDebugger) Step() error {

	_, err := o.send("step")

	return err
}

func (o *OpenOCDDebugger) Reset() error {

	_, err := o.send("reset halt")

	if err != nil {
		return err
	}

	o.state = StateHalted

	return nil
}

func (o *OpenOCDDebugger) ReadRegisters() (
	*Registers,
	error,
) {

	regs := &Registers{}

	read := func(name string) uint32 {

		resp, err := o.send(
			fmt.Sprintf(
				"reg %s",
				name,
			),
		)

		if err != nil {
			return 0
		}

		var value uint32

		idx := strings.Index(resp, "0x")

		if idx == -1 {
			return 0
		}

		hexPart := resp[idx:]

		fmt.Sscanf(
			hexPart,
			"0x%x",
			&value,
		)

		return value
	}

	regs.R0 = read("r0")
	regs.R1 = read("r1")
	regs.R2 = read("r2")
	regs.R3 = read("r3")

	regs.R4 = read("r4")
	regs.R5 = read("r5")
	regs.R6 = read("r6")
	regs.R7 = read("r7")

	regs.R8 = read("r8")
	regs.R9 = read("r9")
	regs.R10 = read("r10")
	regs.R11 = read("r11")
	regs.R12 = read("r12")

	regs.SP = read("sp")
	regs.LR = read("lr")

	regs.PC = read("pc")
	if regs.PC < 0x08000000 {
		regs.PC += 0x08000000
	}
	regs.XPSR = read("xpsr")

	return regs, nil
}

func (o *OpenOCDDebugger) ReadMemory(
	addr uint32,
	size uint32,
) ([]byte, error) {

	cmd := fmt.Sprintf(
		"mdw 0x%08X %d",
		addr,
		size/4,
	)

	resp, err := o.send(cmd)

	if err != nil {
		return nil, err
	}

	return []byte(resp), nil
}

func (o *OpenOCDDebugger) WriteMemory(
	addr uint32,
	data []byte,
) error {

	for i := 0; i < len(data); i += 4 {

		var value uint32

		for j := 0; j < 4 && (i+j) < len(data); j++ {

			value |= uint32(
				data[i+j],
			) << (8 * j)
		}

		cmd := fmt.Sprintf(
			"mww 0x%08X 0x%08X",
			addr+uint32(i),
			value,
		)

		_, err := o.send(cmd)

		if err != nil {
			return err
		}
	}

	return nil
}

func (o *OpenOCDDebugger) SetBreakpoint(
	addr uint32,
) error {

	cmd := fmt.Sprintf(
		"bp 0x%08X 2 hw",
		addr,
	)

	_, err := o.send(cmd)

	return err
}

func (o *OpenOCDDebugger) RemoveBreakpoint(
	addr uint32,
) error {

	cmd := fmt.Sprintf(
		"rbp 0x%08X",
		addr,
	)

	_, err := o.send(cmd)

	return err
}

func (o *OpenOCDDebugger) GetState() DebugState {
	return o.state
}
