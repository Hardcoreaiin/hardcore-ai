package bubbles

import (
	"sort"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// FileTreeBubble tracks all files written during a session and renders them
// as a simple directory tree. Updated by the TUI on stm32_write_file results.
type FileTreeBubble struct {
	files []string // relative paths, e.g. "main.c", "src/led.c"
	theme *theme.Theme
}

func NewFileTreeBubble(t *theme.Theme) *FileTreeBubble {
	return &FileTreeBubble{theme: t}
}

func (b *FileTreeBubble) SetTheme(t *theme.Theme) { b.theme = t }
func (b *FileTreeBubble) HasContent() bool         { return len(b.files) > 0 }

// AddFile records a new file path (idempotent).
func (b *FileTreeBubble) AddFile(rel string) {
	for _, f := range b.files {
		if f == rel {
			return
		}
	}
	b.files = append(b.files, rel)
	sort.Strings(b.files)
}

func (b *FileTreeBubble) Handle(ev agent.Event) bool {
	if _, ok := ev.(agent.UserMessageEvent); ok {
		b.files = nil
		return true
	}
	return false
}

func (b *FileTreeBubble) View(width int) string {
	t := b.theme
	titleStyle := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	dirStyle := lipgloss.NewStyle().Foreground(t.UserFg)
	fileStyle := lipgloss.NewStyle().Foreground(t.Text)

	// Build a simple tree: group by first path component.
	type entry struct {
		dir  string
		file string
	}
	var entries []entry
	for _, f := range b.files {
		parts := strings.SplitN(f, "/", 2)
		if len(parts) == 1 {
			entries = append(entries, entry{"", parts[0]})
		} else {
			entries = append(entries, entry{parts[0], parts[1]})
		}
	}

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("workspace") + "\n")
	sb.WriteString(dirStyle.Render("workspace/") + "\n")

	seenDirs := map[string]bool{}
	for _, e := range entries {
		if e.dir != "" {
			if !seenDirs[e.dir] {
				seenDirs[e.dir] = true
				sb.WriteString("  " + dirStyle.Render(e.dir+"/") + "\n")
			}
			sb.WriteString("    " + fileStyle.Render("└─ "+e.file) + "\n")
		} else {
			sb.WriteString("  " + fileStyle.Render("├─ "+e.file) + "\n")
		}
	}

	body := strings.TrimRight(sb.String(), "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.BorderMuted).
		Padding(0, 1).
		Width(width - 2).
		Render(body)
}
