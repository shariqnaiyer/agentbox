package config

import (
	"testing"
)

func TestConfigRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	c := DefaultConfig()
	c.HostName = "mini"
	c.WebEnabled = true
	if err := Save(c); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.HostName != "mini" || !got.WebEnabled || got.DefaultAgent != "claude" {
		t.Fatalf("Load = %+v", got)
	}
}

func TestHostsUpsertAndTransport(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if err := UpsertHost(Host{Name: "a", TailscaleIP: "100.0.0.1"}); err != nil {
		t.Fatal(err)
	}
	if err := UpsertHost(Host{Name: "b", TailscaleIP: "100.0.0.2"}); err != nil {
		t.Fatal(err)
	}
	// LastTransport learned later is preserved across an upsert that omits it.
	SetLastTransport("a", "mosh")
	if err := UpsertHost(Host{Name: "a", TailscaleIP: "100.0.0.9"}); err != nil {
		t.Fatal(err)
	}
	h, ok := GetHost("a")
	if !ok || h.TailscaleIP != "100.0.0.9" || h.LastTransport != "mosh" {
		t.Fatalf("GetHost(a) = %+v ok=%v", h, ok)
	}
	hs, _ := LoadHosts()
	if len(hs) != 2 {
		t.Fatalf("len hosts = %d", len(hs))
	}
	if err := RemoveHost("a"); err != nil {
		t.Fatal(err)
	}
	if _, ok := GetHost("a"); ok {
		t.Fatal("host a still present after remove")
	}
}

func TestManagedRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if err := AddManaged(ManagedAgent{Name: "x", Agent: "claude", Restart: true}); err != nil {
		t.Fatal(err)
	}
	if err := AddManaged(ManagedAgent{Name: "x", Agent: "codex", Restart: true}); err != nil {
		t.Fatal(err) // replace, not duplicate
	}
	ms, _ := LoadManaged()
	if len(ms) != 1 || ms[0].Agent != "codex" {
		t.Fatalf("managed = %+v", ms)
	}
	if err := RemoveManaged("x"); err != nil {
		t.Fatal(err)
	}
	ms, _ = LoadManaged()
	if len(ms) != 0 {
		t.Fatalf("managed after remove = %+v", ms)
	}
}
