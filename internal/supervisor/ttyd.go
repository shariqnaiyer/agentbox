package supervisor

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/shariqnaiyer/agentbox/internal/transport"
)

// ttydServer manages the optional host-side ttyd process.
type ttydServer struct {
	port   int
	bindIP string
	cmd    *exec.Cmd
}

func newTtydServer(port int, bindIP string) *ttydServer {
	return &ttydServer{port: port, bindIP: bindIP}
}

// Running reports whether the ttyd child is still alive.
func (s *ttydServer) Running() bool {
	if s.cmd == nil || s.cmd.Process == nil {
		return false
	}
	return s.cmd.Process.Signal(syscall.Signal(0)) == nil
}

// Start launches ttyd if it isn't already running.
func (s *ttydServer) Start(session string) error {
	if s.Running() {
		return nil
	}
	if _, err := exec.LookPath("ttyd"); err != nil {
		return err
	}
	args := transport.TtydServerArgs(s.port, s.bindIP, session)
	c := exec.Command("ttyd", args...)
	c.Stdout, c.Stderr = os.Stderr, os.Stderr // ttyd logs to stderr
	if err := c.Start(); err != nil {
		return err
	}
	s.cmd = c
	return nil
}

// Stop kills the ttyd child.
func (s *ttydServer) Stop() {
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
}
