// Package settings persists per-directory user choices.
//
// Settings live in <cwd>/.agent_settings/settings.json. The directory is
// created on first save. Trust is per-directory: the user explicitly
// opts in before the agent starts running tools in that workspace.
package settings

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

const (
	dirName  = ".agent_settings"
	fileName = "settings.json"
)

type Settings struct {
	Theme     string    `json:"theme"`
	Trusted   bool      `json:"trusted"`
	Provider  string    `json:"provider,omitempty"`
	Pet       string    `json:"pet"`
	PetName   string    `json:"pet_name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Path returns the absolute path to the settings file for the given root.
func Path(root string) string {
	return filepath.Join(root, dirName, fileName)
}

// Exists reports whether a settings file is already on disk for the root.
func Exists(root string) bool {
	_, err := os.Stat(Path(root))
	return err == nil
}

// Load reads settings for the root. Returns os.ErrNotExist (wrapped) when
// the file is missing — callers should treat that as "onboarding needed".
func Load(root string) (Settings, error) {
	var s Settings
	b, err := os.ReadFile(Path(root))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return s, err
		}
		return s, err
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return s, err
	}
	return s, nil
}

// Save writes settings to disk, creating .agent_settings/ as needed.
func Save(root string, s Settings) error {
	dir := filepath.Join(root, dirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(root), b, 0o644)
}
