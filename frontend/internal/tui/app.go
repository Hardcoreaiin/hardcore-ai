// Package tui is the Bubble Tea root model.
//
// The root walks through three states: onboarding (theme + trust), the
// welcome splash, and the live chat. Onboarding only runs when no
// .agent_settings/settings.json exists for the working directory.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/bubbles"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/commands"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/config"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/pets"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/settings"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// state is the top-level screen the user is currently looking at.
type state int

const (
	stateOnboarding state = iota
	stateWelcome
	stateChat
)

type Model struct {
	ctx     context.Context
	session *agent.Session

	state   state
	theme   *theme.Theme
	root    string
	user    string
	version string
	trusted  bool
	provider string // active LLM provider key
	pet      string // pet species key (into pets.ByName)
	petName  string // user-assigned nickname

	onboard *onboarding
	welcome *bubbles.Welcome

	chat       *bubbles.Chat
	thought    *bubbles.Thought
	petBubble  *bubbles.PetBubble
	system     []string // system messages from command results
	historyVP  viewport.Model
	showingVP  bool // user scrolled into history; render viewport instead of latest turn

	cmds  *commands.Registry
	popup *cmdPopup

	input    textinput.Model
	turn     <-chan agent.Event
	thinking bool
	tick     int // animation clock, incremented by tickMsg

	width  int
	height int
	err    error
}

type tickMsg struct{}

func tickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}

// Options configures the root model.
type Options struct {
	Root    string
	User    string
	Version string
	// Existing settings; if Loaded is false, onboarding runs.
	Existing settings.Settings
	Loaded   bool
}

func New(ctx context.Context, session *agent.Session, opts Options) *Model {
	ti := textinput.New()
	ti.Placeholder = "ask anything…  (/ for commands, enter to send, ctrl+c to quit)"
	ti.Prompt = "› "
	ti.CharLimit = 4000

	m := &Model{
		ctx:     ctx,
		session: session,
		root:    opts.Root,
		user:    opts.User,
		version: opts.Version,
		input:   ti,
		width:   80,
		height:  24,
	}

	if opts.Loaded {
		th := theme.ByName(opts.Existing.Theme)
		m.theme = &th
		m.trusted = opts.Existing.Trusted
		m.provider = opts.Existing.Provider
		if m.provider == "" {
			m.provider = string(config.Load().Provider)
		}
		m.pet = opts.Existing.Pet
		if m.pet == "" {
			m.pet = pets.Blob.Name
		}
		m.petName = opts.Existing.PetName
		if m.petName == "" {
			m.petName = m.pet
		}
		m.cmds = m.buildCommands()
		m.popup = newCmdPopup(m.cmds)
		m.enterWelcome()
	} else {
		m.state = stateOnboarding
		m.onboard = newOnboarding(opts.Root)
		t := m.onboard.currentTheme()
		m.theme = &t
		m.provider = string(config.Load().Provider)
		m.pet = pets.Blob.Name
		m.petName = pets.Blob.Name
	}
	return m
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
	if m.state == stateOnboarding {
		return nil
	}
	return tea.Batch(textinput.Blink, tickCmd())
}

// SaveFunc persists settings to .agent_settings/settings.json.
type SaveFunc func(s settings.Settings) error

var saveHook SaveFunc

// SetSaveHook lets main register a persistence callback.
func SetSaveHook(f SaveFunc) { saveHook = f }

func (m *Model) persist() {
	if saveHook == nil {
		return
	}
	_ = saveHook(settings.Settings{
		Theme:    m.theme.Name,
		Trusted:  m.trusted,
		Provider: m.provider,
		Pet:      m.pet,
		PetName:  m.petName,
	})
}

func (m *Model) enterWelcome() {
	m.state = stateWelcome
	m.welcome = bubbles.NewWelcome(m.theme, m.version, m.user, m.root, pets.ByName(m.pet).Art)
}

func (m *Model) enterChat() {
	m.state = stateChat
	m.chat = bubbles.NewChat(m.theme)
	m.thought = bubbles.NewThought(m.theme)
	if m.petBubble == nil {
		m.petBubble = bubbles.NewPetBubble(m.theme, pets.ByName(m.pet).Art, m.petName)
	}
	m.historyVP = viewport.New(m.width, m.viewportHeight())
	m.input.Focus()
}

// viewportHeight is the height budget for the scrollback when the user
// pages up into history. Leaves room for the input + status line.
func (m *Model) viewportHeight() int {
	h := m.height - 4
	if h < 5 {
		h = 5
	}
	return h
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.input.Width = msg.Width - 6
		m.historyVP.Width = msg.Width
		m.historyVP.Height = m.viewportHeight()
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch m.state {
		case stateOnboarding:
			return m.updateOnboarding(msg)
		case stateWelcome:
			m.enterChat()
			return m, tea.Batch(textinput.Blink, tickCmd())
		case stateChat:
			return m.updateChat(msg)
		}

	case tickMsg:
		m.tick++
		return m, tickCmd()

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

	if m.state == stateChat {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		m.popup.Refresh(m.input.Value())
		return m, cmd
	}
	return m, nil
}

func (m *Model) updateOnboarding(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	done, trusted, picked, petName, petSpecies := m.onboard.handleKey(msg.String())
	cur := m.onboard.currentTheme()
	m.theme = &cur

	if done {
		final := picked
		m.theme = &final
		m.trusted = trusted
		m.pet = petSpecies
		if m.pet == "" {
			m.pet = pets.Blob.Name
		}
		m.petName = petName
		if m.petName == "" {
			m.petName = m.pet
		}
		m.cmds = m.buildCommands()
		m.popup = newCmdPopup(m.cmds)
		m.persist()
		m.enterWelcome()
		return m, nil
	}
	return m, nil
}

func (m *Model) updateChat(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// History scrollback: pgup/pgdn (and shift+up/down) navigate the
	// scrollback viewport. The popup takes priority when visible so it
	// can use up/down for selection.
	if !m.popup.Visible() {
		switch key {
		case "pgup", "shift+up":
			m.openHistory()
			m.historyVP.HalfPageUp()
			return m, nil
		case "pgdown", "shift+down":
			if m.showingVP {
				m.historyVP.HalfPageDown()
				if m.historyVP.AtBottom() {
					m.showingVP = false
				}
			}
			return m, nil
		case "esc":
			if m.showingVP {
				m.showingVP = false
				return m, nil
			}
		}
	}

	// Popup-aware keys take priority when the popup is open.
	if m.popup.Visible() {
		switch key {
		case "up":
			m.popup.Up()
			return m, nil
		case "down":
			m.popup.Down()
			return m, nil
		case "tab":
			if sug, ok := m.popup.Selected(); ok {
				m.input.SetValue(sug.Value)
				m.input.CursorEnd()
				m.popup.Refresh(m.input.Value())
			}
			return m, nil
		case "esc":
			m.popup.Hide()
			return m, nil
		}
	}

	switch key {
	case "ctrl+r":
		if !m.thinking {
			m.resetSession()
		}
		return m, nil
	case "enter":
		if m.thinking {
			return m, nil
		}
		// If popup has a selection and the input doesn't yet match it, accept it.
		if sug, ok := m.popup.Selected(); ok {
			if m.input.Value() != sug.Value {
				m.input.SetValue(sug.Value)
				m.input.CursorEnd()
				m.popup.Refresh(m.input.Value())
				return m, nil
			}
		}
		text := strings.TrimSpace(m.input.Value())
		if text == "" {
			return m, nil
		}
		m.input.SetValue("")
		m.popup.Hide()

		if commands.IsCommand(text) {
			quit := m.runCommand(text)
			if quit {
				return m, tea.Quit
			}
			return m, nil
		}
		m.turn = m.session.Send(m.ctx, text)
		m.thinking = true
		return m, m.waitEvent()
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.popup.Refresh(m.input.Value())
	return m, cmd
}

func (m *Model) resetSession() {
	m.session.Reset()
	m.chat = bubbles.NewChat(m.theme)
	m.thought = bubbles.NewThought(m.theme)
	m.system = nil
	m.err = nil
	m.showingVP = false
}

// openHistory loads all turns into the viewport and shows it.
func (m *Model) openHistory() {
	hist := m.chat.RenderAll(m.width)
	if hist == "" {
		return
	}
	m.historyVP.SetContent(hist)
	m.historyVP.GotoBottom()
	m.showingVP = true
}

func (m *Model) runCommand(input string) (quit bool) {
	res := m.cmds.Run(input)
	if res.Err != nil {
		m.system = append(m.system, "error: "+res.Err.Error())
		return false
	}
	if res.NewTheme != "" {
		t := theme.ByName(res.NewTheme)
		m.theme = &t
		m.chat.SetTheme(m.theme)
		m.thought.SetTheme(m.theme)
		if m.welcome != nil {
			m.welcome.SetTheme(m.theme)
		}
		if m.petBubble != nil {
			m.petBubble.SetTheme(m.theme)
		}
	}
	if res.NewPet != "" {
		m.pet = res.NewPet
		if m.welcome != nil {
			m.welcome.SetPet(pets.ByName(m.pet).Art)
		}
		if m.petBubble != nil {
			m.petBubble.SetArt(pets.ByName(m.pet).Art)
		}
	}
	if res.NewPetName != "" {
		m.petName = res.NewPetName
		if m.petBubble != nil {
			m.petBubble.SetName(m.petName)
		}
	}
	if res.NewTrusted != nil {
		m.trusted = *res.NewTrusted
	}
	if res.NewProvider != "" {
		cfg := config.LoadForProvider(config.Provider(res.NewProvider))
		m.session.SwapClient(cfg.BuildClient())
		m.provider = res.NewProvider
		m.persist()
	}
	if res.NewTheme != "" || res.NewPet != "" || res.NewPetName != "" || res.NewTrusted != nil {
		m.persist()
	}
	if res.ResetSession {
		m.resetSession()
	}
	if res.ClearVisual {
		m.chat = bubbles.NewChat(m.theme)
		m.thought = bubbles.NewThought(m.theme)
		m.system = nil
		m.err = nil
		m.showingVP = false
	}
	if res.Message != "" {
		m.system = append(m.system, res.Message)
	}
	return res.Quit
}

func (m *Model) route(ev agent.Event) {
	switch e := ev.(type) {
	case agent.ErrorEvent:
		m.err = e.Err
	case agent.TurnEndEvent:
		// handled in Update
	default:
		// Tool start/result/artifact, line, user message, done — chat
		// owns all of them so they can group into a single turn block.
		m.chat.Handle(ev)
		m.thought.Handle(ev)
	}
	if m.petBubble != nil {
		m.petBubble.Handle(ev)
	}
	// New content arrived: snap viewport back to the latest turn.
	if _, isUser := ev.(agent.UserMessageEvent); isUser {
		m.showingVP = false
	}
}

func (m *Model) View() string {
	switch m.state {
	case stateOnboarding:
		return m.onboard.view(m.width, m.height)
	case stateWelcome:
		hint := lipgloss.NewStyle().Foreground(m.theme.Muted).
			Render("press any key to continue…")
		return m.welcome.View(m.width) + "\n" + hint
	}
	return m.chatView()
}

func (m *Model) chatView() string {
	w := m.width
	t := m.theme
	hintStyle := lipgloss.NewStyle().Foreground(t.Muted)
	errStyle := lipgloss.NewStyle().Foreground(t.ErrorFg).Bold(true)
	busyStyle := lipgloss.NewStyle().Foreground(t.WarnFg)
	sysStyle := lipgloss.NewStyle().Foreground(t.Accent).Italic(true)

	var parts []string

	if m.showingVP {
		parts = append(parts, m.historyVP.View())
		parts = append(parts, hintStyle.Render("— scrollback (esc/pgdn to return) —"))
	} else {
		if m.petBubble != nil {
			parts = append(parts, m.petBubble.View(w, m.tick))
		}
		if latest := m.chat.RenderLatest(w); latest != "" {
			parts = append(parts, latest)
		}
		if m.thought.HasContent() {
			parts = append(parts, m.thought.View(w))
		}
	}

	for _, s := range m.system {
		parts = append(parts, sysStyle.Render("» "+s))
	}
	if m.err != nil {
		parts = append(parts, errStyle.Render("error: "+m.err.Error()))
	}

	if m.popup.Visible() {
		parts = append(parts, m.popup.View(m.theme, w))
	}

	trustTag := "trusted"
	if !m.trusted {
		trustTag = "read-only"
	}
	scrollHint := "pgup history"
	if len(m.chat.Turns()) == 0 {
		scrollHint = ""
	}
	status := "ctrl+c quit • ctrl+r reset • / commands • " + trustTag
	if scrollHint != "" {
		status += " • " + scrollHint
	}
	if m.thinking {
		status = busyStyle.Render("● thinking…") + "  " + status
	}
	parts = append(parts, m.input.View(), hintStyle.Render(status))
	return strings.Join(parts, "\n")
}

// buildCommands registers all slash commands. Closures capture the model
// so each command can read or mutate state, but state changes flow back
// through commands.Result so the model decides what to apply.
func (m *Model) buildCommands() *commands.Registry {
	r := commands.New()

	r.Register(&commands.Command{
		Name:        "theme",
		Description: "switch color theme",
		ArgValues: func() []string {
			var out []string
			for _, t := range theme.All() {
				out = append(out, t.Name)
			}
			return out
		},
		Run: func(args []string) commands.Result {
			if len(args) == 0 {
				var names []string
				for _, t := range theme.All() {
					names = append(names, t.Name)
				}
				return commands.Result{Message: "themes: " + strings.Join(names, ", ") + " (current: " + m.theme.Name + ")"}
			}
			name := args[0]
			for _, t := range theme.All() {
				if t.Name == name {
					return commands.Result{NewTheme: name, Message: "theme → " + name}
				}
			}
			return commands.Result{Err: fmt.Errorf("unknown theme: %s", name)}
		},
	})

	r.Register(&commands.Command{
		Name:        "pet",
		Description: "change the tamagotchi",
		ArgValues:   pets.Names,
		Run: func(args []string) commands.Result {
			if len(args) == 0 {
				return commands.Result{Message: "pets: " + strings.Join(pets.Names(), ", ") + " (current: " + m.pet + ")"}
			}
			name := args[0]
			for _, p := range pets.All() {
				if p.Name == name {
					return commands.Result{NewPet: name, Message: "pet → " + name}
				}
			}
			return commands.Result{Err: fmt.Errorf("unknown pet: %s", name)}
		},
	})

	r.Register(&commands.Command{
		Name:        "rename",
		Description: "give your pet a nickname",
		Run: func(args []string) commands.Result {
			if len(args) == 0 {
				return commands.Result{Message: "current pet name: " + m.petName + " (usage: /rename <name>)"}
			}
			newName := strings.Join(args, " ")
			return commands.Result{NewPetName: newName, Message: "pet renamed → " + newName}
		},
	})

	tt := true
	ff := false
	r.Register(&commands.Command{
		Name:        "trust",
		Description: "trust this directory (allow tool use)",
		Run: func(args []string) commands.Result {
			return commands.Result{NewTrusted: &tt, Message: "directory trusted"}
		},
	})
	r.Register(&commands.Command{
		Name:        "untrust",
		Description: "revoke trust for this directory",
		Run: func(args []string) commands.Result {
			return commands.Result{NewTrusted: &ff, Message: "directory untrusted (read-only)"}
		},
	})

	providers := []string{
		string(config.ProviderLlamaCpp),
		string(config.ProviderOpenRouter),
		string(config.ProviderGemini),
	}
	r.Register(&commands.Command{
		Name:        "model",
		Description: "switch LLM provider (llamacpp, openrouter, gemini)",
		ArgValues:   func() []string { return providers },
		Run: func(args []string) commands.Result {
			if len(args) == 0 {
				return commands.Result{Message: "providers: " + strings.Join(providers, ", ") + " (current: " + m.provider + ")"}
			}
			p := args[0]
			for _, v := range providers {
				if v == p {
					return commands.Result{NewProvider: p, Message: "model → " + p}
				}
			}
			return commands.Result{Err: fmt.Errorf("unknown provider: %s", p)}
		},
	})

	r.Register(&commands.Command{
		Name:        "reset",
		Description: "reset the agent session",
		Run: func(args []string) commands.Result {
			return commands.Result{ResetSession: true, Message: "session reset"}
		},
	})
	r.Register(&commands.Command{
		Name:        "clear",
		Description: "clear visible bubbles (keeps session)",
		Run: func(args []string) commands.Result {
			return commands.Result{ClearVisual: true}
		},
	})
	r.Register(&commands.Command{
		Name:        "quit",
		Description: "exit hardcore-ai",
		Run: func(args []string) commands.Result {
			return commands.Result{Quit: true}
		},
	})

	r.Register(&commands.Command{
		Name:        "help",
		Description: "list available commands",
		Run: func(args []string) commands.Result {
			var lines []string
			for _, c := range r.All() {
				lines = append(lines, "/"+c.Name+" — "+c.Description)
			}
			return commands.Result{Message: "commands:\n" + strings.Join(lines, "\n")}
		},
	})

	return r
}
