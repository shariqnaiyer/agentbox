// Package agents is agentbox's agent-agnostic registry. Each agent type
// (Claude Code, Codex, Gemini) declares how it launches, how it authenticates,
// and whether it's installed. Claude is first-class in v1; the others are
// wired end-to-end with stubbed auth so `box new --agent codex` already runs
// the full pipeline (worktree + tmux session + status files).
package agents

import "sort"

// Agent describes one coding-agent CLI.
type Agent interface {
	ID() string                        // "claude" | "codex" | "gemini"
	DisplayName() string               // human label
	LaunchCmd(workdir string) []string // argv to run inside the tmux session
	Available() bool                   // binary present on this host?
	// Bootstrap performs one-time interactive auth setup on the host.
	Bootstrap() error
}

// Registry holds the known agents.
type Registry struct {
	m     map[string]Agent
	order []string
	def   string
}

// NewRegistry returns the default registry (Claude default).
func NewRegistry() *Registry {
	r := &Registry{m: map[string]Agent{}}
	r.register(claudeAgent{})
	r.register(codexAgent{})
	r.register(geminiAgent{})
	r.def = "claude"
	return r
}

func (r *Registry) register(a Agent) {
	r.m[a.ID()] = a
	r.order = append(r.order, a.ID())
}

// Get returns an agent by id.
func (r *Registry) Get(id string) (Agent, bool) {
	a, ok := r.m[id]
	return a, ok
}

// Default returns the default agent.
func (r *Registry) Default() Agent { return r.m[r.def] }

// List returns agents in registration order.
func (r *Registry) List() []Agent {
	out := make([]Agent, 0, len(r.order))
	ids := append([]string(nil), r.order...)
	sort.Strings(ids)
	for _, id := range ids {
		out = append(out, r.m[id])
	}
	return out
}
