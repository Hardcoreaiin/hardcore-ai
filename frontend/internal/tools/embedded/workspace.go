package embedded

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	wsMu         sync.Mutex
	wsProjectDir string
)

// SetWorkspaceRoot sets the initial active directory. Called once at startup
// with the launch cwd so the agent operates on the user's real filesystem
// instead of a hidden sandbox.
func SetWorkspaceRoot(dir string) {
	wsMu.Lock()
	defer wsMu.Unlock()
	if abs, err := filepath.Abs(dir); err == nil {
		wsProjectDir = abs
	} else {
		wsProjectDir = dir
	}
}

// DefaultProjectsDir returns ~/.hardcoreai/projects, creating it if missing.
// This is the sandbox the agent works in by default so it never touches the
// directory the app was launched from. Falls back to the launch cwd if the
// home directory cannot be resolved or the directory cannot be created.
func DefaultProjectsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".hardcoreai", "projects")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// WorkspaceDir returns the active project directory. Falls back to the process
// cwd if nothing has been set yet.
func WorkspaceDir() (string, error) {
	wsMu.Lock()
	defer wsMu.Unlock()
	if wsProjectDir != "" {
		if info, err := os.Stat(wsProjectDir); err == nil && info.IsDir() {
			return wsProjectDir, nil
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	wsProjectDir = cwd
	return cwd, nil
}

// ResolvePath turns a user/agent-supplied path into an absolute path. It
// expands a leading ~, and resolves relative paths against the active
// directory. It does not require the path to exist.
func ResolvePath(p string) (string, error) {
	if p == "" {
		return WorkspaceDir()
	}
	if p == "~" || (len(p) >= 2 && p[:2] == "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if p == "~" {
			p = home
		} else {
			p = filepath.Join(home, p[2:])
		}
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p), nil
	}
	base, err := WorkspaceDir()
	if err != nil {
		return "", err
	}
	return filepath.Clean(filepath.Join(base, p)), nil
}

// SetWorkspaceProject switches the active directory. The argument may be an
// absolute path, a ~-relative path, or a path relative to the current active
// directory. The directory is created if it does not exist.
func SetWorkspaceProject(path string) (string, error) {
	abs, err := ResolvePath(path)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", err
	}
	wsMu.Lock()
	wsProjectDir = abs
	wsMu.Unlock()
	return abs, nil
}

// ChangeDir switches the active directory to an existing directory. Unlike
// SetWorkspaceProject it does not create the directory — it errors if the
// target is missing or is not a directory.
func ChangeDir(path string) (string, error) {
	abs, err := ResolvePath(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("cannot cd into %s: %w", abs, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", abs)
	}
	wsMu.Lock()
	wsProjectDir = abs
	wsMu.Unlock()
	return abs, nil
}

// CurrentProjectName returns the active directory's last path component.
func CurrentProjectName() string {
	wsMu.Lock()
	defer wsMu.Unlock()
	if wsProjectDir == "" {
		return "."
	}
	return filepath.Base(wsProjectDir)
}

// CurrentDir returns the full active directory path.
func CurrentDir() string {
	wsMu.Lock()
	defer wsMu.Unlock()
	return wsProjectDir
}
