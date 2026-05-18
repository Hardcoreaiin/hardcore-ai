package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/config"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/llm"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

func main() {
	prompt := "What is 17 * 23, and then reverse the result as a string?"
	if len(os.Args) > 1 {
		prompt = strings.Join(os.Args[1:], " ")
	}

	reg := tools.NewRegistry()
	tools.RegisterCalculator(reg)
	tools.RegisterStringUtils(reg)

	cfg := config.Load()
	client := buildClient(cfg)
	loop := agent.New(client, reg, agent.Config{})

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	fmt.Println("─── hardcore-ai (phase 1, headless) ───")
	fmt.Println("prompt:", prompt)
	fmt.Println()

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
		}
	}
}

func buildClient(cfg config.LLMConfig) llm.Client {
	switch cfg.Provider {
	case config.ProviderGemini:
		return llm.NewGemini(llm.GeminiConfig{
			URL:    cfg.URL,
			Model:  cfg.Model,
			APIKey: cfg.APIKey,
		})
	default:
		// llamacpp and openrouter both speak the OpenAI chat-completions protocol.
		return llm.NewOpenAI(llm.OpenAIConfig{
			URL:    cfg.URL,
			Model:  cfg.Model,
			APIKey: cfg.APIKey,
		})
	}
}

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
