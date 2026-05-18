package tools

import (
	"context"
	"fmt"
	"strings"
)

type StringUtilsArgs struct {
	Text      string `tool:"text" desc:"input text"`
	Operation string `tool:"operation" desc:"one of: upper, lower, reverse, count_words"`
}

func RegisterStringUtils(r *Registry) {
	Register(r, "string_utils",
		"Transform text. Operations: upper, lower, reverse, count_words.",
		func(_ context.Context, a StringUtilsArgs) (string, []Artifact, error) {
			switch a.Operation {
			case "upper":
				return strings.ToUpper(a.Text), nil, nil
			case "lower":
				return strings.ToLower(a.Text), nil, nil
			case "reverse":
				r := []rune(a.Text)
				for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
					r[i], r[j] = r[j], r[i]
				}
				return string(r), nil, nil
			case "count_words":
				return fmt.Sprintf("%d", len(strings.Fields(a.Text))), nil, nil
			}
			return "", nil, fmt.Errorf("unknown operation: %s", a.Operation)
		})
}
