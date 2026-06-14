package agents

import (
	"fmt"
	"os"
	"os/exec"
)

// codexAgent is a stub: wired end-to-end (registry, worktree, tmux session,
// status files all work) so `box new --agent codex` runs the full pipeline.
// Only the agent-specific auth bootstrap is a fast-follow.
type codexAgent struct{}

func (codexAgent) ID() string          { return "codex" }
func (codexAgent) DisplayName() string { return "OpenAI Codex CLI" }

func (codexAgent) LaunchCmd(workdir string) []string { return []string{"codex"} }

func (codexAgent) Available() bool {
	_, err := exec.LookPath("codex")
	return err == nil
}

func (codexAgent) Bootstrap() error {
	fmt.Fprintln(os.Stderr, "  Codex auth bootstrap is a fast-follow. If `codex` is installed and already")
	fmt.Fprintln(os.Stderr, "  logged in, `box new --agent codex` will run it. Otherwise log in via the codex CLI.")
	return nil
}
