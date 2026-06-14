// Package agentstatus is the read-only bridge to the user's existing
// ~/.agents toolkit. Claude Code hooks already write one file per tmux
// session at ~/.agents/status/<session> containing "<state>\t<unix-ts>".
// We read those files and apply the same staleness rule as agents-status.sh
// (an "active" agent with no event for >300s is treated as idle).
//
// This is the key reuse seam: box ls and the supervisor get live agent
// status for free, with zero new state, as long as agents run in named
// tmux sessions.
package agentstatus

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// State mirrors the toolkit's vocabulary.
type State string

const (
	Active  State = "active"
	Idle    State = "idle"
	Done    State = "done"
	Unknown State = "unknown"
)

// StaleAfter matches agents-status.sh's 300-second threshold.
const StaleAfter = 300 * time.Second

// Agent is one tracked session's status.
type Agent struct {
	Name    string    `json:"name"`
	State   State     `json:"state"`
	Updated time.Time `json:"updated"`
}

// Dir is ~/.agents/status, the toolkit's status directory.
func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agents", "status")
}

// EnsureDir creates the status directory if missing.
func EnsureDir() error { return os.MkdirAll(Dir(), 0o755) }

// Seed writes an initial status for a session, mirroring spawn.sh line 20:
//
//	printf 'idle\t%s\n' "$(date +%s)" > ~/.agents/status/<name>
func Seed(name string, st State, now time.Time) error {
	if err := EnsureDir(); err != nil {
		return err
	}
	line := fmt.Sprintf("%s\t%d\n", st, now.Unix())
	return os.WriteFile(filepath.Join(Dir(), name), []byte(line), 0o644)
}

// Read parses one status file, applying the staleness rule.
func Read(name string, now time.Time) (Agent, error) {
	b, err := os.ReadFile(filepath.Join(Dir(), name))
	if err != nil {
		return Agent{}, err
	}
	return parse(name, string(b), now), nil
}

// List returns the status of every tracked session, newest first.
func List(now time.Time) ([]Agent, error) {
	entries, err := os.ReadDir(Dir())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Agent
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(Dir(), e.Name()))
		if err != nil {
			continue
		}
		out = append(out, parse(e.Name(), string(b), now))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Updated.After(out[j].Updated) })
	return out, nil
}

func parse(name, content string, now time.Time) Agent {
	a := Agent{Name: name, State: Unknown}
	fields := strings.Split(strings.TrimSpace(content), "\t")
	if len(fields) > 0 && fields[0] != "" {
		a.State = State(fields[0])
	}
	if len(fields) > 1 {
		if ts, err := strconv.ParseInt(strings.TrimSpace(fields[1]), 10, 64); err == nil {
			a.Updated = time.Unix(ts, 0)
		}
	}
	// Staleness: an "active" agent silent for >300s is really idle.
	if a.State == Active && !a.Updated.IsZero() && now.Sub(a.Updated) > StaleAfter {
		a.State = Idle
	}
	return a
}

// Icon returns a short glyph for a state, for terminal listings.
func (a Agent) Icon() string {
	switch a.State {
	case Active:
		return "●"
	case Idle:
		return "○"
	case Done:
		return "✓"
	default:
		return "✗"
	}
}
