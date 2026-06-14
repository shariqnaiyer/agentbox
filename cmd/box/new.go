package box

import (
	"flag"
	"fmt"
	"os"

	"github.com/shariqnaiyer/agentbox/internal/agents"
	"github.com/shariqnaiyer/agentbox/internal/config"
	"github.com/shariqnaiyer/agentbox/internal/session"
)

func cmdNew(args []string) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	agent := fs.String("agent", "claude", "agent type: claude|codex|gemini")
	repo := fs.String("repo", "", "git repo to derive the worktree from (default: cwd)")
	branch := fs.String("branch", "", "worktree branch (default: <name>)")
	noManage := fs.Bool("no-manage", false, "don't let the supervisor keep this session alive")
	var attach bool
	fs.BoolVar(&attach, "attach", false, "attach after spawning")
	fs.BoolVar(&attach, "a", false, "attach after spawning (shorthand)")
	pos, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	if len(pos) == 0 {
		return fmt.Errorf("usage: box new <name> [--agent ...] [--repo ...] [--attach]")
	}
	name := pos[0]

	reg := agents.NewRegistry()
	ag, ok := reg.Get(*agent)
	if !ok {
		return fmt.Errorf("unknown agent %q (have: claude, codex, gemini)", *agent)
	}
	if !ag.Available() {
		warn("%s binary not found on PATH; the session will start but the agent may not run", ag.DisplayName())
	}

	repoPath := *repo
	if repoPath == "" {
		repoPath, _ = os.Getwd()
	}
	m := session.Resolve(config.ManagedAgent{
		Name:    name,
		Agent:   *agent,
		Repo:    repoPath,
		Branch:  *branch,
		Restart: !*noManage,
	})

	cfg, _ := config.Load()
	if err := session.Spawn(reg, m, cfg.UnsetAnthropicAPIKey); err != nil {
		return err
	}
	if !*noManage {
		if err := config.AddManaged(m); err != nil {
			warn("record managed agent: %v", err)
		}
	}
	fmt.Printf("spawned %q (agent: %s, session: %s)\n", name, ag.DisplayName(), name)
	if m.Worktree != "" {
		fmt.Printf("  worktree: %s\n", m.Worktree)
	}
	fmt.Printf("attach with:  box %s\n", name)

	if attach {
		return attachLocalSession(name)
	}
	return nil
}
