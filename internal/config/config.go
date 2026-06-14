// Package config owns agentbox's on-disk state: paths, the host config,
// the known-hosts list, and the supervisor's managed-agent declarations.
//
// Decision: we use JSON (stdlib) rather than TOML so agentbox has zero
// external Go dependencies and builds offline on any host. See DECISIONS.md.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config is the per-host configuration written by `box host init`.
type Config struct {
	HostName     string `json:"host_name"`
	DefaultAgent string `json:"default_agent"`
	TtydPort     int    `json:"ttyd_port"`
	// UnsetAnthropicAPIKey, when true, makes spawned agent sessions unset
	// ANTHROPIC_API_KEY so Claude uses the subscription instead of metered API.
	UnsetAnthropicAPIKey bool `json:"unset_anthropic_api_key"`
	// WebEnabled controls whether the supervisor keeps a ttyd server alive.
	WebEnabled bool `json:"web_enabled"`
}

// DefaultConfig returns the baseline config used before `host init` runs.
func DefaultConfig() Config {
	return Config{
		DefaultAgent: "claude",
		TtydPort:     7681,
	}
}

// Dir returns agentbox's config directory, ~/.config/agentbox on every OS.
// We deliberately use ~/.config even on macOS (rather than os.UserConfigDir's
// ~/Library/Application Support) so paths are identical across hosts.
func Dir() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "agentbox")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "agentbox")
}

// StateDir is where the supervisor writes runtime state (state.json, host.log).
// Same directory as config for simplicity.
func StateDir() string { return Dir() }

func configPath() string  { return filepath.Join(Dir(), "config.json") }
func hostsPath() string   { return filepath.Join(Dir(), "hosts.json") }
func managedPath() string { return filepath.Join(Dir(), "managed.json") }

// StatePath is the path of the supervisor's health snapshot.
func StatePath() string { return filepath.Join(StateDir(), "state.json") }

// LogPath is the supervisor's ring-buffer log.
func LogPath() string { return filepath.Join(StateDir(), "host.log") }

// EnsureDir creates the config directory if missing.
func EnsureDir() error { return os.MkdirAll(Dir(), 0o755) }

// Load reads config.json, returning DefaultConfig() if it does not exist.
func Load() (Config, error) {
	c := DefaultConfig()
	b, err := os.ReadFile(configPath())
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	return c, nil
}

// Save writes config.json atomically.
func Save(c Config) error {
	if err := EnsureDir(); err != nil {
		return err
	}
	return WriteAtomic(configPath(), mustJSON(c))
}

// WriteAtomic writes data to path via a temp file + rename, so readers never
// observe a partially written file. Used for every state file we own.
func WriteAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op if the rename succeeded
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func mustJSON(v any) []byte {
	b, _ := json.MarshalIndent(v, "", "  ")
	return append(b, '\n')
}
