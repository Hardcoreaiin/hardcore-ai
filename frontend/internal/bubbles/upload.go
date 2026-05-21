package bubbles

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Hardcoreaiin/hardcore-ai/frontend/internal/theme"
	"github.com/charmbracelet/lipgloss"
)

type FileItem struct {
	Name  string
	IsDir bool
	Size  int64
}

type UploadBubble struct {
	theme       *theme.Theme
	currentDir  string
	parentDir   string
	items       []FileItem
	parentItems []FileItem
	cursor      int
	selected    map[string]bool
	active      bool
}

func NewUploadBubble(t *theme.Theme, cwd string) *UploadBubble {
	u := &UploadBubble{
		theme:    t,
		selected: make(map[string]bool),
		active:   true,
	}
	u.navigateTo(cwd)
	return u
}

func (u *UploadBubble) Active() bool {
	return u.active
}

func (u *UploadBubble) SetActive(active bool) {
	u.active = active
}

func (u *UploadBubble) SelectedFiles() []string {
	var files []string
	for f := range u.selected {
		files = append(files, f)
	}
	sort.Strings(files)
	return files
}

func (u *UploadBubble) navigateTo(path string) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return
	}
	u.currentDir = abs
	u.parentDir = filepath.Dir(abs)

	// Read current directory
	items, _ := readDir(abs)
	u.items = items
	u.cursor = 0

	// Read parent directory
	parentItems, _ := readDir(u.parentDir)
	u.parentItems = parentItems
}

func readDir(path string) ([]FileItem, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var items []FileItem
	for _, entry := range entries {
		info, err := entry.Info()
		var size int64
		if err == nil {
			size = info.Size()
		}
		items = append(items, FileItem{
			Name:  entry.Name(),
			IsDir: entry.IsDir(),
			Size:  size,
		})
	}
	// Sort: directories first, then files alphabetically
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDir && !items[j].IsDir {
			return true
		}
		if !items[i].IsDir && items[j].IsDir {
			return false
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return items, nil
}

// HandleKey processes navigation and selection keys.
// Returns (done, confirmed, selectedFiles).
func (u *UploadBubble) HandleKey(key string) (bool, bool, []string) {
	switch key {
	case "up", "k":
		if u.cursor > 0 {
			u.cursor--
		}
	case "down", "j":
		if u.cursor < len(u.items)-1 {
			u.cursor++
		}
	case "left", "h", "backspace":
		if u.currentDir != u.parentDir {
			oldDirName := filepath.Base(u.currentDir)
			u.navigateTo(u.parentDir)
			// Position cursor on the directory we just came out of
			for i, item := range u.items {
				if item.Name == oldDirName {
					u.cursor = i
					break
				}
			}
		}
	case "right", "l", "enter":
		if len(u.items) > 0 {
			item := u.items[u.cursor]
			if item.IsDir {
				u.navigateTo(filepath.Join(u.currentDir, item.Name))
			}
		}
	case "space", " ":
		if len(u.items) > 0 {
			item := u.items[u.cursor]
			if !item.IsDir && strings.HasSuffix(strings.ToLower(item.Name), ".pdf") {
				path := filepath.Join(u.currentDir, item.Name)
				if u.selected[path] {
					delete(u.selected, path)
				} else {
					u.selected[path] = true
				}
			}
		}
	case "y":
		return true, true, u.SelectedFiles()
	case "q", "esc":
		return true, false, nil
	}
	return false, false, nil
}

func (u *UploadBubble) View(width, height int) string {
	t := u.theme
	boxWidth := width - 4
	boxHeight := height - 4
	if boxWidth < 45 {
		boxWidth = 45
	}
	if boxHeight < 12 {
		boxHeight = 12
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Accent).
		Padding(1, 2).
		Width(boxWidth).
		Height(boxHeight)

	title := lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("RAG File Upload (Ranger-style)")

	colWidth := (boxWidth - 10) / 3
	colHeight := boxHeight - 6
	if colHeight < 4 {
		colHeight = 4
	}

	// 1. Left Column: Parent directory contents
	var leftLines []string
	currentBase := filepath.Base(u.currentDir)
	for i, item := range u.parentItems {
		if i >= colHeight {
			leftLines = append(leftLines, "  ...")
			break
		}
		prefix := "📄 "
		if item.IsDir {
			prefix = "📁 "
		}
		displayName := prefix + item.Name
		truncatedName := truncateRunes(displayName, colWidth-4)
		var line string
		if item.Name == currentBase {
			line = lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("▸ " + truncatedName)
		} else {
			line = "  " + truncatedName
		}
		leftLines = append(leftLines, line)
	}
	if len(leftLines) == 0 {
		leftLines = append(leftLines, lipgloss.NewStyle().Foreground(t.Muted).Render("(empty)"))
	}

	leftCol := lipgloss.NewStyle().
		Width(colWidth).
		Height(colHeight).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(t.BorderMuted).
		PaddingRight(1).
		Render(strings.Join(leftLines, "\n"))

	// 2. Middle Column: Current directory contents (Interactive)
	var middleLines []string
	for i, item := range u.items {
		if i >= colHeight {
			middleLines = append(middleLines, "  ...")
			break
		}
		prefix := "📄 "
		if item.IsDir {
			prefix = "📁 "
		}

		path := filepath.Join(u.currentDir, item.Name)
		checkbox := ""
		if !item.IsDir && strings.HasSuffix(strings.ToLower(item.Name), ".pdf") {
			if u.selected[path] {
				checkbox = "[x] "
			} else {
				checkbox = "[ ] "
			}
		}

		displayName := checkbox + prefix + item.Name
		truncatedName := truncateRunes(displayName, colWidth-4)
		var line string
		if i == u.cursor {
			line = lipgloss.NewStyle().Foreground(t.Accent).Bold(true).Render("▸ " + truncatedName)
		} else {
			line = "  " + truncatedName
		}
		middleLines = append(middleLines, line)
	}
	if len(middleLines) == 0 {
		middleLines = append(middleLines, lipgloss.NewStyle().Foreground(t.Muted).Render("(empty)"))
	}

	middleCol := lipgloss.NewStyle().
		Width(colWidth).
		Height(colHeight).
		BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(t.BorderMuted).
		PaddingRight(1).
		PaddingLeft(1).
		Render(strings.Join(middleLines, "\n"))

	// 3. Right Column: Preview / details of highlighted item
	var rightContent string
	if len(u.items) > 0 && u.cursor < len(u.items) {
		item := u.items[u.cursor]
		if item.IsDir {
			subItems, _ := readDir(filepath.Join(u.currentDir, item.Name))
			rightContent = fmt.Sprintf("Directory: %s\n\nItems: %d", item.Name, len(subItems))
		} else {
			sizeStr := formatSize(item.Size)
			isPDF := strings.HasSuffix(strings.ToLower(item.Name), ".pdf")
			ragAction := "Cannot index (non-PDF)"
			if isPDF {
				ragAction = "Press [space] to select"
			}
			rightContent = fmt.Sprintf("File: %s\n\nSize: %s\n\n%s", item.Name, sizeStr, ragAction)
		}
	} else {
		rightContent = "No items selected"
	}

	// Add list of selected files at the bottom of Right Column if any are selected
	selCount := len(u.selected)
	rightContent += fmt.Sprintf("\n\nSelected for upload: %d", selCount)
	if selCount > 0 {
		rightContent += "\n"
		var selNames []string
		for p := range u.selected {
			selNames = append(selNames, "• "+filepath.Base(p))
		}
		sort.Strings(selNames)
		for i, n := range selNames {
			if i >= 3 {
				rightContent += fmt.Sprintf("\n... and %d more", selCount-3)
				break
			}
			rightContent += "\n" + n
		}
	}

	rightCol := lipgloss.NewStyle().
		Width(colWidth).
		Height(colHeight).
		PaddingLeft(1).
		Render(rightContent)

	cols := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, middleCol, rightCol)
	footer := lipgloss.NewStyle().Foreground(t.Muted).Render(fmt.Sprintf("Current dir: %s\nControls: h/j/k/l or arrow keys | space select PDF | y confirm upload | q cancel", u.currentDir))

	content := lipgloss.JoinVertical(lipgloss.Left, title, "", cols, "", footer)
	return box.Render(content)
}

func formatSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max < 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}
