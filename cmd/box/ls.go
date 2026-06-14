package box

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/shariqnaiyer/agentbox/internal/agentstatus"
	"github.com/shariqnaiyer/agentbox/internal/tmuxutil"
)

func cmdLs(args []string) error {
	fs := flag.NewFlagSet("ls", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	now := time.Now()
	list, err := agentstatus.List(now)
	if err != nil {
		return err
	}
	if *asJSON {
		b, _ := json.MarshalIndent(list, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	if len(list) == 0 {
		fmt.Println("no agent sessions. start one with:  box new <name>")
		return nil
	}
	// Mark which sessions are actually live in tmux right now.
	live := map[string]bool{}
	if names, err := tmuxutil.ListSessions(); err == nil {
		for _, n := range names {
			live[n] = true
		}
	}
	for _, a := range list {
		liveMark := " "
		if live[a.Name] {
			liveMark = "•"
		}
		fmt.Printf("%s %s %-18s %-7s %s\n", a.Icon(), liveMark, a.Name, a.State, ago(a.Updated, now))
	}
	return nil
}

func ago(t, now time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
