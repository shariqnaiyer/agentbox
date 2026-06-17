// Package transport implements the connection ladder. Every rung attaches the
// SAME named tmux session — they differ only in the pipe used to reach it:
//
//	mosh  → UDP, roams across IP/sleep changes (the default, the north-star)
//	et    → TCP single-port, for UDP-blocked cellular/corporate networks
//	ttyd  → WebSocket in a browser, for a device with no native client
//	ssh   → plain SSH + tmux, the universally-available last resort
//
// The client probes reachability, picks the best rung (preferring the one that
// last worked for this host), and execs the client so tmux gets the real TTY.
package transport

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/shariqnaiyer/agentbox/internal/config"
	"github.com/shariqnaiyer/agentbox/internal/tmuxutil"
)

// sshIdentityArgs returns ssh options pinning agentbox's own key (if `box trust`
// has set one up): use only that key, ignore any ssh-agent. Empty when no key
// exists, so default ssh behavior applies.
func sshIdentityArgs() []string {
	kp := config.SSHKeyPath()
	if _, err := os.Stat(kp); err != nil {
		return nil
	}
	return []string{"-o", "IdentitiesOnly=yes", "-o", "IdentityAgent=none", "-i", kp}
}

// Kind identifies a transport rung.
type Kind string

const (
	KindMosh Kind = "mosh"
	KindET   Kind = "et"
	KindTTYD Kind = "ttyd"
	KindSSH  Kind = "ssh"
)

// ladderOrder is the default preference, best first.
var ladderOrder = []Kind{KindMosh, KindET, KindTTYD, KindSSH}

const (
	sshPort    = 22
	etPort     = 2022
	probeDelay = 2 * time.Second
)

// Probe returns the reachable transports for a host, best first. If the host
// recorded a LastTransport that's still reachable, it's promoted to the front.
func Probe(h config.Host) []Kind {
	addr := h.Addr()
	var out []Kind
	for _, k := range ladderOrder {
		if reachable(k, h, addr) {
			out = append(out, k)
		}
	}
	if h.LastTransport != "" {
		out = promote(out, Kind(h.LastTransport))
	}
	return out
}

// Best returns the single best reachable transport, or ("", false).
func Best(h config.Host) (Kind, bool) {
	ks := Probe(h)
	if len(ks) == 0 {
		return "", false
	}
	return ks[0], true
}

func reachable(k Kind, h config.Host, addr string) bool {
	switch k {
	case KindMosh:
		// mosh bootstraps over SSH (TCP 22) then uses UDP. We can't cheaply
		// pre-test UDP, so we require the local client + TCP 22, try it first,
		// and fall through to et if the UDP phase fails at runtime.
		return haveClient("mosh") && tcpOpen(addr, sshPort)
	case KindET:
		return haveClient("et") && tcpOpen(addr, etPort)
	case KindTTYD:
		return h.TtydPort > 0 && tcpOpen(addr, h.TtydPort)
	case KindSSH:
		return tcpOpen(addr, sshPort)
	}
	return false
}

// ConnectArgs returns the binary and argv for a transport, plus a URL for the
// ttyd rung (which is opened rather than exec'd).
func ConnectArgs(k Kind, h config.Host, session string) (bin string, args []string, url string, err error) {
	user := h.SSHUser
	dest := h.Addr()
	target := dest
	if user != "" {
		target = user + "@" + dest
	}
	remote := tmuxutil.RemoteAttachCommand(session)
	id := sshIdentityArgs()
	switch k {
	case KindMosh:
		args := []string{}
		if len(id) > 0 {
			args = append(args, "--ssh=ssh "+strings.Join(id, " "))
		}
		// Run the attach through sh -c so the PATH prefix in `remote` applies
		// (Homebrew tmux isn't on a non-interactive PATH).
		args = append(args, target, "--", "sh", "-c", remote)
		return "mosh", args, "", nil
	case KindET:
		hostport := dest + ":" + itoa(etPort)
		if user != "" {
			hostport = user + "@" + hostport
		}
		return "et", []string{hostport, "-c", remote}, "", nil
	case KindSSH:
		args := append([]string{}, id...)
		args = append(args, "-t", target, remote)
		return "ssh", args, "", nil
	case KindTTYD:
		return "", nil, fmt.Sprintf("http://%s:%d", dest, h.TtydPort), nil
	}
	return "", nil, "", fmt.Errorf("unknown transport %q", k)
}

// Connect attaches to the session over the given transport. For mosh/et/ssh it
// execs the client (replacing this process) so tmux owns the terminal. For
// ttyd it opens the URL in a browser.
func Connect(k Kind, h config.Host, session string) error {
	bin, args, url, err := ConnectArgs(k, h, session)
	if err != nil {
		return err
	}
	if k == KindTTYD {
		fmt.Printf("Open in a browser: %s\n", url)
		return openURL(url)
	}
	path, err := exec.LookPath(bin)
	if err != nil {
		return fmt.Errorf("%s client not installed: %w", bin, err)
	}
	// Replace this process so the client gets our TTY directly.
	argv := append([]string{path}, args...)
	return syscall.Exec(path, argv, os.Environ())
}

func haveClient(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

func tcpOpen(host string, port int) bool {
	c, err := net.DialTimeout("tcp", net.JoinHostPort(host, itoa(port)), probeDelay)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

func promote(ks []Kind, k Kind) []Kind {
	out := []Kind{}
	found := false
	for _, x := range ks {
		if x == k {
			found = true
			continue
		}
		out = append(out, x)
	}
	if found {
		return append([]Kind{k}, out...)
	}
	return ks
}

func openURL(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

func itoa(n int) string { return fmt.Sprintf("%d", n) }
