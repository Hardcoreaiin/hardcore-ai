package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type OpenAIConfig struct {
	URL    string
	Model  string
	APIKey string
}

func DefaultConfig() OpenAIConfig {
	return OpenAIConfig{
		URL:   "http://localhost:8080/v1/chat/completions",
		Model: "prism-ml/Bonsai-8B-gguf:Q1_0",
	}
}

type OpenAIClient struct {
	cfg  OpenAIConfig
	http *http.Client
}

func NewOpenAI(cfg OpenAIConfig) *OpenAIClient {
	return &OpenAIClient{cfg: cfg, http: &http.Client{}}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type chatStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func (c *OpenAIClient) Stream(ctx context.Context, msgs []Message) (<-chan Line, error) {
	body := chatRequest{
		Model:    c.cfg.Model,
		Stream:   true,
		Messages: make([]chatMessage, len(msgs)),
	}
	for i, m := range msgs {
		body.Messages[i] = chatMessage{Role: string(m.Role), Content: m.Content}
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.cfg.URL, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		b, _ := readAll(resp.Body, 2048)
		return nil, fmt.Errorf("llm http %d: %s", resp.StatusCode, b)
	}

	out := make(chan Line, 16)
	go func() {
		defer resp.Body.Close()
		defer close(out)

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 64*1024), 1024*1024)

		var lineBuf strings.Builder

		emit := func(s string) bool {
			select {
			case out <- Line{Text: s}:
				return true
			case <-ctx.Done():
				return false
			}
		}

		flushPartial := func() bool {
			if lineBuf.Len() == 0 {
				return true
			}
			ok := emit(lineBuf.String())
			lineBuf.Reset()
			return ok
		}

		for scanner.Scan() {
			raw := scanner.Text()
			if !strings.HasPrefix(raw, "data:") {
				continue
			}
			payload := strings.TrimSpace(raw[5:])
			if payload == "[DONE]" {
				break
			}
			var chunk chatStreamChunk
			if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) == 0 {
				continue
			}
			token := chunk.Choices[0].Delta.Content
			if token == "" {
				continue
			}

			for {
				idx := strings.IndexByte(token, '\n')
				if idx < 0 {
					lineBuf.WriteString(token)
					break
				}
				lineBuf.WriteString(token[:idx])
				if !emit(lineBuf.String()) {
					return
				}
				lineBuf.Reset()
				token = token[idx+1:]
			}
		}

		if !flushPartial() {
			return
		}
		select {
		case out <- Line{Done: true}:
		case <-ctx.Done():
		}
	}()

	return out, nil
}

func readAll(r interface{ Read(p []byte) (int, error) }, max int) (string, error) {
	buf := make([]byte, max)
	n, err := r.Read(buf)
	return string(buf[:n]), err
}
