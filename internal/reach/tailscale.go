// Package reach wraps the Tailscale CLI — the reachability layer. agentbox
// drives the user's own tailscaled (BYO account in v1) rather than embedding
// it, so connectivity, NAT traversal, and DERP relays are all handled by
// Tailscale and we add zero networking code.
package reach

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
)

// Status is the subset of `tailscale status --json` we care about.
type Status struct {
	BackendState string `json:"BackendState"`
	Self         struct {
		DNSName      string   `json:"DNSName"`
		HostName     string   `json:"HostName"`
		TailscaleIPs []string `json:"TailscaleIPs"`
		Online       bool     `json:"Online"`
	} `json:"Self"`
}

// Bin locates the tailscale CLI: PATH first, then well-known install paths
// (the macOS app bundle, standard Linux location).
func Bin() string {
	if p, err := exec.LookPath("tailscale"); err == nil {
		return p
	}
	for _, c := range []string{
		"/Applications/Tailscale.app/Contents/MacOS/Tailscale",
		"/usr/bin/tailscale",
		"/usr/local/bin/tailscale",
		"/opt/homebrew/bin/tailscale",
	} {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c
		}
	}
	return "tailscale"
}

// Installed reports whether the tailscale CLI is present.
func Installed() bool {
	if _, err := exec.LookPath("tailscale"); err == nil {
		return true
	}
	bin := Bin()
	_, err := os.Stat(bin)
	return err == nil
}

// GetStatus returns parsed `tailscale status --json`.
func GetStatus() (*Status, error) {
	out, err := exec.Command(Bin(), "status", "--json").Output()
	if err != nil {
		return nil, err
	}
	var s Status
	if err := json.Unmarshal(out, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// IsRunning reports whether the tailscale backend is up with an address.
func IsRunning() bool {
	s, err := GetStatus()
	if err != nil {
		return false
	}
	return s.BackendState == "Running" && len(s.Self.TailscaleIPs) > 0
}

// Up joins the tailnet. authKey may be empty (interactive/already-authed);
// hostname sets the MagicDNS name. We never pass --reset or --force-reauth:
// doing so over a remote session would drop the very link we're using.
func Up(authKey, hostname string) error {
	args := []string{"up"}
	if authKey != "" {
		args = append(args, "--authkey="+authKey)
	}
	if hostname != "" {
		args = append(args, "--hostname="+hostname)
	}
	c := exec.Command(Bin(), args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// EnableSSH turns on Tailscale SSH, so clients on the tailnet authenticate via
// their tailnet identity instead of SSH keys/passwords. This is the cleanest
// auth path for agentbox's ssh/mosh transports — no key management at all.
func EnableSSH() error {
	c := exec.Command(Bin(), "set", "--ssh")
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	return c.Run()
}

// SSHEnabled reports whether this node advertises Tailscale SSH.
func SSHEnabled() bool {
	out, err := exec.Command(Bin(), "debug", "prefs").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "\"RunSSH\": true") || strings.Contains(string(out), "RunSSH:true")
}

// DNSName returns the host's MagicDNS name without the trailing dot.
func DNSName() string {
	s, err := GetStatus()
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(s.Self.DNSName, ".")
}

// IPv4 returns the host's 100.x Tailscale address, if any.
func IPv4() string {
	s, err := GetStatus()
	if err != nil {
		return ""
	}
	for _, ip := range s.Self.TailscaleIPs {
		if strings.Count(ip, ".") == 3 {
			return ip
		}
	}
	return ""
}

// Ping reports whether a peer is reachable over the tailnet.
func Ping(addr string) bool {
	return exec.Command(Bin(), "ping", "--c", "1", "--timeout", "3s", addr).Run() == nil
}

// SelfInfo returns (MagicDNS name, IPv4) from a single status call — cheaper
// than calling DNSName and IPv4 separately when you need both.
func SelfInfo() (dns, ipv4 string) {
	s, err := GetStatus()
	if err != nil {
		return "", ""
	}
	dns = strings.TrimSuffix(s.Self.DNSName, ".")
	for _, ip := range s.Self.TailscaleIPs {
		if strings.Count(ip, ".") == 3 {
			ipv4 = ip
			break
		}
	}
	return dns, ipv4
}
