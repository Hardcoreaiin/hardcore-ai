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

// knownTools holds the set of registered tool names so the parser can
// recognize a bare `toolname(args)` call even when a weak/quantized model
// forgets to emit the literal `CALL ` prefix. Set once at startup via
// SetKnownTools; reads are unsynchronized because writes happen before any
// turn runs.
var knownTools map[string]bool

// SetKnownTools registers the valid tool names with the parser. Must be called
// before any turn runs (BuildSystemPrompt does this).
func SetKnownTools(names []string) {
	knownTools = make(map[string]bool, len(names))
	for _, n := range names {
		knownTools[n] = true
	}
}

// looksLikeBareCall reports whether stripped is a bare `name(...)` invocation
// of a known tool with no CALL prefix. Returns the call body (everything from
// the name onward) when it matches.
func looksLikeBareCall(stripped string) (string, bool) {
	open := strings.IndexByte(stripped, '(')
	if open <= 0 {
		return "", false
	}
	name := strings.TrimSpace(stripped[:open])
	// A bare name must be a single identifier — reject prose like
	// "the function build(...)" or "I will call build(...)".
	if strings.ContainsAny(name, " \t") {
		return "", false
	}
	if !knownTools[name] {
		return "", false
	}
	return stripped, true
}

// ParseFullText scans a complete (multi-line) model response for a CALL block.
// Unlike ParseLine it can find a CALL whose argument list spans multiple lines
// (e.g. a file_write with embedded newlines). Returns a LineCall ParsedLine if
// found, otherwise a LinePlain zero value.
func ParseFullText(text string) ParsedLine {
	upper := strings.ToUpper(text)
	idx := strings.Index(upper, "\nCALL ")
	if idx == -1 {
		// Also check if the text starts with CALL (no leading newline)
		if strings.HasPrefix(upper, "CALL ") || strings.HasPrefix(upper, "CALL\t") {
			idx = -1 // handled below
		} else {
			// Recovery path: scan each line for a bare `toolname(args)` call
			// emitted without the `CALL ` prefix.
			if pl := scanBareCall(text); pl.Kind == LineCall {
				return pl
			}
			return ParsedLine{Kind: LinePlain}
		}
	}

	var callBody string
	if idx == -1 {
		// CALL at the very start
		callBody = strings.TrimLeft(text[4:], ":= \t")
	} else {
		after := text[idx+1:] // skip the \n
		callBody = strings.TrimLeft(after[4:], ":= \t")
	}

	// Find matching closing paren — it may be on a later line.
	open := strings.IndexByte(callBody, '(')
	if open == -1 {
		return ParsedLine{Kind: LinePlain}
	}
	close := matchCloseParen(callBody, open)
	if close == -1 {
		// No balanced close — fall back to the last ')'. A truncated
		// runaway response may legitimately lack the closing paren.
		close = strings.LastIndexByte(callBody, ')')
	}
	if close == -1 || close < open {
		return ParsedLine{Kind: LinePlain}
	}

	name := strings.TrimSpace(callBody[:open])
	args := callBody[open+1 : close]
	tokens := Tokenize(args)
	return ParsedLine{Kind: LineCall, FuncName: name, Tokens: tokens, Text: callBody[:close+1]}
}

// matchCloseParen returns the index of the ')' that balances the '(' at
// position open, counting nesting depth and skipping parens inside "quoted
// strings" (with \-escapes). Returns -1 if no balanced close exists. This
// stops parsing at the end of the FIRST call when a model concatenates
// several `CALL name(...)CALL name(...)` invocations in one response.
func matchCloseParen(s string, open int) int {
	depth := 0
	inStr := false
	for i := open; i < len(s); i++ {
		c := s[i]
		if inStr {
			if c == '\\' {
				i++ // skip escaped char
				continue
			}
			if c == '"' {
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// scanBareCall looks for a bare `toolname(args)` invocation anywhere in a
// multi-line response — used when the model omits the `CALL ` prefix. The
// argument list may span multiple lines (e.g. file_write content), so once a
// known tool name + `(` is found, it scans to the matching final `)`.
func scanBareCall(text string) ParsedLine {
	for _, line := range strings.Split(text, "\n") {
		stripped := strings.TrimSpace(line)
		open := strings.IndexByte(stripped, '(')
		if open <= 0 {
			continue
		}
		name := strings.TrimSpace(stripped[:open])
		if strings.ContainsAny(name, " \t") || !knownTools[name] {
			continue
		}
		// Found the call's start. The args may continue on later lines, so
		// take everything from this name onward in the full buffer.
		nameIdx := strings.Index(text, name+"(")
		if nameIdx == -1 {
			continue
		}
		body := text[nameIdx:]
		bOpen := strings.IndexByte(body, '(')
		bClose := matchCloseParen(body, bOpen)
		if bClose == -1 {
			bClose = strings.LastIndexByte(body, ')')
		}
		if bClose <= bOpen {
			continue
		}
		args := body[bOpen+1 : bClose]
		return ParsedLine{
			Kind:     LineCall,
			Text:     body[:bClose+1],
			FuncName: name,
			Tokens:   Tokenize(args),
		}
	}
	return ParsedLine{Kind: LinePlain}
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
		// Recovery path: a weak/quantized model often forgets the `CALL `
		// prefix and emits a bare `toolname(args)`. Accept it when the name
		// matches a registered tool.
		if body, ok := looksLikeBareCall(stripped); ok {
			callBody = body
			break
		}
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
	// Match the paren that balances the first '(' so a line containing several
	// concatenated `name(...)CALL name(...)` calls stops at the first one.
	end := matchCloseParen(s, open)
	if end == -1 {
		if !strings.HasSuffix(strings.TrimRight(s, " \t"), ")") {
			return "", nil, fmt.Errorf("no closing parenthesis: %s", s)
		}
		end = strings.LastIndexByte(s, ')')
	}
	name := strings.TrimSpace(s[:open])
	args := strings.TrimSpace(s[open+1 : end])
	if args == "" {
		return name, nil, nil
	}
	return name, Tokenize(args), nil
}
