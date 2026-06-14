// Package box implements the `box` CLI. Dispatch is hand-rolled (stdlib only)
// so agentbox has zero external dependencies and builds offline anywhere.
package box

import (
	"fmt"
	"os"
)

const usage = `box — run a coding agent on any always-on Unix box, reach it from anywhere.

USAGE
  box                       Connect to your box and attach the agent (the money command)
  box <host> [session]      Connect to a specific paired host / session

HOST SETUP (run on the always-on box)
  box host init             One-time setup: deps, Tailscale, autostart, agent auth, pairing
  box host                  Run the supervisor daemon (autostart launches this)
  box host status [--json]  Show host health without attaching

AGENTS
  box new <name> [--agent claude|codex|gemini] [--repo PATH] [--branch B] [--attach]
  box ls [--json]           List agent sessions and their status
  box kill <name> [--worktree]

CONNECT / PAIR
  box connect [host] [session] [--list]
  box pair                  (on host) print a pairing code + QR
  box pair <code>           (on client) record a host
  box hosts [rm <name>]
  box web                   Enable the browser (ttyd) transport

OTHER
  box doctor                Diagnose the host
  box version
`

// commands maps a subcommand name to its handler.
var commands = map[string]func([]string) error{
	"host":    cmdHost,
	"new":     cmdNew,
	"ls":      cmdLs,
	"connect": cmdConnect,
	"pair":    cmdPair,
	"hosts":   cmdHosts,
	"kill":    cmdKill,
	"web":     cmdWeb,
	"doctor":  cmdDoctor,
}

// Execute is the CLI entry point.
func Execute() {
	args := os.Args[1:]

	// Bare `box` is the money command: connect + attach.
	if len(args) == 0 {
		exit(cmdConnect(nil))
	}

	switch args[0] {
	case "-h", "--help", "help":
		fmt.Print(usage)
		return
	case "-v", "--version", "version":
		fmt.Printf("box %s\n", Version)
		return
	}

	if fn, ok := commands[args[0]]; ok {
		exit(fn(args[1:]))
	}

	// Unknown first arg with no leading dash → treat as a host (or session)
	// target for connect: `box mymac`, `box mymac mytask`.
	if args[0][0] != '-' {
		exit(cmdConnect(args))
	}

	fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", args[0], usage)
	os.Exit(2)
}

func exit(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
