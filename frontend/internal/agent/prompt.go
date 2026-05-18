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

	return "You are a helpful assistant with access to tools.\n\n" +
		"Tools:\n" + strings.Join(lines, "\n") + "\n\n" +
		"Format:\n" +
		"Whenever you want to use a tool, you MUST write:\n" +
		"THINK: <one sentence: what you just learned and what you will do next>\n" +
		"CALL name(\"arg1\", arg2)\n\n" +
		"Rules:\n" +
		"- Always reason in THINK before each CALL. Never skip THINK.\n" +
		"- After a tool result, write another THINK line before your next CALL.\n" +
		"- Never ask the user for information you can discover with a tool.\n" +
		"- If no tool is needed, reply normally without THINK or CALL."
}
