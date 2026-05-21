// Package editmatch implements precise, anchored file editing for the
// file_edit tool.
//
// The agent's other writing tool, file_write, can only overwrite a whole file.
// When a build fails, using file_write means the model regenerates the entire
// file just to fix one line. editmatch lets the model emit only the changed
// span — surrounded by one unchanged context line above and below — and
// applies it surgically by anchoring on that exact surrounding text.
//
// Wire format (FormatPaired): the model supplies, per edit site, TWO fenced
// ``` blocks right after the CALL line. The first block is the exact text to
// find in the file (the original lines plus one context line above and
// below); the second block is the replacement (same context, changed lines).
// Multiple edit sites in one turn = multiple before/after fence pairs.
//
//	CALL file_edit("main.c")
//	```c
//	    LOG("boot");
//	    uint32_t mode = 0
//	    GPIOA[0] |= (1 << 10);
//	```
//	```c
//	    LOG("boot");
//	    uint32_t mode = 0;
//	    GPIOA[0] |= (1 << 10);
//	```
//
// The context lines make the "before" block a unique anchor; ApplyEdit refuses
// to act if it matches zero or multiple places, so a stale or under-specified
// anchor fails loudly instead of editing the wrong line.
//
// The core (ApplyEdit/ApplyAll) is pure string-in/string-out — unit-testable
// with no compiler or toolchain present.
package editmatch

import (
	"fmt"
	"strings"
)

// Edit is one resolved edit site: find Old verbatim in the file, replace with
// New. Both include the unchanged context lines, so Old is a unique anchor.
type Edit struct {
	Old string // exact text to locate (context + original lines)
	New string // replacement text (context + changed lines)
}

// EditResult reports the outcome of applying a single Edit.
type EditResult struct {
	Edit
	Applied   bool
	StartLine int   // 1-based line where Old began (0 if not applied)
	EndLine   int   // 1-based line where Old ended
	Err       error // non-nil if the anchor was missing or ambiguous
}

// ApplyEdit locates exactly one occurrence of e.Old in content and replaces it
// with e.New. It deliberately refuses ambiguous edits: if Old appears zero or
// more than one time it returns an error telling the model how to fix the
// request. This is the same safety contract Claude Code's Edit uses.
func ApplyEdit(content string, e Edit) (string, EditResult) {
	res := EditResult{Edit: e}

	if e.Old == "" {
		res.Err = fmt.Errorf("empty before-block: it must contain the original lines plus one unchanged context line above and below")
		return content, res
	}
	if e.Old == e.New {
		res.Err = fmt.Errorf("before and after blocks are identical — nothing to change")
		return content, res
	}

	n := strings.Count(content, e.Old)
	switch {
	case n == 0:
		res.Err = fmt.Errorf("anchor not found — the before-block does not match the file verbatim; re-read the file and copy the exact lines including indentation")
		return content, res
	case n > 1:
		res.Err = fmt.Errorf("anchor is ambiguous: matched %d places — include more surrounding context lines so it is unique", n)
		return content, res
	}

	idx := strings.Index(content, e.Old)
	res.StartLine = 1 + strings.Count(content[:idx], "\n")
	res.EndLine = res.StartLine + strings.Count(strings.TrimRight(e.Old, "\n"), "\n")
	res.Applied = true

	return content[:idx] + e.New + content[idx+len(e.Old):], res
}

// ApplyAll applies edits sequentially. Each edit sees the result of the
// previous one, so edit sites must not overlap. The first failure stops the
// run; earlier successful edits are kept in the returned content so the model
// can see partial progress in the error report.
func ApplyAll(content string, edits []Edit) (string, []EditResult) {
	results := make([]EditResult, 0, len(edits))
	cur := content
	for _, e := range edits {
		next, r := ApplyEdit(cur, e)
		results = append(results, r)
		if r.Err != nil {
			return cur, results
		}
		cur = next
	}
	return cur, results
}

// ParseEdits extracts before/after fence pairs from a model response into a
// list of edit sites. Fences are consumed two at a time: first = before,
// second = after. It returns an error if the fence structure is wrong (no
// fences, or an odd count).
func ParseEdits(text string) ([]Edit, error) {
	fences := extractFences(text)
	if len(fences) == 0 {
		return nil, fmt.Errorf("no fenced ``` blocks found — supply a before block and an after block")
	}
	if len(fences)%2 != 0 {
		return nil, fmt.Errorf("got %d fenced blocks — file_edit needs an even number (one before + one after per edit site)", len(fences))
	}
	edits := make([]Edit, 0, len(fences)/2)
	for i := 0; i < len(fences); i += 2 {
		edits = append(edits, Edit{Old: fences[i], New: fences[i+1]})
	}
	return edits, nil
}

// extractFences pulls every ``` fenced block body out of text in order.
// Interior whitespace is preserved verbatim — source code is matched
// byte-for-byte — but the trailing newline before the closing ``` is dropped.
func extractFences(text string) []string {
	var out []string
	rest := text
	for {
		start := strings.Index(rest, "```")
		if start == -1 {
			break
		}
		rest = rest[start+3:]
		nl := strings.IndexByte(rest, '\n')
		if nl == -1 {
			break // opening fence with no newline — give up
		}
		body := rest[nl+1:] // skip the language tag line
		end := strings.Index(body, "```")
		if end == -1 {
			break // unterminated fence
		}
		out = append(out, strings.TrimRight(body[:end], "\n"))
		rest = body[end+3:]
	}
	return out
}
