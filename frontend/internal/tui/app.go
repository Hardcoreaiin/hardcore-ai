// Package tui is the Bubble Tea root model.
package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/bubbles"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/commands"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/config"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/pets"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/settings"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/toolchain"
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

	petRendered    string
	petRenderTick  int
	petRenderWidth int

	system    []string
	mainVP    viewport.Model
	showingVP bool

	todoBubble *bubbles.TodoBubble
	askBubble  *bubbles.AskBubble
	toolPanel  *bubbles.ToolPanel
	masonry    *MasonryManager

	cmds  *commands.Registry
	popup *cmdPopup

	input     textinput.Model
	turn      <-chan agent.Event
	appEvents <-chan any
	thinking  bool
	tick      int

	// live token stream ticker
	streamTicker string // latest token text, trimmed to one line
	streamTokens int    // total tokens seen this turn

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
	Events   <-chan any
	Existing settings.Settings
	Loaded   bool
}

func New(ctx context.Context, session *agent.Session, opts Options) *Model {
	ti := textinput.New()
	ti.Placeholder = "ask anything…  (/ for commands, enter to send, ctrl+c to quit)"
	ti.Prompt = "› "
	ti.CharLimit = 4000

	m := &Model{
		ctx:       ctx,
		session:   session,
		root:      opts.Root,
		user:      opts.User,
		version:   opts.Version,
		appEvents: opts.Events,
		input:     ti,
		width:     80,
		height:    24,
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
type appEventMsg struct{ ev any }
type streamClosedMsg struct{}
type appEventsClosedMsg struct{}

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

func (m *Model) waitAppEvent() tea.Cmd {
	ch := m.appEvents
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return appEventsClosedMsg{}
		}
		return appEventMsg{ev: ev}
	}
}

func (m *Model) Init() tea.Cmd {
	if m.state == stateOnboarding {
		return m.waitAppEvent()
	}
	return tea.Batch(textinput.Blink, tickCmd(), m.waitAppEvent())
}

type SaveFunc func(s settings.Settings) error

var saveHook SaveFunc

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
	m.mainVP = viewport.New(m.width, m.vpHeight())
	m.toolPanel = bubbles.NewToolPanel(m.theme)
	m.masonry = NewMasonryManager(m.theme)
	m.input.Focus()
}

func (m *Model) vpHeight() int {
	// fixed rows below viewport: input (1) + hint/status (1) + stream ticker (1)
	h := m.height - 3
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
		m.petRendered = ""
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

	case appEventsClosedMsg:
		m.appEvents = nil
		return m, nil

	case eventMsg:
		m.route(msg.ev)
		if _, ok := msg.ev.(agent.TurnEndEvent); ok {
			m.thinking = false
		}
		return m, m.waitEvent()

	case appEventMsg:
		m.routeAppEvent(msg.ev)
		return m, m.waitAppEvent()
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

	// Ask bubble intercepts all input while active.
	if m.askBubble != nil && !m.askBubble.Answered() {
		return m.updateAsk(msg)
	}

	// Masonry focus / dismiss — only when popup is hidden and not thinking.
	if !m.popup.Visible() && m.masonry != nil && m.masonry.HasContent() {
		switch key {
		case "tab":
			m.masonry.FocusNext()
			return m, nil
		case "shift+tab":
			m.masonry.FocusPrev()
			return m, nil
		case "x":
			// Only intercept when a bubble is explicitly focused via tab.
			if m.masonry.AnyFocused() && m.masonry.DismissFocused() {
				return m, nil
			}
		case "-", "m":
			// Only intercept when a bubble is explicitly focused via tab.
			if m.masonry.AnyFocused() && m.masonry.ToggleMinimizeFocused() {
				return m, nil
			}
		default:
			if m.masonry.UpdateFocused(msg) {
				ft := m.masonry.FileTree()
				if ft.OnSelect == nil {
					ft.OnSelect = func(path string) {
						full := filepath.Join(m.root, filepath.FromSlash(path))
						if b, err := os.ReadFile(full); err == nil {
							m.masonry.Code().Update(path, string(b))
						}
					}
				}
				return m, nil
			}
		}
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
		// Return to input clears masonry focus.
		if m.masonry != nil {
			m.masonry.ClearFocus()
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

func (m *Model) resetSession() {
	m.session.Reset()
	m.chat = bubbles.NewChat(m.theme)
	m.thought = bubbles.NewThought(m.theme)
	m.toolPanel = bubbles.NewToolPanel(m.theme)
	if m.masonry != nil {
		m.masonry.Reset()
	}
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
		if m.masonry != nil {
			m.masonry.SetTheme(m.theme)
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
		if m.todoBubble != nil {
			m.todoBubble.Handle(ev)
		}
		if m.masonry != nil {
			m.masonry.AddToolCall(e)
		}
		m.routeToolStart(e)

	case agent.ToolResultEvent:
		m.toolPanel.Handle(ev)
		if m.masonry != nil {
			m.masonry.ApplyToolResult(e)
		}
		m.routeToolResult(e)

	case agent.CodeFenceEvent:
		if m.masonry != nil {
			code := m.masonry.NewCodeBubble()
			code.UpdateFence(e.Lang, e.Content)
		}

	case agent.ArtifactEvent:
		m.toolPanel.Handle(ev)
		if m.masonry != nil {
			m.masonry.ApplyArtifact(e)
			m.routeArtifact(e)
		}

	case agent.UserMessageEvent:
		m.chat.Handle(ev)
		m.thought.Handle(ev)
		m.toolPanel.Handle(ev)
		m.petRendered = ""
		m.showingVP = false
		m.askBubble = nil
		m.todoBubble = nil
		m.streamTicker = ""
		m.streamTokens = 0

	default:
		m.chat.Handle(ev)
		m.thought.Handle(ev)
		if m.todoBubble != nil {
			m.todoBubble.Handle(ev)
		}
		// Update stream ticker on every raw token.
		if tok, ok := ev.(agent.TokenEvent); ok {
			m.updateStreamTicker(tok.Text)
		}
	}

	if m.petBubble != nil {
		m.petBubble.Handle(ev)
		m.petRendered = ""
	}
}

func (m *Model) routeArtifact(e agent.ArtifactEvent) {
	if m.masonry == nil {
		return
	}
	switch e.Artifact.Type {
	case "project_dir":
		if dir, ok := e.Artifact.Payload.(string); ok {
			project := filepath.Base(filepath.Clean(dir))
			if project != "." && project != string(filepath.Separator) {
				m.masonry.FileTree().SetProject(project)
				m.masonry.BuildStatus().SetProject(project)
			}
		}
	}
}

func (m *Model) routeAppEvent(ev any) {
	switch e := ev.(type) {
	case toolchain.Event:
		if m.masonry == nil {
			return
		}
		m.masonry.BuildStatus().ToolchainEvent(e)
		if e.Stage == "download" || e.Stage == "extract" {
			m.masonry.Console().Start("toolchain")
		}
		if e.Stage == "ready" {
			m.masonry.Console().Done("toolchain ready: "+string(e.Tool), false)
		}
		if e.Stage == "error" && e.Err != nil {
			m.masonry.Console().Done(e.Err.Error(), true)
		}
	}
}

func (m *Model) routeToolStart(e agent.ToolStartEvent) {
	if m.masonry == nil {
		return
	}
	switch e.Name {
	case "build":
		bs := m.masonry.BuildStatus()
		con := m.masonry.Console()
		if len(e.Args) > 0 {
			if t, ok := e.Args[0].(string); ok {
				bs.SetTarget(t)
			}
		}
		bs.BuildStart()
		con.Start("build")
	case "emulate":
		m.masonry.BuildStatus().EmulateStart()
		m.masonry.Console().Start("emulate")
	case "flash":
		m.masonry.Console().Start("flash")
	case "workspace_init":
		bs := m.masonry.BuildStatus()
		ft := m.masonry.FileTree()
		if len(e.Args) > 0 {
			if name, ok := e.Args[0].(string); ok {
				bs.SetProject(name)
				ft.SetProject(name)
			}
		}
	case "workspace_status":
		m.masonry.FileTree()
	case "file_write":
		// Each file_write gets its OWN CodeBubble so writes appear as
		// separate bubbles in the masonry panel.
		code := m.masonry.NewCodeBubble()
		ft := m.masonry.FileTree()
		if len(e.Args) > 0 {
			if path, ok := e.Args[0].(string); ok {
				ft.AddFile(path)
				if len(e.Args) > 1 {
					if content, ok := e.Args[1].(string); ok {
						oldContent := ""
						if m.root != "" {
							full := filepath.Join(m.root, filepath.FromSlash(path))
							if b, err := os.ReadFile(full); err == nil {
								oldContent = string(b)
							}
						}
						code.UpdateDiff(path, content, oldContent)
					}
				}
			}
		}
	case "file_list":
		m.masonry.FileTree() // ensure visible
	}
}

func (m *Model) routeToolResult(e agent.ToolResultEvent) {
	result := e.Result
	failed := strings.HasPrefix(result, "ERROR:") ||
		strings.HasPrefix(result, "BUILD FAILED:") ||
		strings.HasPrefix(result, "FLASH FAILED:") ||
		strings.HasPrefix(result, "QEMU error:")

	if m.masonry == nil {
		return
	}
	switch e.Name {
	case "build":
		bs := m.masonry.BuildStatus()
		con := m.masonry.Console()
		if strings.Contains(result, "toolchain is downloading") {
			bs.BuildStart()
			con.Start("toolchain")
		} else if failed {
			bs.BuildFail(result)
			con.Done(result, true)
		} else {
			bs.BuildOK(result)
			con.Done(result, false)
		}
	case "emulate":
		bs := m.masonry.BuildStatus()
		con := m.masonry.Console()
		if failed {
			bs.EmulateFail(result)
			con.Done(result, true)
		} else {
			bs.EmulateOK(result)
			con.Done(result, false)
		}
	case "flash":
		m.masonry.Console().Done(result, failed)
	case "file_write":
		if !failed {
			if path, content := m.extractWriteFileArgs(); path != "" {
				oldContent := ""
				if m.root != "" {
					full := filepath.Join(m.root, filepath.FromSlash(path))
					if b, err := os.ReadFile(full); err == nil {
						oldContent = string(b)
					}
				}
				// Update the most recent code bubble (created in routeToolStart).
				if code := m.masonry.LatestCode(); code != nil {
					code.UpdateDiff(path, content, oldContent)
				}
				m.masonry.FileTree().AddFile(path)
			}
		}
	case "file_list":
		ft := m.masonry.FileTree()
		for _, line := range strings.Split(result, "\n") {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "(") && !strings.HasPrefix(line, "ERROR") {
				ft.AddFile(line)
			}
		}
	case "workspace_init":
		if !failed {
			args := m.toolPanel.LastCallArgs("workspace_init")
			if len(args) > 0 {
				if name, ok := args[0].(string); ok {
					m.masonry.FileTree().SetProject(name)
					m.masonry.BuildStatus().SetProject(name)
				}
			}
		}
	case "workspace_status":
		ft := m.masonry.FileTree()
		if current := parseWorkspaceCurrent(result); current != "" {
			ft.SetProject(current)
			m.masonry.BuildStatus().SetProject(current)
		}
	}
}

func parseWorkspaceCurrent(result string) string {
	for _, line := range strings.Split(result, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "current project:") {
			rest := strings.TrimSpace(strings.TrimPrefix(line, "current project:"))
			if rest == "" {
				return ""
			}
			return strings.Fields(rest)[0]
		}
	}
	return ""
}

func (m *Model) extractWriteFileArgs() (path, content string) {
	args := m.toolPanel.LastCallArgs("file_write")
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
		content := lipgloss.JoinVertical(lipgloss.Center, m.welcome.View(m.width), hint)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}
	return m.chatView()
}

func (m *Model) chatView() string {
	w := m.width
	t := m.theme
	hintStyle := lipgloss.NewStyle().Foreground(t.Muted)
	errStyle := lipgloss.NewStyle().Foreground(t.ErrorFg).Bold(true)
	busyStyle := lipgloss.NewStyle().Foreground(t.WarnFg)
	tickStyle := lipgloss.NewStyle().Foreground(t.Muted).Italic(true)
	sysStyle := lipgloss.NewStyle().Foreground(t.Accent).Italic(true)

	hasRight := m.masonry != nil && m.masonry.HasContent()
	const minRightWidth = 42
	useTwoCol := hasRight && w >= minRightWidth+30

	// The viewport holds the full scrollable history: all chat turns + live
	// bubbles. This makes pgup/pgdown scroll through everything.
	var vpContent string
	if useTwoCol {
		rightW := w * 60 / 100
		if rightW < minRightWidth {
			rightW = minRightWidth
		}
		leftW := w - rightW - 1

		var left []string
		// Full chat history (all turns with assistant prose).
		if all := m.chat.RenderAll(leftW); all != "" {
			left = append(left, all)
		}
		// Live per-turn bubbles appended below history.
		if m.thought.HasContent() {
			left = append(left, m.thought.View(leftW))
		}
		if m.todoBubble != nil {
			left = append(left, m.todoBubble.View(leftW))
		}
		if m.askBubble != nil {
			left = append(left, m.askBubble.View(leftW))
		}
		for _, s := range m.system {
			left = append(left, sysStyle.Render("» "+s))
		}
		if m.err != nil {
			left = append(left, errStyle.Render("error: "+m.err.Error()))
		}
		// Pet pinned at the bottom of the left column.
		if m.petBubble != nil {
			left = append(left, m.petView(leftW))
		}

		leftStr := strings.Join(left, "\n")
		rightStr := m.masonry.View(rightW, m.tick)

		vpContent = lipgloss.JoinHorizontal(lipgloss.Bottom,
			lipgloss.NewStyle().Width(leftW).Render(leftStr),
			" ",
			lipgloss.NewStyle().Width(rightW).Render(rightStr),
		)
	} else {
		var content []string
		if all := m.chat.RenderAll(w); all != "" {
			content = append(content, all)
		}
		if m.thought.HasContent() {
			content = append(content, m.thought.View(w))
		}
		if m.todoBubble != nil {
			content = append(content, m.todoBubble.View(w))
		}
		if hasRight {
			content = append(content, m.masonry.View(w, m.tick))
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
		if m.petBubble != nil {
			content = append(content, m.petView(w))
		}
		vpContent = strings.Join(content, "\n")
	}

	m.mainVP.SetContent(vpContent)
	if !m.showingVP {
		m.mainVP.GotoBottom()
	}

	var rows []string
	rows = append(rows, m.mainVP.View())

	if m.popup.Visible() {
		rows = append(rows, m.popup.View(m.theme, w))
	}

	if m.askBubble != nil && !m.askBubble.Answered() {
		rows = append(rows, hintStyle.Render("↑↓ navigate • enter select • ctrl+c quit"))
	} else {
		trustTag := "trusted"
		if !m.trusted {
			trustTag = "read-only"
		}
		status := "ctrl+c quit • ctrl+r reset • / commands • tab focus • x dismiss • " + trustTag
		if len(m.chat.Turns()) > 0 {
			if m.showingVP {
				status += " • esc to latest"
			} else {
				status += " • pgup/pgdn scroll"
			}
		}
		if m.thinking {
			status = busyStyle.Render("● thinking…") + "  " + status
		}
		rows = append(rows, m.input.View(), hintStyle.Render(status))
	}

	// Stream ticker: one rolling line showing live tokens + count.
	// Shown whenever a turn is in flight; blank otherwise.
	if m.thinking && m.streamTicker != "" {
		maxTok := w - 16
		if maxTok < 10 {
			maxTok = 10
		}
		tok := m.streamTicker
		if len([]rune(tok)) > maxTok {
			runes := []rune(tok)
			tok = "…" + string(runes[len(runes)-maxTok:])
		}
		tickerLine := fmt.Sprintf("[%4d tok] %s", m.streamTokens, tok)
		rows = append(rows, tickStyle.Render(tickerLine))
	} else if m.thinking {
		rows = append(rows, tickStyle.Render("waiting for tokens…"))
	} else {
		rows = append(rows, "") // keep height stable
	}

	return strings.Join(rows, "\n")
}

func (m *Model) updateStreamTicker(text string) {
	if text == "" {
		return
	}
	m.streamTokens++
	// Collapse whitespace to spaces so the ticker stays single-line.
	flat := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		return r
	}, text)
	// Append to rolling buffer, capped at 500 chars.
	m.streamTicker += flat
	if len(m.streamTicker) > 500 {
		m.streamTicker = m.streamTicker[len(m.streamTicker)-500:]
	}
}

func (m *Model) petView(w int) string {
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
