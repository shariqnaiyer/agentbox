package transport

// TtydPickerSession is the tmux session ttyd serves by default — an
// attach-or-create session that shows whatever the user last looked at.
const TtydPickerSession = "agentbox"

// TtydServerArgs builds the argv for the host-side ttyd server. It binds to the
// Tailscale IP (so it's only reachable on the tailnet) and serves a writable
// terminal attached to the named tmux session.
func TtydServerArgs(port int, bindIP, session string) []string {
	args := []string{"-p", itoa(port), "-W"}
	if bindIP != "" {
		args = append(args, "-i", bindIP)
	}
	args = append(args, "tmux", "new-session", "-A", "-s", session)
	return args
}
