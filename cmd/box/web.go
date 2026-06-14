package box

import (
	"fmt"

	"github.com/shariqnaiyer/agentbox/internal/config"
	"github.com/shariqnaiyer/agentbox/internal/pairing"
	"github.com/shariqnaiyer/agentbox/internal/reach"
)

func cmdWeb(args []string) error {
	cfg, _ := config.Load()
	cfg.WebEnabled = true
	if cfg.TtydPort == 0 {
		cfg.TtydPort = 7681
	}
	if err := config.Save(cfg); err != nil {
		return err
	}
	if !onPath("ttyd") {
		warn("ttyd not installed. Install it (box host init --web), then the supervisor will serve it.")
	}
	ip := reach.IPv4()
	if ip == "" {
		fmt.Println("web transport enabled. Join the tailnet (box host init) to get a reachable URL.")
		return nil
	}
	url := fmt.Sprintf("http://%s:%d", ip, cfg.TtydPort)
	fmt.Printf("web transport enabled. The supervisor serves it on the tailnet at:\n  %s\n", url)
	if qr := pairing.RenderQR(url); qr != "" {
		fmt.Println(qr)
	}
	return nil
}
