package box

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

// onPath reports whether a binary is on PATH.
func onPath(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

// parseFlags parses a flag set allowing flags and positionals to be
// interspersed. Go's flag package stops at the first positional, so
// `box new name --agent codex` would otherwise ignore the flags. This
// re-parses the remainder after each positional, collecting positionals.
func parseFlags(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			break
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
	return positional, nil
}

// Version is the build version, overridable via -ldflags.
var Version = "0.1.0-dev"

var stdin = bufio.NewReader(os.Stdin)

// prompt reads a single line, showing label and an optional default.
func prompt(label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := stdin.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

// confirm asks a yes/no question (default yes).
func confirm(label string) bool {
	fmt.Printf("%s [Y/n]: ", label)
	line, _ := stdin.ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "" || line == "y" || line == "yes"
}

func currentUsername() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return os.Getenv("USER")
}

func hostnameOr(def string) string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return strings.TrimSuffix(h, ".local")
	}
	return def
}

func warn(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "  ! "+format+"\n", a...)
}

func step(format string, a ...any) {
	fmt.Printf("→ "+format+"\n", a...)
}

func okmsg(format string, a ...any) {
	fmt.Printf("  ✓ "+format+"\n", a...)
}
