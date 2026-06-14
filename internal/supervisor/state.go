package supervisor

import (
	"encoding/json"
	"os"
	"time"

	"github.com/shariqnaiyer/agentbox/internal/agentstatus"
	"github.com/shariqnaiyer/agentbox/internal/config"
)

// CompState is a component's health snapshot.
type CompState struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

// State is the supervisor's health snapshot, written atomically to state.json
// and read by `box host status` and the phone/web UI — without attaching.
type State struct {
	Host         string              `json:"host"`
	OS           string              `json:"os"`
	PID          int                 `json:"pid"`
	StartedAt    time.Time           `json:"started_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
	TailscaleDNS string              `json:"tailscale_dns"`
	TailscaleIP  string              `json:"tailscale_ip"`
	KeepAwake    bool                `json:"keep_awake"`
	Components   []CompState         `json:"components"`
	Agents       []agentstatus.Agent `json:"agents"`
}

// WriteState persists the snapshot atomically.
func WriteState(s State) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return config.WriteAtomic(config.StatePath(), append(b, '\n'))
}

// ReadState loads the last-written snapshot.
func ReadState() (State, error) {
	var s State
	b, err := os.ReadFile(config.StatePath())
	if err != nil {
		return s, err
	}
	return s, json.Unmarshal(b, &s)
}

// Healthy reports whether every component is OK.
func (s State) Healthy() bool {
	for _, c := range s.Components {
		if !c.OK {
			return false
		}
	}
	return true
}
