package main

import (
	"fmt"
	"hardcore-debug/debug"
	"time"

	"github.com/charmbracelet/bubbles/viewport"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	logsViewport viewport.Model
}
type tickMsg time.Time

func initialModel() model {

	vp := viewport.New(
		120,
		10,
	)

	vp.SetContent("Loading...")

	return model{
		logsViewport: vp,
	}
}

func tickCmd() tea.Cmd {

	return tea.Tick(
		time.Millisecond*200,
		func(t time.Time) tea.Msg {
			return tickMsg(t)
		},
	)

}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {

	case tickMsg:

		debug.RenderMutex.Lock()

		logs := append(
			[]string{},
			debug.RenderedLogs...,
		)

		debug.RenderMutex.Unlock()

		logsContent := ""

		for _, log := range logs {

			logsContent += log + "\n"
		}

		m.logsViewport.SetContent(
			logsContent,
		)

		return m, tickCmd()

	case tea.WindowSizeMsg:

		m.logsViewport.Width = msg.Width
		m.logsViewport.Height = 10

	case tea.KeyMsg:

		switch msg.String() {
		case "up":
			m.logsViewport.LineUp(1)

		case "down":
			m.logsViewport.LineDown(1)

		case "pgup":
			m.logsViewport.HalfViewUp()

		case "pgdown":
			m.logsViewport.HalfViewDown()

		case "b":
			go debug.SetBreakpoint("main")

		case "h":

			go debug.HaltCPU()

		case "c":

			go debug.ContinueCPU()

		case "j":
			m.logsViewport.LineDown(1)

		case "k":
			m.logsViewport.LineUp(1)

		case "s":

			go debug.StepCPU()
		case "u":
			m.logsViewport.HalfViewUp()

		case "d":
			m.logsViewport.HalfViewDown()

		case "q", "ctrl+c":

			debug.Cleanup()

			return m, tea.Quit

		}

	}
	var cmd tea.Cmd

	m.logsViewport, cmd =
		m.logsViewport.Update(msg)

	return m, cmd
}

func (m model) View() string {

	s := "\n"

	s += "========== LIVE DEBUG DASHBOARD ==========\n\n"

	s += fmt.Sprintf(
		"Total Events : %d\n",
		debug.SystemStats.TotalEvents,
	)

	s += fmt.Sprintf(
		"Info Events  : %d\n",
		debug.SystemStats.InfoCount,
	)

	s += fmt.Sprintf(
		"Warnings     : %d\n",
		debug.SystemStats.WarningCount,
	)

	s += fmt.Sprintf(
		"Errors       : %d\n",
		debug.SystemStats.ErrorCount,
	)

	s += fmt.Sprintf(
		"Debug Events : %d\n",
		debug.SystemStats.DebugCount,
	)

	s += fmt.Sprintf(
		"EPS          : %.2f\n",
		debug.SystemStats.EventsPerSecond,
	)

	s += fmt.Sprintf(
		"Queue Usage  : %d\n",
		debug.SystemStats.QueueUsage,
	)

	s += fmt.Sprintf(
		"Queue Pressure : %.2f%%\n",
		debug.SystemStats.QueuePressure,
	)

	s += fmt.Sprintf(
		"Dropped Events : %d\n",
		debug.SystemStats.DroppedEvents,
	)

	s += "\n=========== CPU STATE ===========\n\n"

	s += fmt.Sprintf(
		"PC    : %s\n",
		debug.CPUState.PC,
	)

	s += fmt.Sprintf(
		"SP    : %s\n",
		debug.CPUState.SP,
	)

	s += fmt.Sprintf(
		"LR    : %s\n",
		debug.CPUState.LR,
	)

	s += fmt.Sprintf(
		"XPSR  : %s\n",
		debug.CPUState.XPSR,
	)

	s += fmt.Sprintf(
		"STATE : %s\n",
		debug.CPUState.State,
	)

	s += "\n=========== CONTROLS ===========\n\n"

	s += "b = breakpoint at main\n"
	s += "h = halt cpu\n"
	s += "c = continue cpu\n"
	s += "s = single step\n"
	s += "q = quit\n"
	s += "j/k = scroll\n"

	s += "\n----------- RECENT EVENTS -----------\n\n"

	s += m.logsViewport.View()

	s += "\nPress q to quit.\n"

	return s

}

func main() {

	debug.InitEventStream()
	debug.InitProcessors()
	debug.StartEventListener()
	debug.InitStats()

	go debug.StartSerialReader(
		"COM10",
		9600,
	)

	go debug.StartOpenOCD()

	time.Sleep(time.Second * 2)

	go debug.ConnectGDB()

	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {

		panic(err)

	}

}
