package agent

import (
	"fmt"
	"strings"
)

type LineKind int

const (
	LinePlain LineKind = iota
	LineThink
	LineCall
	LineTodo
	LineAsk
)

type ParsedLine struct {
	Kind     LineKind
	Text     string // Plain: raw line. Think: thought (after "THINK:"). Call: original.
	FuncName string // Call only
	Tokens   []any  // Call only
	Err      error  // Call only — set if call line was malformed
}

// ParseLine classifies a single line of model output.
func ParseLine(raw string) ParsedLine {
	stripped := strings.TrimSpace(raw)
	upper := strings.ToUpper(stripped)

	if strings.HasPrefix(upper, "THINK:") {
		return ParsedLine{Kind: LineThink, Text: strings.TrimSpace(stripped[6:])}
	}

	if strings.HasPrefix(upper, "TODO:") {
		body := strings.TrimSpace(stripped[5:])
		items := splitPipe(body)
		return ParsedLine{Kind: LineTodo, Text: body, Tokens: stringsToAny(items)}
	}

	if strings.HasPrefix(upper, "ASK:") {
		body := strings.TrimSpace(stripped[4:])
		parts := splitPipe(body)
		return ParsedLine{Kind: LineAsk, Text: body, Tokens: stringsToAny(parts)}
	}

	var callBody string
	switch {
	case strings.HasPrefix(upper, "CALL"):
		callBody = strings.TrimLeft(stripped[4:], ":= \t")
	case strings.Contains(stripped, "<|tool_call>"):
		after := strings.SplitN(stripped, "<|tool_call>", 2)[1]
		callBody = strings.TrimLeft(after, "call:Call: ")
	default:
		return ParsedLine{Kind: LinePlain, Text: raw}
	}

	name, tokens, err := parseCall(strings.TrimSpace(callBody))
	return ParsedLine{Kind: LineCall, Text: raw, FuncName: name, Tokens: tokens, Err: err}
}

// splitPipe splits s on " | " (or "|") and trims each part.
func splitPipe(s string) []string {
	raw := strings.Split(s, "|")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func stringsToAny(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

func parseCall(s string) (string, []any, error) {
	open := strings.IndexByte(s, '(')
	if open == -1 {
		return "", nil, fmt.Errorf("no opening parenthesis: %s", s)
	}
	if !strings.HasSuffix(strings.TrimRight(s, " \t"), ")") {
		return "", nil, fmt.Errorf("no closing parenthesis: %s", s)
	}
	name := strings.TrimSpace(s[:open])
	end := strings.LastIndexByte(s, ')')
	args := strings.TrimSpace(s[open+1 : end])
	if args == "" {
		return name, nil, nil
	}
	return name, Tokenize(args), nil
}
