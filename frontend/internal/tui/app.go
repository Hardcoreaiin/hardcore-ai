// Package tui is the Bubble Tea root model for the bubble prototype.
//
// Interactive: user types into a text input at the bottom, Enter sends a
// new turn through the agent Session. Every event from the agent is
// routed to the right bubble. Chat is always-on, thought + tool bubbles
// appear as their events arrive.
package tui

import (
	"context"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/bubbles"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	ctx     context.Context
	session *agent.Session

	chat    *bubbles.Chat
	thought *bubbles.Thought
	calls   []*bubbles.ToolCall

	input    textinput.Model
	turn     <-chan agent.Event
	thinking bool

	width  int
	height int
	err    error
}

func New(ctx context.Context, session *agent.Session) *Model {
	ti := textinput.New()
	ti.Placeholder = "ask anything…  (enter to send, ctrl+c to quit, ctrl+r to reset)"
	ti.Prompt = "› "
	ti.Focus()
	ti.CharLimit = 4000

	return &Model{
		ctx:     ctx,
		session: session,
		chat:    bubbles.NewChat(),
		thought: bubbles.NewThought(),
		input:   ti,
		width:   80,
		height:  24,
	}
}

type eventMsg struct{ ev agent.Event }
type streamClosedMsg struct{}

func (m *Model) waitEvent() tea.Cmd {
	ch := m.turn
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return streamClosedMsg{}
		}
		return eventMsg{ev: ev}
	}
}

func (m *Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.input.Width = msg.Width - 6
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+r":
			if !m.thinking {
				m.session.Reset()
				m.chat = bubbles.NewChat()
				m.thought = bubbles.NewThought()
				m.calls = nil
				m.err = nil
			}
			return m, nil
		case "enter":
			if m.thinking {
				return m, nil
			}
			text := strings.TrimSpace(m.input.Value())
			if text == "" {
				return m, nil
			}
			m.input.SetValue("")
			m.turn = m.session.Send(m.ctx, text)
			m.thinking = true
			return m, m.waitEvent()
		}

	case streamClosedMsg:
		m.thinking = false
		m.turn = nil
		return m, nil

	case eventMsg:
		m.route(msg.ev)
		if _, ok := msg.ev.(agent.TurnEndEvent); ok {
			m.thinking = false
		}
		return m, m.waitEvent()
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) route(ev agent.Event) {
	switch e := ev.(type) {
	case agent.ToolStartEvent:
		m.calls = append(m.calls, bubbles.NewToolCall(e))
	case agent.ToolResultEvent:
		if c := m.lastCallByName(e.Name); c != nil {
			c.ApplyResult(e)
		}
	case agent.ArtifactEvent:
		if c := m.lastCallByName(e.From); c != nil {
			c.ApplyArtifact(e)
		}
	case agent.ErrorEvent:
		m.err = e.Err
	case agent.TurnEndEvent:
		// handled in Update
	default:
		m.chat.Handle(ev)
		m.thought.Handle(ev)
	}
}

func (m *Model) lastCallByName(name string) *bubbles.ToolCall {
	for i := len(m.calls) - 1; i >= 0; i-- {
		if m.calls[i].Name == name {
			return m.calls[i]
		}
	}
	return nil
}

var (
	hintStyle = lipgloss.NewStyle().Faint(true)
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	busyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

func (m *Model) View() string {
	w := m.width
	var parts []string
	if m.chat.HasContent() {
		parts = append(parts, m.chat.View(w))
	}
	if m.thought.HasContent() {
		parts = append(parts, m.thought.View(w))
	}
	for _, c := range m.calls {
		parts = append(parts, c.View(w))
	}
	if m.err != nil {
		parts = append(parts, errStyle.Render("error: "+m.err.Error()))
	}

	status := "ctrl+c quit • ctrl+r reset • enter send"
	if m.thinking {
		status = busyStyle.Render("● thinking…") + "  " + status
	}
	parts = append(parts, m.input.View(), hintStyle.Render(status))
	return strings.Join(parts, "\n")
}
