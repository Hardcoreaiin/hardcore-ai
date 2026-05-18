package agent

import (
	"strconv"
	"strings"
)

// Tokenize splits a tool-call argument list. Handles "quoted strings" with
// \-escapes and bare tokens separated by commas. Strips `key: value`
// named-arg prefixes the model may emit, and coerces ints/floats/bools/null.
func Tokenize(argsStr string) []any {
	var tokens []any
	i := 0
	n := len(argsStr)

	for i < n {
		for i < n && (argsStr[i] == ' ' || argsStr[i] == '\t' || argsStr[i] == ',') {
			i++
		}
		if i >= n {
			break
		}

		if argsStr[i] == '"' {
			i++
			var sb strings.Builder
			for i < n && argsStr[i] != '"' {
				if argsStr[i] == '\\' && i+1 < n {
					sb.WriteByte(argsStr[i+1])
					i += 2
					continue
				}
				sb.WriteByte(argsStr[i])
				i++
			}
			if i < n {
				i++ // closing quote
			}
			tokens = append(tokens, sb.String())
			continue
		}

		start := i
		for i < n && argsStr[i] != ',' && argsStr[i] != ')' {
			i++
		}
		raw := strings.TrimSpace(argsStr[start:i])

		if colon := strings.IndexByte(raw, ':'); colon != -1 {
			key := strings.TrimSpace(raw[:colon])
			if key != "" && !strings.ContainsAny(key, " \t") {
				raw = strings.TrimSpace(raw[colon+1:])
				raw = strings.Trim(raw, `"`)
			}
		}

		tokens = append(tokens, coerce(raw))
	}

	return tokens
}

func coerce(raw string) any {
	if len(raw) >= 2 && raw[1] == ':' {
		switch raw[0] {
		case 's', 'i', 'f', 'b':
			raw = raw[2:]
		}
	}
	switch strings.ToLower(raw) {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}
	if i, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(raw, 64); err == nil {
		return f
	}
	return raw
}
