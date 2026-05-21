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
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/bus"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/commands"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/config"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/pets"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/settings"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/toolchain"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools/embedded"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"hardcoreai-rag/indexing"
	"hardcoreai-rag/ingestion"
	"hardcoreai-rag/storage"
)

type state int

const (
	stateOnboarding state = iota
	stateWelcome
	stateChat
	stateUpload
)

type RAGIngestEvent struct {
	Stage string // "start", "progress", "done", "error"
	File  string
	Count int
	Total int
	Err   error
}

type Model struct {
	ctx     context.Context
	session *agent.Session
	db      *storage.DB
	bus     *bus.Bus

	state        state
	uploadBubble *bubbles.UploadBubble
	theme        *theme.Theme
	root         string
	user         string
	version      string
	trusted      bool
	provider     string
	pet          string
	petName      string

	onboard   *onboarding
	welcome   *bubbles.Welcome
	chat      *bubbles.Chat
	thought   *bubbles.Thought
	petBubble *bubbles.PetBubble

	petRendered    string
	petRenderTick  int
	petRenderWidth int

	system    []string
	mainVP    viewport.Model
	showingVP bool

	todoBubble    *bubbles.TodoBubble
	askBubble     *bubbles.AskBubble
	confirmBubble *bubbles.ConfirmBubble
	toolPanel     *bubbles.ToolPanel
	masonry       *MasonryManager

	// confirmReqs carries shell-command approval requests from the agent's
	// bash tool to the TUI. The agent goroutine blocks until the user decides.
	confirmReqs    chan confirmRequest
	pendingConfirm *confirmRequest

	cmds  *commands.Registry
	popup *cmdPopup

	input     textinput.Model
	turn      <-chan agent.Event
	appEvents <-chan any
	thinking  bool
	tick      int
	ticking   bool // true once a tickCmd chain is running — prevents duplicates

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

// startTick begins the animation tick chain, but only once. Spawning a second
// chain doubles the render rate every time it happens; left unchecked across
// state transitions it compounds into a render storm. Returns nil if a chain
// is already running.
func (m *Model) startTick() tea.Cmd {
	if m.ticking {
		return nil
	}
	m.ticking = true
	return tickCmd()
}

type Options struct {
	Root     string
	User     string
	Version  string
	Events   <-chan any
	Bus      *bus.Bus
	RAGDB    *storage.DB
	Existing settings.Settings
	Loaded   bool
}

func New(ctx context.Context, session *agent.Session, opts Options) *Model {
	ti := textinput.New()
	ti.Placeholder = "ask anything…  (/ for commands, enter to send, ctrl+c to quit)"
	ti.Prompt = "› "
	ti.CharLimit = 4000

	m := &Model{
		ctx:         ctx,
		session:     session,
		db:          opts.RAGDB,
		bus:         opts.Bus,
		root:        opts.Root,
		user:        opts.User,
		version:     opts.Version,
		appEvents:   opts.Events,
		input:       ti,
		width:       80,
		height:      24,
		confirmReqs: make(chan confirmRequest),
	}

	// Install the bash confirmation gate: the agent's bash tool calls this
	// from its goroutine, which blocks until the user answers in the TUI.
	embedded.SetBashConfirmFunc(func(command, dir string) bool {
		reply := make(chan bool, 1)
		select {
		case m.confirmReqs <- confirmRequest{command: command, dir: dir, reply: reply}:
		case <-ctx.Done():
			return false
		}
		select {
		case ok := <-reply:
			return ok
		case <-ctx.Done():
			return false
		}
	})

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

// confirmRequest is a shell-command approval request. The agent's bash tool
// fills Command/Dir, sends the request to the TUI, then blocks on Reply until
// the user decides.
type confirmRequest struct {
	command string
	dir     string
	reply   chan bool
}

type confirmReqMsg struct{ req confirmRequest }

func (m *Model) waitConfirm() tea.Cmd {
	ch := m.confirmReqs
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		req, ok := <-ch
		if !ok {
			return nil
		}
		return confirmReqMsg{req: req}
	}
}

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
		return tea.Batch(m.waitAppEvent(), m.waitConfirm())
	}
	return tea.Batch(textinput.Blink, m.startTick(), m.waitAppEvent(), m.waitConfirm())
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
			return m, tea.Batch(textinput.Blink, m.startTick())
		case stateChat:
			return m.updateChat(msg)
		case stateUpload:
			return m.updateUpload(msg)
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

	case confirmReqMsg:
		req := msg.req
		// A confirm is already on screen — reject the new request immediately
		// rather than orphaning its goroutine. waitConfirm is NOT re-armed here;
		// resolveConfirm re-arms it once the current request is decided, so at
		// most one request is in flight at a time.
		if m.pendingConfirm != nil {
			req.reply <- false
			return m, m.waitConfirm()
		}
		m.pendingConfirm = &req
		m.confirmBubble = bubbles.NewConfirmBubble(m.theme, req.command, req.dir)
		if m.state == stateChat {
			m.input.Blur()
		}
		return m, nil
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

func (m *Model) updateUpload(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.uploadBubble == nil {
		m.state = stateChat
		return m, nil
	}
	key := msg.String()
	done, confirmed, files := m.uploadBubble.HandleKey(key)
	if done {
		m.state = stateChat
		m.uploadBubble = nil
		if confirmed {
			if len(files) == 0 {
				m.system = append(m.system, "RAG upload cancelled (no files selected)")
			} else {
				m.system = append(m.system, fmt.Sprintf("Confirmed upload of %d files. Ingesting in background...", len(files)))
				go m.ingestFiles(files)
			}
		} else {
			m.system = append(m.system, "RAG upload cancelled")
		}
	}
	return m, nil
}

func (m *Model) updateChat(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Confirm bubble intercepts all input while a command awaits approval.
	if m.confirmBubble != nil && !m.confirmBubble.Decided() {
		return m.updateConfirm(msg)
	}

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
		case "esc":
			// Unfocus any focused bubble and hand control back to the input.
			if m.masonry.AnyFocused() {
				m.masonry.ClearFocus()
				return m, nil
			}
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

func (m *Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cb := m.confirmBubble
	switch msg.String() {
	case "left", "h":
		cb.MoveLeft()
	case "right", "l", "tab":
		cb.MoveRight()
	case "y", "Y":
		cb.SetApproved(true)
		return m.resolveConfirm()
	case "n", "N", "esc":
		cb.SetApproved(false)
		return m.resolveConfirm()
	case "enter", " ":
		cb.Confirm()
		return m.resolveConfirm()
	}
	return m, nil
}

// resolveConfirm sends the user's decision back to the blocked bash tool and
// dismisses the confirm bubble. It re-arms waitConfirm so the next queued
// request can be received — this is the only place waitConfirm is re-armed
// while a request was pending, which bounds in-flight confirms to one.
func (m *Model) resolveConfirm() (tea.Model, tea.Cmd) {
	cb := m.confirmBubble
	if cb == nil || !cb.Decided() {
		return m, nil
	}
	if m.pendingConfirm != nil {
		m.pendingConfirm.reply <- cb.Approved()
		m.pendingConfirm = nil
	}
	m.confirmBubble = nil
	if m.state == stateChat {
		m.input.Focus()
	}
	return m, m.waitConfirm()
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
	// Release any blocked bash goroutine so it doesn't leak on reset.
	if m.pendingConfirm != nil {
		m.pendingConfirm.reply <- false
		m.pendingConfirm = nil
	}
	m.confirmBubble = nil
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
	if res.StartUpload {
		m.uploadBubble = bubbles.NewUploadBubble(m.theme, m.root)
		m.state = stateUpload
	}
	if res.ClearRAGDB {
		if m.db != nil {
			m.db.Close()
		}
		dbPath, err := config.RAGDBPath()
		if err != nil {
			m.system = append(m.system, "error resolving RAG DB path: "+err.Error())
		} else {
			os.Remove(dbPath)
			os.Remove(dbPath + "-wal")
			os.Remove(dbPath + "-shm")

			newDB, err := storage.NewDB(dbPath)
			if err != nil {
				m.system = append(m.system, "error re-initializing database: "+err.Error())
			} else {
				if m.db != nil {
					m.db.DB = newDB.DB
					m.db.HasFTS5 = newDB.HasFTS5
				} else {
					m.db = newDB
				}
				m.system = append(m.system, "✅ RAG database successfully cleared and re-initialized.")
			}
		}
	}
	if res.NewActiveDir != "" && m.masonry != nil {
		project := filepath.Base(res.NewActiveDir)
		m.masonry.FileTree().SetProject(project)
		m.masonry.BuildStatus().SetProject(project)
	}
	if res.SpawnBubble != "" && m.masonry != nil {
		switch res.SpawnBubble {
		case "code":
			m.masonry.NewCodeBubble()
		case "filetree":
			m.masonry.FileTree()
		case "console":
			m.masonry.Console()
		case "buildstatus":
			m.masonry.BuildStatus()
		}
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
			// emitCodeFences re-scans the whole assistant buffer every loop
			// step, so the same fence re-fires each step. Only spawn a new
			// bubble for genuinely new content — otherwise the masonry would
			// accumulate an unbounded stack of identical code bubbles.
			if latest := m.masonry.LatestCode(); latest != nil && latest.Content() == e.Content {
				break
			}
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
	case "file_diff":
		// file_edit emits the before/after content of the file it patched.
		// Render it in its own code bubble as a diff so the precise change
		// is visible, just like a file_write.
		if d, ok := e.Artifact.Payload.(embedded.FileDiff); ok {
			code := m.masonry.NewCodeBubble()
			code.UpdateDiff(d.Path, d.New, d.Old)
			m.masonry.FileTree().AddFile(d.Path)
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
	case userToolResult:
		if e.err != nil {
			m.system = append(m.system, fmt.Sprintf("» tool %s failed: %s", e.name, sanitizeError(e.err)))
		} else {
			out := strings.TrimSpace(e.result)
			if out == "" {
				out = "(no output)"
			}
			m.system = append(m.system, fmt.Sprintf("» tool %s →\n%s", e.name, out))
		}
	case RAGIngestEvent:
		switch e.Stage {
		case "start":
			m.system = append(m.system, fmt.Sprintf("Starting ingestion of %d files...", e.Total))
		case "progress":
			m.system = append(m.system, fmt.Sprintf("[%d/%d] Successfully indexed: %s", e.Count, e.Total, e.File))
		case "done":
			m.system = append(m.system, fmt.Sprintf("✅ RAG ingestion complete! %d files are now indexed and searchable.", e.Total))
		case "error":
			m.system = append(m.system, fmt.Sprintf("❌ RAG ingestion error: %s", sanitizeError(e.Err)))
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
	case "workspace_init", "cd":
		bs := m.masonry.BuildStatus()
		ft := m.masonry.FileTree()
		if len(e.Args) > 0 {
			if name, ok := e.Args[0].(string); ok {
				project := filepath.Base(filepath.Clean(name))
				bs.SetProject(project)
				ft.SetProject(project)
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
	case "workspace_init", "cd":
		if !failed {
			if dir := parseActivePath(result); dir != "" {
				project := filepath.Base(dir)
				m.masonry.FileTree().SetProject(project)
				m.masonry.BuildStatus().SetProject(project)
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

// parseActivePath extracts the directory from a workspace_init/cd result line
// like "active project: /path" or "active directory: /path".
func parseActivePath(result string) string {
	for _, line := range strings.Split(result, "\n") {
		line = strings.TrimSpace(line)
		for _, prefix := range []string{"active project:", "active directory:"} {
			if strings.HasPrefix(line, prefix) {
				return strings.TrimSpace(strings.TrimPrefix(line, prefix))
			}
		}
	}
	return ""
}

func parseWorkspaceCurrent(result string) string {
	for _, line := range strings.Split(result, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "current directory:") {
			rest := strings.TrimSpace(strings.TrimPrefix(line, "current directory:"))
			if rest == "" {
				return ""
			}
			return filepath.Base(rest)
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
	case stateUpload:
		if m.uploadBubble != nil {
			return m.uploadBubble.View(m.width, m.height)
		}
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
		if m.confirmBubble != nil {
			left = append(left, m.confirmBubble.View(leftW))
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
		if m.confirmBubble != nil {
			content = append(content, m.confirmBubble.View(w))
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

	if m.confirmBubble != nil && !m.confirmBubble.Decided() {
		rows = append(rows, hintStyle.Render("←→ move • enter confirm • y approve • n reject • ctrl+c quit"))
	} else if m.askBubble != nil && !m.askBubble.Answered() {
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
		Name:        "upload",
		Description: "browse and upload PDF manuals to local RAG database",
		Run: func(args []string) commands.Result {
			return commands.Result{StartUpload: true}
		},
	})
	r.Register(&commands.Command{
		Name:        "clear-db",
		Description: "completely clear and re-initialize the local RAG database",
		Run: func(args []string) commands.Result {
			return commands.Result{ClearRAGDB: true}
		},
	})
	r.Register(&commands.Command{
		Name:        "cd",
		Description: "change the active directory (absolute, ~-relative, or relative path)",
		Run: func(args []string) commands.Result {
			if len(args) == 0 {
				return commands.Result{Message: "current directory: " + embedded.CurrentDir() + " (usage: /cd <path>)"}
			}
			dir, err := embedded.ChangeDir(strings.Join(args, " "))
			if err != nil {
				return commands.Result{Err: err}
			}
			return commands.Result{NewActiveDir: dir, Message: "active directory → " + dir}
		},
	})

	r.Register(&commands.Command{
		Name:        "tool",
		Description: "run an agent tool directly: /tool <name> <arg1> <arg2> …",
		ArgValues:   func() []string { return m.session.ToolNames() },
		Run: func(args []string) commands.Result {
			if len(args) == 0 {
				return commands.Result{Message: "tools: " + strings.Join(m.session.ToolNames(), ", ") + " (usage: /tool <name> <args>)"}
			}
			go m.runToolDirect(args[0], args[1:])
			return commands.Result{Message: "running tool: " + args[0] + " …"}
		},
	})

	bubbleKinds := []string{"code", "filetree", "console", "buildstatus"}
	r.Register(&commands.Command{
		Name:        "bubble",
		Description: "spawn a masonry bubble: code, filetree, console, buildstatus",
		ArgValues:   func() []string { return bubbleKinds },
		Run: func(args []string) commands.Result {
			if len(args) == 0 {
				return commands.Result{Message: "bubble kinds: " + strings.Join(bubbleKinds, ", ") + " (usage: /bubble <kind>)"}
			}
			kind := args[0]
			for _, k := range bubbleKinds {
				if k == kind {
					return commands.Result{SpawnBubble: kind, Message: "spawned bubble: " + kind}
				}
			}
			return commands.Result{Err: fmt.Errorf("unknown bubble kind: %s", kind)}
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

func (m *Model) ingestFiles(files []string) {
	if m.bus == nil || m.db == nil {
		return
	}

	dbPath, err := config.RAGDBPath()
	if err != nil {
		m.bus.Publish(RAGIngestEvent{Stage: "error", Err: fmt.Errorf("failed to get global RAG DB path: %w", err)})
		return
	}

	m.bus.Publish(RAGIngestEvent{Stage: "start", Total: len(files)})

	embedder := indexing.NewEmbedder()
	indexer, err := indexing.NewIndexer(dbPath, embedder)
	if err != nil {
		m.bus.Publish(RAGIngestEvent{Stage: "error", Err: fmt.Errorf("failed to initialize indexer: %w", err)})
		return
	}
	defer indexer.Close()

	parser := ingestion.NewPDFParser()
	chunker := ingestion.NewChunker()

	for i, localPath := range files {
		filename := filepath.Base(localPath)

		// Metadata inference
		docType := "document"
		lowerName := strings.ToLower(filename)
		if strings.Contains(lowerName, "reference_manual") || strings.Contains(lowerName, "rm") {
			docType = "reference_manual"
		} else if strings.Contains(lowerName, "datasheet") || strings.Contains(lowerName, "ds") {
			docType = "datasheet"
		} else if strings.Contains(lowerName, "programming_manual") || strings.Contains(lowerName, "pm") {
			docType = "programming_manual"
		}

		chipFamily := "STM32"
		if strings.Contains(lowerName, "stm32f4") {
			chipFamily = "STM32F4"
		} else if strings.Contains(lowerName, "stm32f7") {
			chipFamily = "STM32F7"
		} else if strings.Contains(lowerName, "stm32h7") {
			chipFamily = "STM32H7"
		}

		chipModel := chipFamily
		words := strings.FieldsFunc(lowerName, func(r rune) bool {
			return r == '_' || r == '-' || r == '.' || r == ' '
		})
		for _, w := range words {
			if strings.HasPrefix(w, "stm32") && len(w) > 7 {
				chipModel = strings.ToUpper(w)
				break
			}
		}

		version := "v1.0"
		for _, w := range words {
			if (strings.HasPrefix(w, "rm") || strings.HasPrefix(w, "pm")) && len(w) == 6 {
				version = strings.ToUpper(w)
				break
			}
		}

		// Delete any existing document with this ID to handle clean overwrites/re-indexing.
		if err := m.db.DeleteDocument("local_" + filename); err != nil {
			m.bus.Publish(RAGIngestEvent{Stage: "error", Err: fmt.Errorf("failed to clean existing document %s: %w", filename, err)})
			continue
		}

		// Parse PDF
		doc, err := parser.ParsePDF(localPath)
		if err != nil {
			m.bus.Publish(RAGIngestEvent{Stage: "error", Err: fmt.Errorf("failed to parse %s: %w", filename, err)})
			continue
		}

		// Insert document record
		docID, err := m.db.InsertDocument(storage.Document{
			MongoID:    "local_" + filename,
			Filename:   filename,
			LocalPath:  localPath,
			DocType:    docType,
			ChipFamily: chipFamily,
			ChipModel:  chipModel,
			Version:    version,
		})
		if err != nil {
			m.bus.Publish(RAGIngestEvent{Stage: "error", Err: fmt.Errorf("failed to insert document %s: %w", filename, err)})
			continue
		}

		// Chunk
		ingestionChunks := chunker.ChunkDocument(doc)

		// Convert to storage chunks
		storageChunks := make([]storage.Chunk, len(ingestionChunks))
		for idx, c := range ingestionChunks {
			storageChunks[idx] = storage.Chunk{
				DocumentID:   int(docID),
				ChunkText:    c.ChunkText,
				SectionTitle: c.SectionTitle,
				Peripheral:   c.Peripheral,
				RegisterName: c.RegisterName,
				PageNumber:   c.PageNumber,
				TokenCount:   c.TokenCount,
				ChunkIndex:   c.ChunkIndex,
			}
		}

		// Insert chunks
		chunkIDs, err := m.db.InsertChunksAndReturnIDs(storageChunks)
		if err != nil {
			m.db.UpdateDocumentStatus(docID, "failed", err.Error())
			m.bus.Publish(RAGIngestEvent{Stage: "error", Err: fmt.Errorf("failed to insert chunks for %s: %w", filename, err)})
			continue
		}

		// Extract texts
		chunkTexts := make([]string, len(ingestionChunks))
		for idx, c := range ingestionChunks {
			chunkTexts[idx] = c.ChunkText
		}

		// Index embeddings
		if err := indexer.IndexChunks(chunkIDs, chunkTexts); err != nil {
			m.db.UpdateDocumentStatus(docID, "failed", err.Error())
			m.bus.Publish(RAGIngestEvent{Stage: "error", Err: fmt.Errorf("failed to index embeddings for %s: %w", filename, err)})
			continue
		}

		m.db.UpdateDocumentStatus(docID, "indexed", "")
		m.bus.Publish(RAGIngestEvent{Stage: "progress", File: filename, Count: i + 1, Total: len(files)})
	}

	m.bus.Publish(RAGIngestEvent{Stage: "done", Total: len(files)})
}

// userToolResult carries the outcome of a user-invoked tool (/tool) back to
// the TUI so it can be rendered as a system line.
type userToolResult struct {
	name   string
	result string
	err    error
}

// runToolDirect executes a tool on behalf of the user (via /tool) off the UI
// goroutine, then publishes the result. String args are passed as-is; the
// tool's reflection layer coerces them to the right types.
func (m *Model) runToolDirect(name string, args []string) {
	tokens := make([]any, len(args))
	for i, a := range args {
		tokens[i] = a
	}
	result, _, err := m.session.RunTool(m.ctx, name, tokens)
	if m.bus != nil {
		m.bus.Publish(userToolResult{name: name, result: result, err: err})
	}
}

func sanitizeError(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	var sb strings.Builder
	for _, r := range s {
		if r >= 32 && r < 127 {
			sb.WriteRune(r)
		} else if r == '\n' || r == '\t' || r == '\r' {
			sb.WriteRune(' ')
		} else {
			sb.WriteRune('?')
		}
	}
	return sb.String()
}
