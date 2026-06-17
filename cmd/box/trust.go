package box

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/shariqnaiyer/agentbox/internal/config"
)

// cmdTrust sets up passwordless SSH from this client to a host using a
// dedicated agentbox key — bypassing ssh-agents (e.g. 1Password) that offer
// many keys and trip sshd's MaxAuthFailures. One-time: prompts for the host
// password to install the key, then `box` connects with no further prompts.
func cmdTrust(args []string) error {
	pos, err := parseFlags(flag.NewFlagSet("trust", flag.ContinueOnError), args)
	if err != nil {
		return err
	}
	hosts, _ := config.LoadHosts()
	if len(hosts) == 0 {
		return fmt.Errorf("no paired hosts. Run `box pair <code>` first.")
	}
	h, _, err := resolveHost(hosts, "", pos)
	if err != nil {
		return err
	}

	// 1. Ensure the agentbox key exists.
	key := config.SSHKeyPath()
	if _, err := os.Stat(key); err != nil {
		if err := config.EnsureDir(); err != nil {
			return err
		}
		fmt.Println("Generating agentbox SSH key...")
		gen := exec.Command("ssh-keygen", "-t", "ed25519", "-f", key, "-N", "", "-C", "agentbox")
		gen.Stdout, gen.Stderr = os.Stdout, os.Stderr
		if err := gen.Run(); err != nil {
			return fmt.Errorf("ssh-keygen: %w", err)
		}
	}
	pub, err := os.ReadFile(key + ".pub")
	if err != nil {
		return fmt.Errorf("read public key: %w", err)
	}

	// 2. Install the public key on the host (password auth, this once).
	user := h.SSHUser
	target := h.Addr()
	if user != "" {
		target = user + "@" + target
	}
	fmt.Printf("Installing key on %s (enter the host's login password once)...\n", h.Name)
	remote := "mkdir -p ~/.ssh && chmod 700 ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys"
	ssh := exec.Command("ssh",
		"-o", "IdentityAgent=none",
		"-o", "IdentitiesOnly=yes",
		"-o", "PubkeyAuthentication=no",
		"-o", "PreferredAuthentications=password",
		"-o", "StrictHostKeyChecking=accept-new",
		target, remote,
	)
	ssh.Stdin = strings.NewReader(string(pub))
	ssh.Stdout, ssh.Stderr = os.Stdout, os.Stderr
	if err := ssh.Run(); err != nil {
		return fmt.Errorf("install key: %w (is Remote Login on, and the password correct?)", err)
	}

	config.SetLastTransport(h.Name, "") // re-probe next connect
	fmt.Printf("\nTrusted %s. Connect with no password:  box %s\n", h.Name, h.Name)
	return nil
}
