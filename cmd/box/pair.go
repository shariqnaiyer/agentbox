package box

import (
	"fmt"

	"github.com/shariqnaiyer/agentbox/internal/config"
	"github.com/shariqnaiyer/agentbox/internal/pairing"
)

func cmdPair(args []string) error {
	// No code → host side: print our pairing code + QR.
	if len(args) == 0 {
		cfg, _ := config.Load()
		printPairing(cfg)
		return nil
	}
	// Code given → client side: record the host.
	p, err := pairing.Decode(args[0])
	if err != nil {
		return err
	}
	h := config.Host{
		Name:         p.HostName,
		TailscaleDNS: p.TailscaleDNS,
		TailscaleIP:  p.TailscaleIP,
		SSHUser:      p.SSHUser,
		Transports:   p.Transports,
		TtydPort:     p.TtydPort,
	}
	if h.Name == "" {
		return fmt.Errorf("pairing code has no host name")
	}
	if err := config.UpsertHost(h); err != nil {
		return err
	}
	fmt.Printf("paired host %q (%s). Connect with:  box %s\n", h.Name, h.Addr(), h.Name)
	return nil
}
