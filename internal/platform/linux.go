//go:build linux

package platform

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

const linuxLabel = "agentbox-host"

const unitPath = "/etc/systemd/system/" + linuxLabel + ".service"

type linuxPlatform struct{}

func newPlatform() Platform { return linuxPlatform{} }

func (linuxPlatform) OS() string { return "linux" }

func (linuxPlatform) DaemonLabel() string { return linuxLabel }

// PreventSleep blocks sleep/idle/lid via systemd-inhibit, held by a child
// `sleep infinity`. On a box with no systemd (rare; some containers) this is
// a no-op — most always-on Linux hosts never sleep anyway.
func (linuxPlatform) PreventSleep(reason string) (KeepAwake, error) {
	if !which("systemd-inhibit") {
		return noopKeepAwake{}, nil
	}
	return startDetached("systemd-inhibit",
		"--what=sleep:idle:handle-lid-switch",
		"--who=agentbox",
		"--why="+reason,
		"--mode=block",
		"sleep", "infinity",
	)
}

// InstallAutostart installs a SYSTEM unit that runs as the invoking user.
// A system unit (vs a --user unit) survives logout without enable-linger,
// which is what an always-on headless box needs. Requires one-time sudo.
func (l linuxPlatform) InstallAutostart(spec AutostartSpec) error {
	u, err := user.Current()
	if err != nil {
		return err
	}
	unit := renderUnit(spec, u.Username, u.HomeDir)

	// Write to a temp file the user owns, then install it with root.
	tmp, err := os.CreateTemp("", "agentbox-unit-*.service")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(unit); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()

	if err := elevate([]string{"install", "-m", "0644", tmp.Name(), unitPath}); err != nil {
		return fmt.Errorf("install unit (needs sudo; run init interactively): %w", err)
	}
	if err := elevate([]string{"systemctl", "daemon-reload"}); err != nil {
		return err
	}
	return elevate([]string{"systemctl", "enable", "--now", linuxLabel})
}

func (linuxPlatform) RemoveAutostart() error {
	_ = elevate([]string{"systemctl", "disable", "--now", linuxLabel})
	return elevate([]string{"rm", "-f", unitPath})
}

func (linuxPlatform) AutostartStatus() (AutostartState, error) {
	st := AutostartState{}
	if _, err := os.Stat(unitPath); err == nil {
		st.Installed = true
	}
	if out, _ := runCapture("systemctl", "is-active", linuxLabel); out == "active" {
		st.Running = true
	}
	if out, err := runCapture("systemctl", "show", "-p", "MainPID", "--value", linuxLabel); err == nil {
		if pid, e := strconv.Atoi(strings.TrimSpace(out)); e == nil {
			st.PID = pid
		}
	}
	return st, nil
}

func (linuxPlatform) Elevate(args []string, reason string) error { return elevate(args) }

func (linuxPlatform) PackageManager() PkgMgr {
	for _, m := range []string{"apt-get", "dnf", "apk", "pacman"} {
		if which(m) {
			return PkgMgr{Name: m, Available: true}
		}
	}
	return PkgMgr{}
}

// linuxPkgNames maps logical names to per-manager package names. A missing
// entry means that manager has no package for it (e.g. eternalterminal is
// not in default apt repos) — we skip it rather than fail.
var linuxPkgNames = map[string]map[string]string{
	"apt-get": {"mosh": "mosh", "ttyd": "ttyd", "tmux": "tmux"}, // no et
	"dnf":     {"mosh": "mosh", "ttyd": "ttyd", "tmux": "tmux"}, // et via copr only
	"apk":     {"mosh": "mosh", "ttyd": "ttyd", "tmux": "tmux"},
	"pacman":  {"mosh": "mosh", "ttyd": "ttyd", "tmux": "tmux"},
}

func (l linuxPlatform) InstallPackages(logical ...string) error {
	pm := l.PackageManager()
	if !pm.Available {
		return fmt.Errorf("no supported package manager found; install %v manually", logical)
	}
	names := linuxPkgNames[pm.Name]
	var pkgs, skipped []string
	for _, lg := range logical {
		if n, ok := names[lg]; ok {
			pkgs = append(pkgs, n)
		} else {
			skipped = append(skipped, lg)
		}
	}
	if len(skipped) > 0 {
		fmt.Fprintf(os.Stderr, "note: %s has no package for %v; the transport ladder will skip them\n", pm.Name, skipped)
	}
	if len(pkgs) == 0 {
		return nil
	}
	var cmd []string
	switch pm.Name {
	case "apt-get":
		cmd = append([]string{"apt-get", "install", "-y"}, pkgs...)
	case "dnf":
		cmd = append([]string{"dnf", "install", "-y"}, pkgs...)
	case "apk":
		cmd = append([]string{"apk", "add"}, pkgs...)
	case "pacman":
		cmd = append([]string{"pacman", "-S", "--noconfirm"}, pkgs...)
	}
	return elevate(cmd)
}

func renderUnit(spec AutostartSpec, username, home string) string {
	exec := strings.Join(quoteAll(spec.Exec), " ")
	restart := "no"
	if spec.KeepAlive {
		restart = "always"
	}
	logDir := filepath.Join(home, ".config", "agentbox")
	return fmt.Sprintf(`[Unit]
Description=%s
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=%s
Environment=HOME=%s
Environment=XDG_CONFIG_HOME=%s
ExecStart=%s
Restart=%s
RestartSec=3

[Install]
WantedBy=multi-user.target
`, spec.Description, username, home, filepath.Dir(logDir), exec, restart)
}

func quoteAll(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " \t") {
			out[i] = `"` + a + `"`
		} else {
			out[i] = a
		}
	}
	return out
}
