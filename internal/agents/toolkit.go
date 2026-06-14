package agents

import (
	"embed"
	"os"
	"path/filepath"
)

// toolkitFS holds vendored copies of the ~/.agents status scripts. On a fresh
// host that doesn't already have the user's toolkit, InstallToolkit drops these
// in so the status-file convention (and thus `box ls`) works everywhere.
//
//go:embed toolkit/*.sh
var toolkitFS embed.FS

// AgentsDir is ~/.agents.
func AgentsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agents")
}

// InstallToolkit writes the vendored toolkit scripts into ~/.agents if they
// are not already present. Existing user scripts are never overwritten — the
// real toolkit on the user's Mac stays the source of truth.
func InstallToolkit() error {
	dir := AgentsDir()
	if err := os.MkdirAll(filepath.Join(dir, "status"), 0o755); err != nil {
		return err
	}
	entries, err := toolkitFS.ReadDir("toolkit")
	if err != nil {
		return err
	}
	for _, e := range entries {
		dst := filepath.Join(dir, e.Name())
		if _, err := os.Stat(dst); err == nil {
			continue // don't clobber the user's version
		}
		data, err := toolkitFS.ReadFile("toolkit/" + e.Name())
		if err != nil {
			return err
		}
		if err := os.WriteFile(dst, data, 0o755); err != nil {
			return err
		}
	}
	return nil
}
