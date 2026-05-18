package toolchain

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Manager handles downloading, extracting, and locating xpack toolchain binaries.
// All tools are stored under BaseDir (default: ~/.hardcoreai/toolchain/).
type Manager struct {
	BaseDir    string
	OnProgress func(tool Tool, downloaded, total int64)
	OnEvent    func(Event)

	mu      sync.Mutex
	pending map[Tool]*installState
}

type Event struct {
	Tool       Tool
	Stage      string
	Downloaded int64
	Total      int64
	Err        error
}

type installState struct {
	done chan struct{}
	err  error
}

type manifest struct {
	Installed map[string]string `json:"installed"` // tool name → installed dir name
}

func DefaultManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot find home dir: %w", err)
	}
	return &Manager{
		BaseDir: filepath.Join(home, ".hardcoreai", "toolchain"),
		pending: make(map[Tool]*installState),
	}, nil
}

// EnsureAsync starts a background download of tool if not installed.
// Returns (true, nil) when already installed.
// Returns (false, nil) when download is in progress — caller should retry later.
// Returns (false, err) if the download failed.
func (m *Manager) EnsureAsync(t Tool) (ready bool, err error) {
	if _, err := m.installedDir(t); err == nil {
		return true, nil
	}

	m.mu.Lock()
	st := m.pending[t]
	if st == nil {
		st = &installState{done: make(chan struct{})}
		m.pending[t] = st
		go func() {
			st.err = m.Ensure(context.Background(), t)
			close(st.done)
			m.mu.Lock()
			delete(m.pending, t)
			m.mu.Unlock()
		}()
	}
	m.mu.Unlock()

	select {
	case <-st.done:
		return st.err == nil, st.err
	default:
		return false, nil
	}
}

// BinPath returns the absolute path to a named binary inside the given tool's
// install directory, or an error if the tool is not installed.
func (m *Manager) BinPath(t Tool, binary string) (string, error) {
	dir, err := m.installedDir(t)
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	p := filepath.Join(dir, "bin", binary)
	if _, err := os.Stat(p); err != nil {
		return "", fmt.Errorf("binary %q not found in %s: %w", binary, dir, err)
	}
	return p, nil
}

// Ensure downloads and extracts the tool if not already present.
func (m *Manager) Ensure(ctx context.Context, t Tool) error {
	if _, err := m.installedDir(t); err == nil {
		m.emit(Event{Tool: t, Stage: "ready"})
		return nil // already installed
	}

	rel, err := resolveRelease(t)
	if err != nil {
		m.emit(Event{Tool: t, Stage: "error", Err: err})
		return err
	}

	if err := os.MkdirAll(m.BaseDir, 0o755); err != nil {
		err = fmt.Errorf("create toolchain dir: %w", err)
		m.emit(Event{Tool: t, Stage: "error", Err: err})
		return err
	}

	archivePath := filepath.Join(m.BaseDir, rel.filename+rel.ext)
	if err := m.download(ctx, t, rel, archivePath); err != nil {
		m.emit(Event{Tool: t, Stage: "error", Err: err})
		return err
	}
	defer os.Remove(archivePath)

	m.emit(Event{Tool: t, Stage: "extract"})
	if err := m.extract(archivePath, m.BaseDir, rel.ext); err != nil {
		err = fmt.Errorf("extract %s: %w", archivePath, err)
		m.emit(Event{Tool: t, Stage: "error", Err: err})
		return err
	}

	if err := m.saveManifest(t, rel.filename); err != nil {
		m.emit(Event{Tool: t, Stage: "error", Err: err})
		return err
	}
	m.emit(Event{Tool: t, Stage: "ready"})
	return nil
}

func (m *Manager) download(ctx context.Context, t Tool, rel release, dest string) error {
	url := rel.baseURL + "/" + rel.filename + rel.ext
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}

	total := resp.ContentLength
	m.emit(Event{Tool: t, Stage: "download", Downloaded: 0, Total: total})

	// Fetch .sha checksum file for verification
	shaURL := url + ".sha"
	expectedSHA, _ := m.fetchSHA(ctx, shaURL)

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}
	defer f.Close()

	h := sha256.New()
	var downloaded int64
	buf := make([]byte, 32*1024)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return werr
			}
			h.Write(buf[:n])
			downloaded += int64(n)
			if m.OnProgress != nil {
				m.OnProgress(t, downloaded, total)
			}
			m.emit(Event{Tool: t, Stage: "download", Downloaded: downloaded, Total: total})
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}

	if expectedSHA != "" {
		got := hex.EncodeToString(h.Sum(nil))
		if got != expectedSHA {
			return fmt.Errorf("checksum mismatch for %s: got %s, want %s", rel.filename, got, expectedSHA)
		}
	}

	return nil
}

func (m *Manager) emit(ev Event) {
	if m.OnEvent != nil {
		m.OnEvent(ev)
	}
}

func (m *Manager) fetchSHA(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	// xpack .sha files contain "<hash>  <filename>" — take the first field
	parts := strings.Fields(string(data))
	if len(parts) == 0 {
		return "", fmt.Errorf("empty sha file")
	}
	return parts[0], nil
}

func (m *Manager) extract(archivePath, destDir, ext string) error {
	switch ext {
	case ".tar.gz":
		return extractTarGz(archivePath, destDir)
	case ".zip":
		return extractZip(archivePath, destDir)
	default:
		return fmt.Errorf("unknown archive extension: %s", ext)
	}
}

func extractTarGz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		target := filepath.Join(dest, filepath.FromSlash(hdr.Name))
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("path traversal in archive: %s", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		case tar.TypeSymlink:
			os.Remove(target)
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(dest, filepath.FromSlash(f.Name))
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("path traversal in archive: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0o755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// installedDir returns the root directory of an installed tool or an error if missing.
func (m *Manager) installedDir(t Tool) (string, error) {
	mf, err := m.loadManifest()
	if err == nil {
		if dir, ok := mf.Installed[string(t)]; ok {
			full := filepath.Join(m.BaseDir, dir)
			if _, serr := os.Stat(full); serr == nil {
				return full, nil
			}
		}
	}
	// Fallback: scan BaseDir for a matching prefix
	entries, err := os.ReadDir(m.BaseDir)
	if err != nil {
		return "", fmt.Errorf("tool %s not installed", t)
	}
	prefix := xpackDirPrefix(t)
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), prefix) {
			return filepath.Join(m.BaseDir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("tool %s not installed", t)
}

func xpackDirPrefix(t Tool) string {
	switch t {
	case ToolGCC:
		return "xpack-arm-none-eabi-gcc-"
	case ToolQEMU:
		return "xpack-qemu-arm-"
	default:
		return string(t)
	}
}

func (m *Manager) manifestPath() string {
	return filepath.Join(m.BaseDir, "manifest.json")
}

func (m *Manager) loadManifest() (manifest, error) {
	data, err := os.ReadFile(m.manifestPath())
	if err != nil {
		return manifest{}, err
	}
	var mf manifest
	return mf, json.Unmarshal(data, &mf)
}

func (m *Manager) saveManifest(t Tool, dirName string) error {
	mf, _ := m.loadManifest()
	if mf.Installed == nil {
		mf.Installed = map[string]string{}
	}
	mf.Installed[string(t)] = dirName
	data, err := json.MarshalIndent(mf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.manifestPath(), data, 0o644)
}
