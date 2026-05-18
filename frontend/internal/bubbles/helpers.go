package bubbles

import (
	"fmt"
	"regexp"
	"strings"
)

// anyToString renders a tool argument for display. Strings get quoted;
// everything else uses fmt's default representation.
func anyToString(a any) string {
	if s, ok := a.(string); ok {
		return fmt.Sprintf("%q", s)
	}
	return fmt.Sprintf("%v", a)
}

// latexCommands maps common TeX command sequences to plain-text equivalents.
var latexCommands = strings.NewReplacer(
	`\times`, "×",
	`\cdot`, "·",
	`\div`, "÷",
	`\pm`, "±",
	`\neq`, "≠",
	`\leq`, "≤",
	`\geq`, "≥",
	`\approx`, "≈",
	`\infty`, "∞",
	`\sqrt`, "√",
	`\sum`, "Σ",
	`\prod`, "Π",
	`\alpha`, "α",
	`\beta`, "β",
	`\gamma`, "γ",
	`\delta`, "δ",
	`\pi`, "π",
	`\theta`, "θ",
	`\lambda`, "λ",
	`\mu`, "μ",
	`\sigma`, "σ",
	`\left(`, "(",
	`\right)`, ")",
	`\left[`, "[",
	`\right]`, "]",
	`\left{`, "{",
	`\right}`, "}",
)

// reFrac matches \frac{a}{b} and replaces with (a/b).
var reFrac = regexp.MustCompile(`\\frac\{([^}]*)\}\{([^}]*)\}`)

// reInlineMath strips surrounding $ or $$ delimiters, keeping the inner text.
var reDisplayMath = regexp.MustCompile(`\$\$([^$]+)\$\$`)
var reInlineMath = regexp.MustCompile(`\$([^$\n]+)\$`)

// StripLatex converts common LaTeX notation to readable plain-text Unicode.
func StripLatex(s string) string {
	// Display math first ($$...$$), then inline ($...$).
	s = reDisplayMath.ReplaceAllString(s, "$1")
	s = reInlineMath.ReplaceAllString(s, "$1")
	// \frac{a}{b} → (a/b)
	s = reFrac.ReplaceAllStringFunc(s, func(m string) string {
		sub := reFrac.FindStringSubmatch(m)
		if len(sub) == 3 {
			return "(" + sub[1] + "/" + sub[2] + ")"
		}
		return m
	})
	// Named commands
	s = latexCommands.Replace(s)
	// Strip remaining lone backslash-word sequences (e.g. \text, \mathrm)
	s = regexp.MustCompile(`\\[a-zA-Z]+\s*`).ReplaceAllString(s, "")
	// Collapse curly braces used as grouping
	s = strings.NewReplacer("{", "", "}", "").Replace(s)
	return s
}
