package bubbles

import (
	"sort"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/agent"
	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

// FileTreeBubble tracks all files written during a session.
// Persists across turns — only reset explicitly via Reset().
type FileTreeBubble struct {
	project string   // project name shown as root
	files   []string // relative paths
	theme   *theme.Theme
}

func NewFileTreeBubble(t *theme.Theme) *FileTreeBubble {
	return &FileTreeBubble{theme: t, project: "workspace"}
}

func (b *FileTreeBubble) Title() string           { return "workspace" }
func (b *FileTreeBubble) SetTheme(t *theme.Theme) { b.theme = t }
func (b *FileTreeBubble) HasContent() bool        { return b.project != "" || len(b.files) > 0 }

func (b *FileTreeBubble) SetProject(name string) {
	if name != "" {
		b.project = name
	}
}

func (b *FileTreeBubble) AddFile(rel string) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return
	}
	for _, f := range b.files {
		if f == rel {
			return
		}
	}
	b.files = append(b.files, rel)
	sort.Strings(b.files)
}

func (b *FileTreeBubble) Handle(_ agent.Event) bool { return false }

func (b *FileTreeBubble) Reset() {
	b.files = nil
	b.project = "workspace"
}

func (b *FileTreeBubble) View(width int) string {
	t := b.theme
	titleStyle := lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	rootStyle := lipgloss.NewStyle().Foreground(t.UserFg).Bold(true)
	dirStyle := lipgloss.NewStyle().Foreground(t.UserFg)
	fileStyle := lipgloss.NewStyle().Foreground(t.Text)

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
	sb.WriteString(rootStyle.Render(b.project+"/") + "\n")

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
