package config

import (
	"bufio"
	"os"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/llm"
)

type Provider string

const (
	ProviderLlamaCpp   Provider = "llamacpp"
	ProviderOpenRouter Provider = "openrouter"
	ProviderGemini     Provider = "gemini"
)

type LLMConfig struct {
	Provider Provider
	URL      string
	Model    string
	APIKey   string
}

// BuildClient constructs the llm.Client for this config.
func (c LLMConfig) BuildClient() llm.Client {
	switch c.Provider {
	case ProviderGemini:
		return llm.NewGemini(llm.GeminiConfig{URL: c.URL, Model: c.Model, APIKey: c.APIKey})
	default:
		return llm.NewOpenAI(llm.OpenAIConfig{URL: c.URL, Model: c.Model, APIKey: c.APIKey})
	}
}

func Load() LLMConfig {
	loadDotEnv(".env")
	return LoadForProvider(Provider(getenv("LLM_PROVIDER", string(ProviderLlamaCpp))))
}

// LoadForProvider returns an LLMConfig for the given provider, reading
// the relevant env vars (after any .env file has been loaded).
func LoadForProvider(provider Provider) LLMConfig {
	cfg := LLMConfig{Provider: provider}
	switch provider {
	case ProviderOpenRouter:
		cfg.URL = getenv("OPENROUTER_URL", "https://openrouter.ai/api/v1/chat/completions")
		cfg.Model = getenv("OPENROUTER_MODEL", "openai/gpt-oss-120b")
		cfg.APIKey = os.Getenv("OPENROUTER_API_KEY")
	case ProviderGemini:
		cfg.URL = getenv("GEMINI_URL", "https://generativelanguage.googleapis.com/v1beta")
		cfg.Model = getenv("GEMINI_MODEL", "gemini-2.5-flash")
		cfg.APIKey = os.Getenv("GEMINI_API_KEY")
	default:
		cfg.Provider = ProviderLlamaCpp
		cfg.URL = getenv("LLAMACPP_URL", "http://localhost:8080/v1/chat/completions")
		cfg.Model = getenv("LLAMACPP_MODEL", "prism-ml/Bonsai-8B-gguf:Q1_0")
	}
	return cfg
}

func getenv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

// loadDotEnv reads a simple KEY=VALUE file and sets env vars that are not
// already set in the process environment. Lines starting with # are ignored.
// Values may be optionally wrapped in single or double quotes.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		if len(val) >= 2 {
			first, last := val[0], val[len(val)-1]
			if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
}
