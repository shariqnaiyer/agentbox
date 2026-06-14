package box

import (
	"fmt"

	"github.com/shariqnaiyer/agentbox/internal/config"
)

func cmdHosts(args []string) error {
	if len(args) >= 1 && args[0] == "rm" {
		if len(args) < 2 {
			return fmt.Errorf("usage: box hosts rm <name>")
		}
		if err := config.RemoveHost(args[1]); err != nil {
			return err
		}
		fmt.Printf("removed host %q\n", args[1])
		return nil
	}
	hs, err := config.LoadHosts()
	if err != nil {
		return err
	}
	if len(hs) == 0 {
		fmt.Println("no paired hosts. On a host run `box pair`, then `box pair <code>` here.")
		return nil
	}
	for _, h := range hs {
		last := h.LastTransport
		if last == "" {
			last = "-"
		}
		fmt.Printf("%-16s %-34s last:%s\n", h.Name, h.Addr(), last)
	}
	return nil
}
