package config

import (
	"encoding/json"
	"os"
	"sort"
)

// Host is a paired remote box recorded on a client.
type Host struct {
	Name          string   `json:"name"`
	TailscaleDNS  string   `json:"tailscale_dns"`
	TailscaleIP   string   `json:"tailscale_ip"`
	SSHUser       string   `json:"ssh_user"`
	Transports    []string `json:"transports"`
	TtydPort      int      `json:"ttyd_port,omitempty"`
	LastTransport string   `json:"last_transport,omitempty"`
}

// Addr returns the preferred address to dial: the MagicDNS name if known,
// otherwise the raw Tailscale IP.
func (h Host) Addr() string {
	if h.TailscaleDNS != "" {
		return h.TailscaleDNS
	}
	return h.TailscaleIP
}

// LoadHosts reads hosts.json (empty list if absent).
func LoadHosts() ([]Host, error) {
	b, err := os.ReadFile(hostsPath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var hs []Host
	if err := json.Unmarshal(b, &hs); err != nil {
		return nil, err
	}
	return hs, nil
}

// SaveHosts writes hosts.json atomically.
func SaveHosts(hs []Host) error {
	if err := EnsureDir(); err != nil {
		return err
	}
	return WriteAtomic(hostsPath(), mustJSON(hs))
}

// UpsertHost adds or replaces a host by name and persists the list.
func UpsertHost(h Host) error {
	hs, err := LoadHosts()
	if err != nil {
		return err
	}
	found := false
	for i := range hs {
		if hs[i].Name == h.Name {
			// preserve a previously learned transport unless the new one sets it
			if h.LastTransport == "" {
				h.LastTransport = hs[i].LastTransport
			}
			hs[i] = h
			found = true
			break
		}
	}
	if !found {
		hs = append(hs, h)
	}
	sort.Slice(hs, func(i, j int) bool { return hs[i].Name < hs[j].Name })
	return SaveHosts(hs)
}

// RemoveHost deletes a host by name.
func RemoveHost(name string) error {
	hs, err := LoadHosts()
	if err != nil {
		return err
	}
	out := hs[:0]
	for _, h := range hs {
		if h.Name != name {
			out = append(out, h)
		}
	}
	return SaveHosts(out)
}

// GetHost returns a host by name.
func GetHost(name string) (Host, bool) {
	hs, _ := LoadHosts()
	for _, h := range hs {
		if h.Name == name {
			return h, true
		}
	}
	return Host{}, false
}

// SetLastTransport records the transport that last worked for a host.
func SetLastTransport(name, transport string) {
	hs, err := LoadHosts()
	if err != nil {
		return
	}
	for i := range hs {
		if hs[i].Name == name {
			hs[i].LastTransport = transport
			_ = SaveHosts(hs)
			return
		}
	}
}

// ManagedAgent is an agent session the supervisor keeps alive.
type ManagedAgent struct {
	Name     string `json:"name"`     // == tmux session name
	Agent    string `json:"agent"`    // registry id: claude|codex|gemini
	Repo     string `json:"repo"`     // git repo the worktree derives from
	Branch   string `json:"branch"`   // worktree branch
	Worktree string `json:"worktree"` // absolute worktree path
	Restart  bool   `json:"restart"`  // re-spawn if the session dies
}

// LoadManaged reads the supervisor's managed-agent declarations.
func LoadManaged() ([]ManagedAgent, error) {
	b, err := os.ReadFile(managedPath())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ms []ManagedAgent
	if err := json.Unmarshal(b, &ms); err != nil {
		return nil, err
	}
	return ms, nil
}

// SaveManaged writes the managed-agent list atomically.
func SaveManaged(ms []ManagedAgent) error {
	if err := EnsureDir(); err != nil {
		return err
	}
	return WriteAtomic(managedPath(), mustJSON(ms))
}

// AddManaged adds or replaces a managed agent by name.
func AddManaged(m ManagedAgent) error {
	ms, err := LoadManaged()
	if err != nil {
		return err
	}
	for i := range ms {
		if ms[i].Name == m.Name {
			ms[i] = m
			return SaveManaged(ms)
		}
	}
	return SaveManaged(append(ms, m))
}

// RemoveManaged drops a managed agent by name.
func RemoveManaged(name string) error {
	ms, err := LoadManaged()
	if err != nil {
		return err
	}
	out := ms[:0]
	for _, m := range ms {
		if m.Name != name {
			out = append(out, m)
		}
	}
	return SaveManaged(out)
}
