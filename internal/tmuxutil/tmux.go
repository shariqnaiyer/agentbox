// Package tmuxutil is a thin wrapper over the tmux CLI. We drive tmux as a
// subprocess rather than linking a library so the binary stays dependency-free
// and the persistence layer is exactly the tmux the user already knows.
package tmuxutil

import (
	"os/exec"
	"strings"
)

// Bin is the tmux binary name; overridable for tests.
var Bin = "tmux"

// Available reports whether tmux is on PATH.
func Available() bool {
	_, err := exec.LookPath(Bin)
	return err == nil
}

// ServerRunning reports whether a tmux server is up for this user.
func ServerRunning() bool {
	// `tmux ls` exits non-zero ("no server running") when there's no server.
	return exec.Command(Bin, "ls").Run() == nil
}

// StartServer ensures a tmux server is running.
func StartServer() error {
	if ServerRunning() {
		return nil
	}
	return exec.Command(Bin, "start-server").Run()
}

// HasSession reports whether a session with the given name exists.
func HasSession(name string) bool {
	return exec.Command(Bin, "has-session", "-t", "="+name).Run() == nil
}

// NewDetached creates a detached session named `name` in directory `dir`
// running `cmd`. Mirrors spawn.sh:
//
//	tmux new-session -d -s <name> -c <dir> '<cmd>'
func NewDetached(name, dir string, cmd []string) error {
	args := []string{"new-session", "-d", "-s", name}
	if dir != "" {
		args = append(args, "-c", dir)
	}
	args = append(args, cmd...)
	return exec.Command(Bin, args...).Run()
}

// KillSession kills a session by name.
func KillSession(name string) error {
	return exec.Command(Bin, "kill-session", "-t", "="+name).Run()
}

// ListSessions returns the names of all sessions (empty if no server).
func ListSessions() ([]string, error) {
	if !ServerRunning() {
		return nil, nil
	}
	out, err := exec.Command(Bin, "list-sessions", "-F", "#{session_name}").Output()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			names = append(names, line)
		}
	}
	return names, nil
}

// Capture returns the last `lines` lines of a session's active pane.
func Capture(name string, lines int) (string, error) {
	out, err := exec.Command(Bin, "capture-pane", "-p", "-t", "="+name).Output()
	if err != nil {
		return "", err
	}
	s := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if lines > 0 && len(s) > lines {
		s = s[len(s)-lines:]
	}
	return strings.Join(s, "\n"), nil
}

// AttachArgs returns the argv that attaches to `name`, creating it if absent
// (`tmux new-session -A`). Transports append this after their own argv.
func AttachArgs(name string) []string {
	return []string{Bin, "new-session", "-A", "-s", name}
}

// RemoteAttachCommand returns a single shell string that attaches-or-creates
// the named session, for transports that take a remote command string
// (ssh/et). It prepends the Homebrew bin dirs so tmux is found even in a
// non-interactive ssh session that hasn't loaded the user's PATH.
func RemoteAttachCommand(name string) string {
	return `PATH="/opt/homebrew/bin:/usr/local/bin:$PATH" tmux new-session -A -s ` + shellQuote(name)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
