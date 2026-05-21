package agent

import "testing"

func TestBareCallRecovery(t *testing.T) {
	SetKnownTools([]string{"workspace_status", "build", "file_write"})

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"bare single line", "workspace_status()", "workspace_status"},
		{"bare with args", `build("stm32f4", "main.c")`, "build"},
		{"prose with paren rejected", "I will call build(stm32) now", ""},
		{"unknown tool rejected", "frobnicate()", ""},
		{"still works with CALL", `CALL workspace_status()`, "workspace_status"},
	}
	for _, c := range cases {
		got := ParseLine(c.in)
		if c.want == "" {
			if got.Kind == LineCall {
				t.Errorf("%s: expected non-call, got call %q", c.name, got.FuncName)
			}
			continue
		}
		if got.Kind != LineCall || got.FuncName != c.want {
			t.Errorf("%s: got kind=%d name=%q, want call %q", c.name, got.Kind, got.FuncName, c.want)
		}
	}
}

func TestScanBareCallMultiline(t *testing.T) {
	SetKnownTools([]string{"file_write"})
	text := "THINK: writing the file\nfile_write(\"main.c\", \"int main(){\nreturn 0;\n}\")"
	got := ParseFullText(text)
	if got.Kind != LineCall || got.FuncName != "file_write" {
		t.Fatalf("got kind=%d name=%q, want file_write call", got.Kind, got.FuncName)
	}
}
