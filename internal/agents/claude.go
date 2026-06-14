package agents

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// claudeAgent is the first-class Claude Code integration.
type claudeAgent struct{}

func (claudeAgent) ID() string          { return "claude" }
func (claudeAgent) DisplayName() string { return "Claude Code" }

// LaunchCmd runs the claude CLI; tmux sets the working directory via -c.
func (claudeAgent) LaunchCmd(workdir string) []string { return []string{"claude"} }

func (claudeAgent) Available() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

// Bootstrap handles the headless-auth realities documented in the plan:
//   - the ANTHROPIC_API_KEY shadow footgun (silently bills metered API)
//   - the macOS Keychain-over-SSH lock (creds unreadable over SSH)
//   - full-scope interactive /login (required for Remote Control), vs a
//     headless inference-only token.
//
// It surfaces each clearly, then launches an interactive claude session so the
// user can run /login once (full scope). It never silently falls back to API
// billing.
func (claudeAgent) Bootstrap() error {
	if !claudeAvailable() {
		fmt.Fprintln(os.Stderr, "  ! claude CLI not found on PATH — install Claude Code first, then re-run `box host init`.")
		return fmt.Errorf("claude not installed")
	}

	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		fmt.Fprintln(os.Stderr, "  ! ANTHROPIC_API_KEY is set. Claude Code will use the metered API instead of your")
		fmt.Fprintln(os.Stderr, "    subscription. agentbox unsets it for spawned sessions when unset_anthropic_api_key")
		fmt.Fprintln(os.Stderr, "    is enabled (default). Remove it from your shell profile to be safe.")
	}

	if runtime.GOOS == "darwin" && overSSH() {
		fmt.Fprintln(os.Stderr, "  ! You're initializing over SSH on macOS. Claude stores credentials in the login")
		fmt.Fprintln(os.Stderr, "    Keychain, which is locked over SSH. If /login fails, run once on the physical")
		fmt.Fprintln(os.Stderr, "    console, or: security unlock-keychain ~/Library/Keychains/login.keychain-db")
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  Launching Claude so you can authenticate. Inside it, run:  /login")
	fmt.Fprintln(os.Stderr, "  (Full-scope /login also enables the optional Remote Control phone transport.)")
	fmt.Fprintln(os.Stderr, "  When you're signed in, exit Claude to continue setup.")
	fmt.Fprintln(os.Stderr, "")

	c := exec.Command("claude")
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	// A non-zero exit here just means the user quit; that's fine.
	_ = c.Run()
	return nil
}

func claudeAvailable() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

// overSSH reports whether the current process is running inside an SSH session.
func overSSH() bool {
	return os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CONNECTION") != ""
}
