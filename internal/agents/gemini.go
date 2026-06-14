package agents

import (
	"fmt"
	"os"
	"os/exec"
)

// geminiAgent is a stub, mirroring codexAgent. The full pipeline runs;
// auth bootstrap is a fast-follow.
type geminiAgent struct{}

func (geminiAgent) ID() string          { return "gemini" }
func (geminiAgent) DisplayName() string { return "Gemini CLI" }

func (geminiAgent) LaunchCmd(workdir string) []string { return []string{"gemini"} }

func (geminiAgent) Available() bool {
	_, err := exec.LookPath("gemini")
	return err == nil
}

func (geminiAgent) Bootstrap() error {
	fmt.Fprintln(os.Stderr, "  Gemini auth bootstrap is a fast-follow. If `gemini` is installed and already")
	fmt.Fprintln(os.Stderr, "  logged in, `box new --agent gemini` will run it. Otherwise log in via the gemini CLI.")
	return nil
}
