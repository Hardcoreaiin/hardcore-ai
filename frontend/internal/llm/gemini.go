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

type GeminiConfig struct {
	URL    string // base URL, e.g. https://generativelanguage.googleapis.com/v1beta
	Model  string
	APIKey string
}

type GeminiClient struct {
	cfg  GeminiConfig
	http *http.Client
}

func NewGemini(cfg GeminiConfig) *GeminiClient {
	return &GeminiClient{cfg: cfg, http: &http.Client{}}
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiRequest struct {
	Contents          []geminiContent `json:"contents"`
	SystemInstruction *geminiContent  `json:"systemInstruction,omitempty"`
}

type geminiStreamChunk struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
}

func (c *GeminiClient) Stream(ctx context.Context, msgs []Message) (<-chan Line, error) {
	req := geminiRequest{}

	var systemParts []geminiPart
	for _, m := range msgs {
		switch m.Role {
		case RoleSystem:
			systemParts = append(systemParts, geminiPart{Text: m.Content})
		case RoleUser:
			req.Contents = append(req.Contents, geminiContent{Role: "user", Parts: []geminiPart{{Text: m.Content}}})
		case RoleAssistant:
			req.Contents = append(req.Contents, geminiContent{Role: "model", Parts: []geminiPart{{Text: m.Content}}})
		}
	}
	if len(systemParts) > 0 {
		req.SystemInstruction = &geminiContent{Parts: systemParts}
	}

	buf, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	base := strings.TrimRight(c.cfg.URL, "/")
	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse", base, c.cfg.Model)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if c.cfg.APIKey != "" {
		httpReq.Header.Set("x-goog-api-key", c.cfg.APIKey)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		b, _ := readAll(resp.Body, 2048)
		return nil, fmt.Errorf("gemini http %d: %s", resp.StatusCode, b)
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
			if payload == "" || payload == "[DONE]" {
				continue
			}
			var chunk geminiStreamChunk
			if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
				continue
			}
			if len(chunk.Candidates) == 0 {
				continue
			}
			for _, part := range chunk.Candidates[0].Content.Parts {
				token := part.Text
				if token == "" {
					continue
				}
				// Emit the raw token immediately for the stream ticker.
				select {
				case out <- Line{Token: token}:
				case <-ctx.Done():
					return
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
