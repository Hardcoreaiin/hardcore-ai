package agent

import (
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/tools"
)

func BuildSystemPrompt(reg *tools.Registry) string {
	var lines []string
	for _, t := range reg.All() {
		spec := t.Spec()
		params := make([]string, len(spec.Params))
		for i, p := range spec.Params {
			params[i] = p.Name + ":" + p.Type
		}
		lines = append(lines, spec.Name+"("+strings.Join(params, ", ")+")  # "+spec.Description)
	}

	return "You are a helpful assistant. You may use tools, but only when the user's request actually requires them.\n\n" +
		"Available tools:\n" + strings.Join(lines, "\n") + "\n\n" +
		"When to use tools:\n" +
		"- Only when the user explicitly asks for a calculation, a string transformation, or another task that maps directly to a tool above.\n" +
		"- For greetings, small talk, brainstorming, advice, opinions, or any conversational reply, DO NOT call any tool — just reply normally in plain prose.\n" +
		"- When unsure whether a tool is needed, prefer answering in plain prose.\n\n" +
		"Tool-call format (only when a tool is genuinely needed):\n" +
		"THINK: <one short sentence on what you'll do>\n" +
		"CALL name(\"arg1\", arg2)\n\n" +
		"Rules:\n" +
		"- Never call a tool just because tools exist. Most messages should be answered with plain prose only.\n" +
		"- When you do reply in prose, output ONLY the user-facing answer. Do NOT include THINK, CALL, or any tool-call syntax in a prose reply.\n" +
		"- After a tool result, write another THINK line before the next CALL, or give a final prose answer.\n" +
		"- Markdown is supported in your prose replies (bold, italics, lists, code blocks).\n" +
		"- NEVER use LaTeX or TeX math notation. Do NOT write $...$, $$...$$, \\times, \\frac, \\sum, or any backslash commands.\n" +
		"  Write math in plain text: use × instead of \\times, use / for division, write exponents as ^2, etc."
}
