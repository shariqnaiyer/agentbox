// Package doctorcheck runs read-only host diagnostics. The same checks back
// both the `box doctor` command and the supervisor's health view, so "is this
// box healthy?" has one definition.
package doctorcheck

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/shariqnaiyer/agentbox/internal/agents"
	"github.com/shariqnaiyer/agentbox/internal/config"
	"github.com/shariqnaiyer/agentbox/internal/platform"
	"github.com/shariqnaiyer/agentbox/internal/reach"
)

// Severity classifies a check result.
type Severity string

const (
	SevOK    Severity = "ok"
	SevWarn  Severity = "warn"
	SevError Severity = "error"
)

// Check is one diagnostic result.
type Check struct {
	Name     string   `json:"name"`
	OK       bool     `json:"ok"`
	Severity Severity `json:"severity"`
	Detail   string   `json:"detail"`
	Fix      string   `json:"fix,omitempty"`
}

// RunAll runs every diagnostic for the host.
func RunAll(plat platform.Platform) []Check {
	var cs []Check

	cs = append(cs, binCheck("tmux", SevError, "Install tmux (box host init does this)."))
	if reach.Installed() {
		if reach.IsRunning() {
			cs = append(cs, Check{Name: "tailscale", OK: true, Severity: SevOK, Detail: "running " + reach.IPv4()})
		} else {
			cs = append(cs, Check{Name: "tailscale", OK: false, Severity: SevError,
				Detail: "installed but not running", Fix: "Run: box host init  (joins your tailnet)"})
		}
	} else {
		cs = append(cs, Check{Name: "tailscale", OK: false, Severity: SevError,
			Detail: "not installed", Fix: "Install Tailscale, then box host init."})
	}

	cs = append(cs, binCheck("mosh", SevWarn, "Default transport. box host init installs it."))
	cs = append(cs, binCheck("et", SevWarn, "TCP fallback for UDP-blocked networks. Optional."))
	cs = append(cs, binCheck("ttyd", SevWarn, "Browser fallback for clientless devices. Optional."))
	cs = append(cs, binCheck("claude", SevWarn, "Install Claude Code to run the default agent."))

	// Autostart unit.
	if st, err := plat.AutostartStatus(); err == nil {
		switch {
		case st.Installed && st.Running:
			cs = append(cs, Check{Name: "autostart", OK: true, Severity: SevOK, Detail: "installed & running"})
		case st.Installed:
			cs = append(cs, Check{Name: "autostart", OK: false, Severity: SevWarn,
				Detail: "installed but not running", Fix: "Start it: box host  (or reboot)."})
		default:
			cs = append(cs, Check{Name: "autostart", OK: false, Severity: SevWarn,
				Detail: "not installed", Fix: "Run: box host init"})
		}
	}

	// ANTHROPIC_API_KEY shadow footgun.
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		cs = append(cs, Check{Name: "anthropic-api-key", OK: false, Severity: SevWarn,
			Detail: "set — Claude will use metered API, not your subscription",
			Fix:    "Remove ANTHROPIC_API_KEY from your profile, or rely on unset_anthropic_api_key (default on)."})
	} else {
		cs = append(cs, Check{Name: "anthropic-api-key", OK: true, Severity: SevOK, Detail: "not set (subscription auth)"})
	}

	// macOS Keychain-over-SSH.
	if runtime.GOOS == "darwin" && overSSH() {
		cs = append(cs, Check{Name: "keychain-over-ssh", OK: false, Severity: SevWarn,
			Detail: "init over SSH on macOS; login Keychain is locked",
			Fix:    "Run once on the console, or: security unlock-keychain ~/Library/Keychains/login.keychain-db"})
	}

	// ~/.agents toolkit + hooks.
	if dirExists(agents.AgentsDir()) {
		cs = append(cs, Check{Name: "agents-toolkit", OK: true, Severity: SevOK, Detail: agents.AgentsDir()})
	} else {
		cs = append(cs, Check{Name: "agents-toolkit", OK: false, Severity: SevWarn,
			Detail: "~/.agents missing", Fix: "Run: box host init  (installs the status toolkit)."})
	}
	if installed, _ := agents.HooksInstalled(); installed {
		cs = append(cs, Check{Name: "claude-hooks", OK: true, Severity: SevOK, Detail: "status hooks wired"})
	} else {
		cs = append(cs, Check{Name: "claude-hooks", OK: false, Severity: SevWarn,
			Detail: "status hooks not wired", Fix: "Run: box host init  (wires set-status hooks)."})
	}

	// Initialized?
	if _, err := os.Stat(filepath.Join(config.Dir(), "config.json")); err == nil {
		cs = append(cs, Check{Name: "config", OK: true, Severity: SevOK, Detail: config.Dir()})
	} else {
		cs = append(cs, Check{Name: "config", OK: false, Severity: SevWarn,
			Detail: "host not initialized", Fix: "Run: box host init"})
	}

	return cs
}

// Failures returns the error-severity checks (the smoke-test gate).
func Failures(cs []Check) []Check {
	var out []Check
	for _, c := range cs {
		if !c.OK && c.Severity == SevError {
			out = append(out, c)
		}
	}
	return out
}

func binCheck(bin string, sev Severity, fix string) Check {
	if _, err := exec.LookPath(bin); err == nil {
		return Check{Name: bin, OK: true, Severity: SevOK, Detail: "present"}
	}
	return Check{Name: bin, OK: false, Severity: sev, Detail: "not found", Fix: fix}
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func overSSH() bool {
	return os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CONNECTION") != ""
}
