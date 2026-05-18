package embedded

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	wsMu         sync.Mutex
	wsRootDir    string
	wsProjectDir string
)

// WorkspaceDir returns the active project directory, creating it if needed.
func WorkspaceDir() (string, error) {
	wsMu.Lock()
	defer wsMu.Unlock()
	if wsProjectDir != "" {
		if _, err := os.Stat(wsProjectDir); err == nil {
			return wsProjectDir, nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	wsRootDir = filepath.Join(home, ".hardcoreai", "workspace")
	wsProjectDir = filepath.Join(wsRootDir, "default")
	if err := os.MkdirAll(wsProjectDir, 0o755); err != nil {
		return "", err
	}
	return wsProjectDir, nil
}

// SetWorkspaceProject switches the active project directory.
func SetWorkspaceProject(name string) (string, error) {
	wsMu.Lock()
	defer wsMu.Unlock()
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	wsRootDir = filepath.Join(home, ".hardcoreai", "workspace")
	dir := filepath.Join(wsRootDir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	wsProjectDir = dir
	return dir, nil
}

// CurrentProjectName returns the active project name (last path component).
func CurrentProjectName() string {
	wsMu.Lock()
	defer wsMu.Unlock()
	if wsProjectDir == "" {
		return "default"
	}
	return filepath.Base(wsProjectDir)
}
