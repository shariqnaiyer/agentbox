package box

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/shariqnaiyer/agentbox/internal/agentstatus"
	"github.com/shariqnaiyer/agentbox/internal/config"
	"github.com/shariqnaiyer/agentbox/internal/transport"
)

func cmdConnect(args []string) error {
	fs := flag.NewFlagSet("connect", flag.ContinueOnError)
	hostFlag := fs.String("host", "", "host to connect to")
	list := fs.Bool("list", false, "list reachable transports and exit")
	pos, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	hosts, _ := config.LoadHosts()

	// Local mode: no paired hosts and no explicit host → attach to a local
	// tmux session (e.g. running `box` directly on the host itself).
	if len(hosts) == 0 && *hostFlag == "" {
		return attachLocalSession(localSession(pos, 0))
	}

	h, sessionArg, err := resolveHost(hosts, *hostFlag, pos)
	if err != nil {
		return err
	}
	session := sessionArg
	if session == "" {
		session = transport.TtydPickerSession
	}

	if *list {
		ks := transport.Probe(h)
		fmt.Printf("transports for %s (%s): ", h.Name, h.Addr())
		if len(ks) == 0 {
			fmt.Println("none reachable")
		} else {
			for _, k := range ks {
				fmt.Printf("%s ", k)
			}
			fmt.Println()
		}
		return nil
	}

	best, ok := transport.Best(h)
	if !ok {
		return fmt.Errorf("no reachable transport for %s (%s). Is the box up and on the tailnet? Try: box host status", h.Name, h.Addr())
	}
	config.SetLastTransport(h.Name, string(best))
	fmt.Fprintf(os.Stderr, "connecting to %s via %s (session %s)...\n", h.Name, best, session)
	return transport.Connect(best, h, session)
}

// resolveHost picks the target host and optional session from flags/positionals.
func resolveHost(hosts []config.Host, hostFlag string, pos []string) (config.Host, string, error) {
	byName := func(n string) (config.Host, bool) {
		for _, h := range hosts {
			if h.Name == n {
				return h, true
			}
		}
		return config.Host{}, false
	}

	if hostFlag != "" {
		h, ok := byName(hostFlag)
		if !ok {
			return config.Host{}, "", fmt.Errorf("unknown host %q (see: box hosts)", hostFlag)
		}
		return h, first(pos, 0), nil
	}

	// positional[0] may be a host name.
	if len(pos) > 0 {
		if h, ok := byName(pos[0]); ok {
			return h, first(pos, 1), nil
		}
		// Not a host name. If there's exactly one host, treat pos[0] as a session.
		if len(hosts) == 1 {
			return hosts[0], pos[0], nil
		}
		return config.Host{}, "", fmt.Errorf("unknown host %q (see: box hosts)", pos[0])
	}

	if len(hosts) == 1 {
		return hosts[0], "", nil
	}
	// Multiple hosts, none specified.
	var names []string
	for _, h := range hosts {
		names = append(names, h.Name)
	}
	return config.Host{}, "", fmt.Errorf("multiple hosts %v — specify one: box <host>", names)
}

// localSession picks a session to attach to on the host itself: an explicit
// positional, else the most-recently-active agent, else the default session.
func localSession(pos []string, idx int) string {
	if s := first(pos, idx); s != "" {
		return s
	}
	if list, _ := agentstatus.List(time.Now()); len(list) > 0 {
		return list[0].Name
	}
	return transport.TtydPickerSession
}

// attachLocalSession execs tmux to attach-or-create a session locally.
func attachLocalSession(session string) error {
	path, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not installed: %w", err)
	}
	argv := []string{path, "new-session", "-A", "-s", session}
	return syscall.Exec(path, argv, os.Environ())
}

func first(s []string, i int) string {
	if i < len(s) {
		return s[i]
	}
	return ""
}
