// Package supervisor is the `box host` daemon: a reconciliation loop over a
// set of components. It owns no PTYs (tmux does), so it is crash-safe — if it
// dies and autostart restarts it, the agents and their tmux sessions are
// untouched and it simply re-reconciles. This is what makes the box "never
// break."
package supervisor

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/shariqnaiyer/agentbox/internal/agents"
	"github.com/shariqnaiyer/agentbox/internal/agentstatus"
	"github.com/shariqnaiyer/agentbox/internal/config"
	"github.com/shariqnaiyer/agentbox/internal/platform"
	"github.com/shariqnaiyer/agentbox/internal/reach"
)

// DefaultInterval is the reconcile tick.
const DefaultInterval = 5 * time.Second

// Supervisor reconciles host components on a fixed interval.
type Supervisor struct {
	plat      platform.Platform
	reg       *agents.Registry
	cfg       config.Config
	comps     []Component
	interval  time.Duration
	startedAt time.Time
	keepAwake platform.KeepAwake
}

// New builds a supervisor with the standard component set, in dependency order.
func New(plat platform.Platform, reg *agents.Registry, cfg config.Config) *Supervisor {
	bindIP := reach.IPv4()
	comps := []Component{
		tmuxComponent{},
		tailscaleComponent{},
		agentsComponent{reg: reg, unsetAPIKey: cfg.UnsetAnthropicAPIKey},
		&transportComponent{cfg: cfg, bindIP: bindIP},
	}
	return &Supervisor{
		plat:      plat,
		reg:       reg,
		cfg:       cfg,
		comps:     comps,
		interval:  DefaultInterval,
		startedAt: time.Now(),
	}
}

// Run holds a keep-awake assertion for its lifetime and reconciles until the
// context is cancelled.
func (s *Supervisor) Run(ctx context.Context) error {
	ka, err := s.plat.PreventSleep("agentbox host active")
	if err != nil {
		// Non-fatal: a server that never sleeps doesn't need the assertion.
		logf("keep-awake not asserted: %v", err)
	}
	s.keepAwake = ka
	defer func() {
		if s.keepAwake != nil {
			_ = s.keepAwake.Release()
		}
	}()

	s.tick() // reconcile immediately on startup
	t := time.NewTicker(s.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			s.tick()
		}
	}
}

// tick checks every component in order and repairs the first unhealthy one
// (one repair per tick avoids a thundering herd; the next tick handles the
// next problem).
func (s *Supervisor) tick() {
	var states []CompState
	repaired := false
	for _, c := range s.comps {
		h := c.Check()
		if !h.OK && !repaired {
			if err := c.Repair(); err != nil {
				h = bad("repair failed: " + err.Error())
			} else {
				h = c.Check()
			}
			repaired = true
		}
		states = append(states, CompState{Name: c.Name(), OK: h.OK, Detail: h.Detail})
	}
	s.writeState(states)
}

func (s *Supervisor) writeState(states []CompState) {
	host := s.cfg.HostName
	if host == "" {
		host, _ = os.Hostname()
	}
	agentList, _ := agentstatus.List(time.Now())
	dns, ip := reach.SelfInfo()
	st := State{
		Host:         host,
		OS:           s.plat.OS(),
		PID:          os.Getpid(),
		StartedAt:    s.startedAt,
		UpdatedAt:    time.Now(),
		TailscaleDNS: dns,
		TailscaleIP:  ip,
		KeepAwake:    s.keepAwake != nil,
		Components:   states,
		Agents:       agentList,
	}
	if err := WriteState(st); err != nil {
		logf("write state: %v", err)
	}
}

func logf(format string, args ...any) {
	// The daemon's stdout/stderr is captured to host.log by launchd/systemd.
	fmt.Fprintf(os.Stderr, time.Now().Format(time.RFC3339)+" [supervisor] "+format+"\n", args...)
}
