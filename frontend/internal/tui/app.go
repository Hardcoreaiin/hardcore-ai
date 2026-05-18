// Package tui is the Bubble Tea root model.
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

type state int

const (
	stateOnboarding state = iota
	stateWelcome
	stateChat
)

type Model struct {
	ctx     context.Context
	session *agent.Session

	state    state
	theme    *theme.Theme
	root     string
	user     string
	version  string
	trusted  bool
	provider string
	pet      string
	petName  string

	onboard *onboarding
	welcome *bubbles.Welcome

	chat      *bubbles.Chat
	thought   *bubbles.Thought
	petBubble *bubbles.PetBubble

	// petRendered caches the pet bubble render so glamour isn't called on every tick.
	// Invalidated when petTick changes or pet content changes.
	petRendered    string
	petRenderTick  int // last tick at which pet was rendered (updated every 3 ticks for animation)
	petRenderWidth int // invalidate cache if width changes

	system    []string
	mainVP    viewport.Model // single scrollable viewport for all chat content
	showingVP bool           // true while user is in scroll mode

	todoBubble  *bubbles.TodoBubble
	askBubble   *bubbles.AskBubble
	toolPanel   *bubbles.ToolPanel
	codeBubble  *bubbles.CodeBubble
	fileTree    *bubbles.FileTreeBubble
	stm32Status *bubbles.STM32StatusBubble

	cmds  *commands.Registry
	popup *cmdPopup

	input    textinput.Model
	turn     <-chan agent.Event
	thinking bool
	tick     int

	width  int
	height int
	err    error
}

type tickMsg struct{}

func tickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}

type Options struct {
	Root     string
	User     string
	Version  string
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

type SaveFunc func(s settings.Settings) error

var saveHook SaveFunc

func SetSaveHook(f SaveFunc) { saveHook = f }

func (m *Model) persist() {
	if saveHook == nil {
		return
	}
	_ = saveHook(settings.Settings{
		Theme:   m.theme.Name,
		Trusted: m.trusted,
		Provider: m.provider,
		Pet:     m.pet,
		PetName: m.petName,
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
	m.mainVP = viewport.New(m.width, m.vpHeight())
	m.toolPanel = bubbles.NewToolPanel(m.theme)
	m.codeBubble = bubbles.NewCodeBubble(m.theme)
	m.fileTree = bubbles.NewFileTreeBubble(m.theme)
	m.stm32Status = bubbles.NewSTM32StatusBubble(m.theme)
	m.input.Focus()
}

// vpHeight is the usable height for the main scrollable viewport.
// Reserves 2 lines: input + status bar.
func (m *Model) vpHeight() int {
	h := m.height - 2
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
		m.mainVP.Width = msg.Width
		m.mainVP.Height = m.vpHeight()
		m.petRendered = "" // invalidate cache on resize
		return m, nil

	case tea.MouseMsg:
		if m.state == stateChat {
			return m.updateChatMouse(msg)
		}
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
	}
	return m, nil
}

func (m *Model) updateChat(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Ask bubble intercepts all input while waiting for an answer.
	if m.askBubble != nil && !m.askBubble.Answered() {
		return m.updateAsk(msg)
	}

	if !m.popup.Visible() {
		switch key {
		case "pgup", "shift+up":
			m.showingVP = true
			m.mainVP.HalfPageUp()
			return m, nil
		case "pgdown", "shift+down":
			m.mainVP.HalfPageDown()
			if m.mainVP.AtBottom() {
				m.showingVP = false
			}
			return m, nil
		case "esc":
			if m.showingVP {
				m.showingVP = false
				m.mainVP.GotoBottom()
				return m, nil
			}
		}
	}

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

func (m *Model) updateAsk(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	ask := m.askBubble
	key := msg.String()

	if ask.OtherInput().Focused() {
		switch key {
		case "enter":
			if ask.Confirm() {
				return m.submitAskAnswer()
			}
		case "up", "esc":
			ask.MoveUp()
		default:
			ti, c := ask.OtherInput().Update(msg)
			*ask.OtherInput() = ti
			return m, c
		}
		return m, nil
	}

	switch key {
	case "up", "k":
		ask.MoveUp()
	case "down", "j":
		ask.MoveDown()
	case "enter", " ":
		if ask.Confirm() {
			return m.submitAskAnswer()
		}
	}
	return m, nil
}

func (m *Model) submitAskAnswer() (tea.Model, tea.Cmd) {
	ask := m.askBubble
	answer := ask.Answer()
	question := ask.Question()
	m.askBubble = nil
	m.input.Focus()
	payload := "[Answer to: " + question + "]\n" + answer
	m.turn = m.session.Send(m.ctx, payload)
	m.thinking = true
	return m, m.waitEvent()
}

func (m *Model) updateChatMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.MouseWheelUp:
		m.showingVP = true
		m.mainVP.LineUp(3)
	case tea.MouseWheelDown:
		m.mainVP.LineDown(3)
		if m.mainVP.AtBottom() {
			m.showingVP = false
		}
	}
	return m, nil
}

func (m *Model) resetSession() {
	m.session.Reset()
	m.chat = bubbles.NewChat(m.theme)
	m.thought = bubbles.NewThought(m.theme)
	m.toolPanel = bubbles.NewToolPanel(m.theme)
	m.codeBubble = bubbles.NewCodeBubble(m.theme)
	m.fileTree = bubbles.NewFileTreeBubble(m.theme)
	m.stm32Status = bubbles.NewSTM32StatusBubble(m.theme)
	m.petRendered = ""
	m.system = nil
	m.err = nil
	m.showingVP = false
	m.askBubble = nil
	m.todoBubble = nil
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
			m.petRendered = ""
		}
		if m.toolPanel != nil {
			m.toolPanel.SetTheme(m.theme)
		}
		if m.codeBubble != nil {
			m.codeBubble.SetTheme(m.theme)
		}
		if m.fileTree != nil {
			m.fileTree.SetTheme(m.theme)
		}
		if m.stm32Status != nil {
			m.stm32Status.SetTheme(m.theme)
		}
	}
	if res.NewPet != "" {
		m.pet = res.NewPet
		if m.welcome != nil {
			m.welcome.SetPet(pets.ByName(m.pet).Art)
		}
		if m.petBubble != nil {
			m.petBubble.SetArt(pets.ByName(m.pet).Art)
			m.petRendered = ""
		}
	}
	if res.NewPetName != "" {
		m.petName = res.NewPetName
		if m.petBubble != nil {
			m.petBubble.SetName(m.petName)
			m.petRendered = ""
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
		if m.todoBubble != nil {
			m.todoBubble.Handle(ev)
		}

	case agent.TodoEvent:
		m.todoBubble = bubbles.NewTodoBubble(m.theme, e.Items)

	case agent.AskEvent:
		m.askBubble = bubbles.NewAskBubble(m.theme, e.Question, e.Options)
		m.input.Blur()

	case agent.ToolStartEvent:
		m.toolPanel.Handle(ev)
		m.chat.Handle(ev)
		if m.todoBubble != nil {
			m.todoBubble.Handle(ev)
		}
		switch e.Name {
		case "stm32_compile":
			if len(e.Args) > 0 {
				if t, ok := e.Args[0].(string); ok {
					m.stm32Status.SetTarget(t)
				}
			}
			m.stm32Status.CompileStart()
		case "stm32_emulate":
			m.stm32Status.EmulateStart()
		}

	case agent.ToolResultEvent:
		m.toolPanel.Handle(ev)
		m.chat.Handle(ev)
		m.routeToolResult(e)

	case agent.ArtifactEvent:
		m.toolPanel.Handle(ev)

	case agent.UserMessageEvent:
		m.chat.Handle(ev)
		m.thought.Handle(ev)
		m.toolPanel.Handle(ev)
		m.codeBubble.Handle(ev)
		m.fileTree.Handle(ev)
		m.stm32Status.Handle(ev)
		m.petRendered = "" // pet may change phase
		m.showingVP = false
		m.askBubble = nil
		m.todoBubble = nil

	default:
		m.chat.Handle(ev)
		m.thought.Handle(ev)
		if m.todoBubble != nil {
			m.todoBubble.Handle(ev)
		}
	}

	if m.petBubble != nil {
		m.petBubble.Handle(ev)
		m.petRendered = "" // content changed, invalidate
	}
}

func (m *Model) routeToolResult(e agent.ToolResultEvent) {
	result := e.Result
	failed := strings.HasPrefix(result, "ERROR:") ||
		strings.HasPrefix(result, "COMPILE FAILED:") ||
		strings.HasPrefix(result, "QEMU error:")

	switch e.Name {
	case "stm32_compile":
		if failed {
			m.stm32Status.CompileFail(result)
		} else {
			m.stm32Status.CompileOK(result)
		}
	case "stm32_emulate":
		if failed {
			m.stm32Status.EmulateFail(result)
		} else {
			m.stm32Status.EmulateOK(result)
		}
	case "stm32_write_file":
		if !failed {
			if path, content := m.extractWriteFileArgs(); path != "" {
				m.codeBubble.Update(path, content)
				m.fileTree.AddFile(path)
			}
		}
	case "stm32_list_files":
		for _, line := range strings.Split(result, "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "ERROR") {
				m.fileTree.AddFile(line)
			}
		}
	}
}

func (m *Model) extractWriteFileArgs() (path, content string) {
	args := m.toolPanel.LastCallArgs("stm32_write_file")
	if len(args) >= 1 {
		path, _ = args[0].(string)
	}
	if len(args) >= 2 {
		content, _ = args[1].(string)
	}
	return
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

	// Build all chat content into one string for the main viewport.
	var content []string

	if m.petBubble != nil {
		content = append(content, m.petView(w))
	}
	if latest := m.chat.RenderLatest(w); latest != "" {
		content = append(content, latest)
	}
	if m.thought.HasContent() {
		content = append(content, m.thought.View(w))
	}
	if m.todoBubble != nil {
		content = append(content, m.todoBubble.View(w))
	}
	if m.toolPanel != nil && m.toolPanel.HasContent() {
		content = append(content, m.toolPanel.View(w))
	}
	if m.stm32Status != nil && m.stm32Status.HasContent() {
		content = append(content, m.stm32Status.View(w))
	}
	if m.codeBubble != nil && m.codeBubble.HasContent() {
		content = append(content, m.codeBubble.View(w))
	}
	if m.fileTree != nil && m.fileTree.HasContent() {
		content = append(content, m.fileTree.View(w))
	}
	if m.askBubble != nil {
		content = append(content, m.askBubble.View(w))
	}
	for _, s := range m.system {
		content = append(content, sysStyle.Render("» "+s))
	}
	if m.err != nil {
		content = append(content, errStyle.Render("error: "+m.err.Error()))
	}

	// Feed built content into the viewport. If not in manual scroll mode,
	// always snap to the bottom so new content is visible immediately.
	vpContent := strings.Join(content, "\n")
	m.mainVP.SetContent(vpContent)
	if !m.showingVP {
		m.mainVP.GotoBottom()
	}

	var rows []string
	rows = append(rows, m.mainVP.View())

	if m.popup.Visible() {
		rows = append(rows, m.popup.View(m.theme, w))
	}

	// Bottom bar: input + status, or ask hint.
	if m.askBubble != nil && !m.askBubble.Answered() {
		rows = append(rows, hintStyle.Render("↑↓ navigate • enter select • ctrl+c quit"))
	} else {
		trustTag := "trusted"
		if !m.trusted {
			trustTag = "read-only"
		}
		status := "ctrl+c quit • ctrl+r reset • / commands • " + trustTag
		if len(m.chat.Turns()) > 0 {
			if m.showingVP {
				status += " • esc to latest"
			} else {
				status += " • scroll to browse"
			}
		}
		if m.thinking {
			status = busyStyle.Render("● thinking…") + "  " + status
		}
		rows = append(rows, m.input.View(), hintStyle.Render(status))
	}

	return strings.Join(rows, "\n")
}

// petView returns the cached pet bubble render, only re-rendering when the
// animation frame or content has changed. This avoids calling glamour on
// every 80ms tick.
func (m *Model) petView(w int) string {
	// Animation: pet updates every 3 ticks (~240ms). Content changes
	// invalidate m.petRendered directly via route().
	animTick := m.tick / 3
	if m.petRendered == "" || animTick != m.petRenderTick || w != m.petRenderWidth {
		m.petRendered = m.petBubble.View(w, m.tick)
		m.petRenderTick = animTick
		m.petRenderWidth = w
	}
	return m.petRendered
}

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
			return commands.Result{NewPetName: strings.Join(args, " "), Message: "pet renamed → " + strings.Join(args, " ")}
		},
	})

	tt := true
	ff := false
	r.Register(&commands.Command{
		Name:        "trust",
		Description: "trust this directory (allow tool use)",
		Run:         func(args []string) commands.Result { return commands.Result{NewTrusted: &tt, Message: "directory trusted"} },
	})
	r.Register(&commands.Command{
		Name:        "untrust",
		Description: "revoke trust for this directory",
		Run:         func(args []string) commands.Result { return commands.Result{NewTrusted: &ff, Message: "directory untrusted (read-only)"} },
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
		Run:         func(args []string) commands.Result { return commands.Result{ResetSession: true, Message: "session reset"} },
	})
	r.Register(&commands.Command{
		Name:        "clear",
		Description: "clear visible bubbles (keeps session)",
		Run:         func(args []string) commands.Result { return commands.Result{ClearVisual: true} },
	})
	r.Register(&commands.Command{
		Name:        "quit",
		Description: "exit hardcore-ai",
		Run:         func(args []string) commands.Result { return commands.Result{Quit: true} },
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
