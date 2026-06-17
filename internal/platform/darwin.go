//go:build darwin

package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const darwinLabel = "ai.agentbox.host"

type darwinPlatform struct{}

func newPlatform() Platform { return darwinPlatform{} }

func (darwinPlatform) OS() string { return "darwin" }

func (darwinPlatform) DaemonLabel() string { return darwinLabel }

// PreventSleep runs `caffeinate -dis`, asserting display/idle/system sleep are
// prevented for as long as the child lives.
//
// Caveat (documented in DECISIONS.md / doctor): on a laptop running on
// battery, closing the lid can still suspend the machine despite caffeinate.
// A dedicated always-on Mac should stay on AC power.
func (darwinPlatform) PreventSleep(reason string) (KeepAwake, error) {
	return startDetached("caffeinate", "-dis")
}

func launchAgentPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", darwinLabel+".plist")
}

func (d darwinPlatform) InstallAutostart(spec AutostartSpec) error {
	plist := renderPlist(spec)
	path := launchAgentPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(plist), 0o644); err != nil {
		return err
	}
	domain := "gui/" + strconv.Itoa(os.Getuid())
	// bootout first so re-running init reliably reloads a changed plist.
	_, _ = runCapture("launchctl", "bootout", domain+"/"+darwinLabel)
	if _, err := runCapture("launchctl", "bootstrap", domain, path); err != nil {
		return fmt.Errorf("launchctl bootstrap: %w", err)
	}
	_, _ = runCapture("launchctl", "enable", domain+"/"+darwinLabel)
	return nil
}

func (darwinPlatform) RemoveAutostart() error {
	domain := "gui/" + strconv.Itoa(os.Getuid())
	_, _ = runCapture("launchctl", "bootout", domain+"/"+darwinLabel)
	return os.Remove(launchAgentPath())
}

func (darwinPlatform) AutostartStatus() (AutostartState, error) {
	st := AutostartState{}
	if _, err := os.Stat(launchAgentPath()); err == nil {
		st.Installed = true
	}
	domain := "gui/" + strconv.Itoa(os.Getuid())
	out, err := runCapture("launchctl", "print", domain+"/"+darwinLabel)
	if err != nil {
		return st, nil // not loaded
	}
	st.Running = strings.Contains(out, "state = running")
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "pid = ") {
			if pid, e := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "pid = "))); e == nil {
				st.PID = pid
			}
		}
	}
	return st, nil
}

// Elevate: on macOS in v1 none of our operations need root (LaunchAgent is
// per-user, brew is non-root), but we honor the contract generically.
func (darwinPlatform) Elevate(args []string, reason string) error { return elevate(args) }

func (darwinPlatform) PackageManager() PkgMgr {
	return PkgMgr{Name: "brew", Available: which("brew")}
}

// darwinPkgNames maps logical names to Homebrew formulae. Eternal Terminal is
// not in homebrew-core — it lives in the MisterTea/et tap, and the tap-qualified
// name auto-taps on install.
var darwinPkgNames = map[string]string{
	"mosh": "mosh",
	"et":   "MisterTea/et/et",
	"ttyd": "ttyd",
	"tmux": "tmux",
}

func (d darwinPlatform) InstallPackages(logical ...string) error {
	if !which("brew") {
		return fmt.Errorf("Homebrew not found; install from https://brew.sh then re-run, or install %v manually", logical)
	}
	// Install one at a time so a single failure (e.g. a tap hiccup) doesn't
	// block the others.
	var failed []string
	for _, l := range logical {
		name, ok := darwinPkgNames[l]
		if !ok {
			continue
		}
		if err := runInteractive("brew", "install", name); err != nil {
			failed = append(failed, l)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("failed to install %v", failed)
	}
	return nil
}

func renderPlist(spec AutostartSpec) string {
	var args strings.Builder
	for _, a := range spec.Exec {
		args.WriteString("    <string>" + xmlEscape(a) + "</string>\n")
	}
	logPath := agentboxLogPath()
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>%s</string>
  <key>ProgramArguments</key>
  <array>
%s  </array>
  <key>RunAtLoad</key>
  <%t/>
  <key>KeepAlive</key>
  <%t/>
  <key>ProcessType</key>
  <string>Background</string>
  <key>StandardOutPath</key>
  <string>%s</string>
  <key>StandardErrorPath</key>
  <string>%s</string>
</dict>
</plist>
`, darwinLabel, args.String(), spec.RunAtLoad, spec.KeepAlive, logPath, logPath)
}

func agentboxLogPath() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "agentbox", "host.log")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "agentbox", "host.log")
}

func xmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}
