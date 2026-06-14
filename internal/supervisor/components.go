package supervisor

import (
	"fmt"

	"github.com/shariqnaiyer/agentbox/internal/agents"
	"github.com/shariqnaiyer/agentbox/internal/config"
	"github.com/shariqnaiyer/agentbox/internal/reach"
	"github.com/shariqnaiyer/agentbox/internal/session"
	"github.com/shariqnaiyer/agentbox/internal/tmuxutil"
	"github.com/shariqnaiyer/agentbox/internal/transport"
)

// tmuxComponent ensures the tmux server (the durable session host) is up.
// It must be healthy before agents can run.
type tmuxComponent struct{}

func (tmuxComponent) Name() string { return "tmux" }
func (tmuxComponent) Check() Health {
	if !tmuxutil.Available() {
		return bad("tmux not installed")
	}
	if !tmuxutil.ServerRunning() {
		return bad("tmux server down")
	}
	return ok("server up")
}
func (tmuxComponent) Repair() error { return tmuxutil.StartServer() }

// tailscaleComponent ensures reachability. Repair only runs a plain `up`,
// never --reset/--force-reauth, which would drop a remote session.
type tailscaleComponent struct{}

func (tailscaleComponent) Name() string { return "tailscale" }
func (tailscaleComponent) Check() Health {
	if !reach.Installed() {
		return bad("tailscale not installed")
	}
	if !reach.IsRunning() {
		return bad("backend not Running")
	}
	return ok(reach.IPv4())
}
func (tailscaleComponent) Repair() error { return reach.Up("", "") }

// agentsComponent keeps the DECLARED managed agents alive. It never spawns
// agents the user didn't ask for; it only re-spawns a managed session that
// died and is marked Restart.
type agentsComponent struct {
	reg         *agents.Registry
	unsetAPIKey bool
}

func (agentsComponent) Name() string { return "agents" }

func (c agentsComponent) Check() Health {
	managed, err := config.LoadManaged()
	if err != nil {
		return bad("read managed: " + err.Error())
	}
	if len(managed) == 0 {
		return ok("no managed agents")
	}
	var dead []string
	for _, m := range managed {
		if m.Restart && !tmuxutil.HasSession(m.Name) {
			dead = append(dead, m.Name)
		}
	}
	if len(dead) > 0 {
		return bad(fmt.Sprintf("down: %v", dead))
	}
	return ok(fmt.Sprintf("%d alive", len(managed)))
}

func (c agentsComponent) Repair() error {
	managed, err := config.LoadManaged()
	if err != nil {
		return err
	}
	for _, m := range managed {
		if m.Restart && !tmuxutil.HasSession(m.Name) {
			if err := session.Spawn(c.reg, m, c.unsetAPIKey); err != nil {
				return fmt.Errorf("respawn %s: %w", m.Name, err)
			}
		}
	}
	return nil
}

// transportComponent keeps the ttyd web server alive when web access is on.
// mosh/et are launched per-connection by clients, so there's nothing to keep
// running for them — this component only manages the optional ttyd server.
type transportComponent struct {
	cfg    config.Config
	bindIP string
	server *ttydServer
}

func (transportComponent) Name() string { return "transport" }

func (c *transportComponent) Check() Health {
	if !c.cfg.WebEnabled {
		return ok("web disabled")
	}
	if c.server != nil && c.server.Running() {
		return ok(fmt.Sprintf("ttyd :%d", c.cfg.TtydPort))
	}
	return bad("ttyd down")
}

func (c *transportComponent) Repair() error {
	if !c.cfg.WebEnabled {
		return nil
	}
	if c.server == nil {
		c.server = newTtydServer(c.cfg.TtydPort, c.bindIP)
	}
	return c.server.Start(transport.TtydPickerSession)
}
