package debug

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

type GDBMI struct {
	cmd *exec.Cmd

	stdin  io.WriteCloser
	stdout io.ReadCloser

	scanner *bufio.Scanner

	mu sync.Mutex
}

func NewGDBMI(
	elfPath string,
) (*GDBMI, error) {

	cmd := exec.Command(
		"arm-none-eabi-gdb",
		"--interpreter=mi2",
		elfPath,
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	g := &GDBMI{
		cmd: cmd,

		stdin:  stdin,
		stdout: stdout,

		scanner: bufio.NewScanner(stdout),
	}

	g.Send("set pagination off")
	g.Read()

	g.Send("set confirm off")
	g.Read()

	g.Send("set mi-async on")
	g.Read()

	return g, nil
}

func (g *GDBMI) Send(
	cmd string,
) error {

	g.mu.Lock()
	defer g.mu.Unlock()

	_, err := fmt.Fprintf(
		g.stdin,
		"%s\n",
		cmd,
	)

	return err
}

func (g *GDBMI) Read() ([]string, error) {

	lines := []string{}

	for g.scanner.Scan() {

		line := strings.TrimSpace(
			g.scanner.Text(),
		)

		if line == "" {
			continue
		}

		fmt.Println("[GDB]", line)

		lines = append(
			lines,
			line,
		)

		// SUCCESS
		if strings.HasPrefix(
			line,
			"^done",
		) {
			return lines, nil
		}

		// STOP EVENT
		if strings.HasPrefix(
			line,
			"*stopped",
		) {
			return lines, nil
		}

		// ERROR
		if strings.HasPrefix(
			line,
			"^error",
		) {
			return lines, nil
		}
	}

	return lines, nil
}
func (g *GDBMI) Connect(
	host string,
	port int,
) error {

	cmd := fmt.Sprintf(
		"-target-select remote %s:%d",
		host,
		port,
	)

	err := g.Send(cmd)
	if err != nil {
		return err
	}

	lines, err := g.Read()
	if err != nil {
		return err
	}

	fmt.Println("CONNECT RESPONSE:")

	for _, l := range lines {

		fmt.Println(l)

		if strings.HasPrefix(
			l,
			"^error",
		) {
			return fmt.Errorf(
				"gdb connect failed: %s",
				l,
			)
		}
	}

	return nil
}
func (g *GDBMI) Continue() error {

	err := g.Send("-exec-continue")
	if err != nil {
		return err
	}

	_, err = g.Read()

	return err
}

func (g *GDBMI) Halt() error {

	err := g.Send("-exec-interrupt")
	if err != nil {
		return err
	}

	_, err = g.Read()

	return err
}

func (g *GDBMI) Backtrace() ([]string, error) {

	err := g.Send("-stack-list-frames")
	if err != nil {
		return nil, err
	}

	return g.Read()
}

func (g *GDBMI) ReadRegisters() ([]string, error) {

	err := g.Send("-data-list-register-values x")
	if err != nil {
		return nil, err
	}

	return g.Read()
}

func (g *GDBMI) SetBreakpoint(
	location string,
) error {

	cmd := fmt.Sprintf(
		"-break-insert %s",
		location,
	)

	err := g.Send(cmd)
	if err != nil {
		return err
	}

	_, err = g.Read()

	return err
}

func (g *GDBMI) Close() error {

	if g.cmd != nil &&
		g.cmd.Process != nil {

		return g.cmd.Process.Kill()
	}

	return nil
}

func (g *GDBMI) ReadMemoryWord(
	addr uint32,
) ([]string, error) {

	cmd := fmt.Sprintf(
		"-data-read-memory-bytes 0x%08X 4",
		addr,
	)

	err := g.Send(cmd)
	if err != nil {
		return nil, err
	}

	return g.Read()
}
