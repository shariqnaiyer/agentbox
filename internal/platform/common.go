package platform

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// which reports whether a binary is on PATH.
func which(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

// runCapture runs a command and returns combined output.
func runCapture(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// runInteractive runs a command with inherited stdio, so prompts (sudo,
// brew, claude /login) reach the user's terminal.
func runInteractive(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// elevate runs args as root: directly if already root, else via sudo with
// inherited stdio so the password prompt works. Shared by both platforms.
func elevate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("elevate: empty command")
	}
	if os.Geteuid() == 0 {
		return runInteractive(args[0], args[1:]...)
	}
	if !which("sudo") {
		return fmt.Errorf("need root for %q but sudo is not available; re-run as root", strings.Join(args, " "))
	}
	return runInteractive("sudo", args...)
}

// procKeepAwake holds a long-running child process (caffeinate /
// systemd-inhibit) that asserts wakefulness until killed.
type procKeepAwake struct {
	cmd *exec.Cmd
}

func (p *procKeepAwake) Release() error {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	err := p.cmd.Process.Kill()
	_, _ = p.cmd.Process.Wait()
	return err
}

// noopKeepAwake is returned when the OS cannot assert wakefulness (e.g. a
// Linux box with no systemd). The host usually never sleeps anyway.
type noopKeepAwake struct{}

func (noopKeepAwake) Release() error { return nil }

// startDetached launches a background assertion process and returns a handle.
func startDetached(name string, args ...string) (KeepAwake, error) {
	if !which(name) {
		return noopKeepAwake{}, fmt.Errorf("%s not found; keep-awake not asserted", name)
	}
	c := exec.Command(name, args...)
	if err := c.Start(); err != nil {
		return noopKeepAwake{}, err
	}
	return &procKeepAwake{cmd: c}, nil
}
