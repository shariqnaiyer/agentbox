package box

import (
	"flag"
	"fmt"

	"github.com/shariqnaiyer/agentbox/internal/config"
	"github.com/shariqnaiyer/agentbox/internal/tmuxutil"
	"github.com/shariqnaiyer/agentbox/internal/worktree"
)

func cmdKill(args []string) error {
	fs := flag.NewFlagSet("kill", flag.ContinueOnError)
	rmWorktree := fs.Bool("worktree", false, "also remove the git worktree")
	pos, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	if len(pos) == 0 {
		return fmt.Errorf("usage: box kill <name> [--worktree]")
	}
	name := pos[0]

	// Find the managed record (for worktree cleanup) before we drop it.
	var m config.ManagedAgent
	if ms, err := config.LoadManaged(); err == nil {
		for _, x := range ms {
			if x.Name == name {
				m = x
			}
		}
	}

	if err := tmuxutil.KillSession(name); err != nil {
		warn("kill session: %v", err)
	}
	_ = config.RemoveManaged(name)

	if *rmWorktree && m.Repo != "" && m.Worktree != "" {
		if err := worktree.Remove(m.Repo, m.Worktree); err != nil {
			warn("remove worktree: %v", err)
		} else {
			fmt.Printf("removed worktree %s\n", m.Worktree)
		}
	}
	fmt.Printf("killed %q\n", name)
	return nil
}
