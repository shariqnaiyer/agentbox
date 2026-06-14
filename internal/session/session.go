// Package session spawns an agent into a tmux session, reusing the ~/.agents
// convention: worktree at wt-<name>, tmux session named <name>, status seeded
// to idle. Shared by `box new` and the supervisor's self-heal so both create
// sessions identically.
package session

import (
	"time"

	"github.com/shariqnaiyer/agentbox/internal/agents"
	"github.com/shariqnaiyer/agentbox/internal/agentstatus"
	"github.com/shariqnaiyer/agentbox/internal/config"
	"github.com/shariqnaiyer/agentbox/internal/tmuxutil"
	"github.com/shariqnaiyer/agentbox/internal/worktree"
)

// Spawn ensures the worktree exists, seeds idle status, and launches the agent
// in a detached tmux session. If unsetAPIKey is true, the agent runs with
// ANTHROPIC_API_KEY stripped so Claude uses the subscription, not metered API.
func Spawn(reg *agents.Registry, m config.ManagedAgent, unsetAPIKey bool) error {
	ag, ok := reg.Get(m.Agent)
	if !ok {
		ag = reg.Default()
	}

	// When a repo is given, ensure the worktree exists (idempotent: Add
	// returns the existing one). We must do this even when m.Worktree is
	// pre-populated by Resolve — otherwise the session launches in a path
	// that was never created.
	workdir := m.Worktree
	if m.Repo != "" {
		wt, err := worktree.Add(m.Repo, m.Name, branchOr(m))
		if err != nil {
			return err
		}
		workdir = wt
	}

	if err := agentstatus.Seed(m.Name, agentstatus.Idle, time.Now()); err != nil {
		return err
	}

	cmd := ag.LaunchCmd(workdir)
	if unsetAPIKey {
		cmd = append([]string{"env", "-u", "ANTHROPIC_API_KEY"}, cmd...)
	}
	return tmuxutil.NewDetached(m.Name, workdir, cmd)
}

// Resolve fills in the worktree path for a managed agent (without creating it),
// so the supervisor can persist a stable record.
func Resolve(m config.ManagedAgent) config.ManagedAgent {
	if m.Worktree == "" && m.Repo != "" {
		m.Worktree = worktree.Path(m.Repo, m.Name)
	}
	if m.Branch == "" {
		m.Branch = m.Name
	}
	return m
}

func branchOr(m config.ManagedAgent) string {
	if m.Branch != "" {
		return m.Branch
	}
	return m.Name
}
