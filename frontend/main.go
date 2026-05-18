package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"os/user"
	"strings"
	"syscall"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/config"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/llm"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/settings"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

const version = "v0.2.0"

func main() {
	headless := false
	var keep []string
	for _, a := range os.Args[1:] {
		if a == "--headless" {
			headless = true
			continue
		}
		keep = append(keep, a)
	}

	reg := tools.NewRegistry()
	tools.RegisterCalculator(reg)
	tools.RegisterStringUtils(reg)

	cfg := config.Load()
	client := buildClient(cfg)
	loop := agent.New(client, reg, agent.Config{})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if headless {
		prompt := strings.Join(keep, " ")
		if prompt == "" {
			fmt.Fprintln(os.Stderr, "--headless requires a prompt argument")
			os.Exit(2)
		}
		runHeadless(ctx, loop, prompt)
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "cwd error:", err)
		os.Exit(1)
	}

	existing, loaded := loadSettings(cwd)

	// If a provider was saved, rebuild the client before starting the session.
	if loaded && existing.Provider != "" {
		loop.SwapClient(config.LoadForProvider(config.Provider(existing.Provider)).BuildClient())
	}

	tui.SetSaveHook(func(s settings.Settings) error {
		return settings.Save(cwd, s)
	})

	session := loop.NewSession()
	model := tui.New(ctx, session, tui.Options{
		Root:     cwd,
		User:     currentUser(),
		Version:  version,
		Existing: existing,
		Loaded:   loaded,
	})
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tui error:", err)
		os.Exit(1)
	}
}

func loadSettings(cwd string) (settings.Settings, bool) {
	s, err := settings.Load(cwd)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return settings.Settings{}, false
		}
		fmt.Fprintln(os.Stderr, "settings load warning:", err)
		return settings.Settings{}, false
	}
	return s, true
}

func currentUser() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	if n := os.Getenv("USER"); n != "" {
		return n
	}
	return "friend"
}

func runHeadless(ctx context.Context, loop *agent.Loop, prompt string) {
	fmt.Println("─── hardcore-ai (headless) ───")
	fmt.Println("prompt:", prompt)
	for ev := range loop.Run(ctx, prompt) {
		switch e := ev.(type) {
		case agent.ThinkEvent:
			fmt.Printf("\033[2m[think]\033[0m %s\n", e.Text)
		case agent.LineEvent:
			fmt.Println(e.Text)
		case agent.ToolStartEvent:
			fmt.Printf("\033[36m[call]\033[0m  %s(%s)\n", e.Name, formatArgs(e.Args))
		case agent.ToolResultEvent:
			fmt.Printf("\033[36m[result]\033[0m %s → %s\n", e.Name, e.Result)
		case agent.ArtifactEvent:
			fmt.Printf("\033[35m[artifact]\033[0m %s/%s: %v\n", e.From, e.Artifact.Type, e.Artifact.Payload)
		case agent.StepLimitEvent:
			fmt.Printf("\033[33m[limit]\033[0m tool step limit reached (%d)\n", e.Limit)
		case agent.ErrorEvent:
			fmt.Fprintf(os.Stderr, "\033[31m[error]\033[0m %v\n", e.Err)
			os.Exit(1)
		case agent.DoneEvent:
			fmt.Println("\n─── done ───")
			return
		}
	}
}

func buildClient(cfg config.LLMConfig) llm.Client { return cfg.BuildClient() }

func formatArgs(args []any) string {
	parts := make([]string, len(args))
	for i, a := range args {
		switch v := a.(type) {
		case string:
			parts[i] = fmt.Sprintf("%q", v)
		default:
			parts[i] = fmt.Sprintf("%v", v)
		}
	}
	return strings.Join(parts, ", ")
}
