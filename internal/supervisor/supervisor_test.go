package supervisor

import (
	"testing"

	"github.com/shariqnaiyer/agentbox/internal/config"
	"github.com/shariqnaiyer/agentbox/internal/platform"
)

// fakePlatform satisfies platform.Platform without touching the OS, so the
// reconcile loop is tested OS-independently.
type fakePlatform struct{}

func (fakePlatform) OS() string                                      { return "test" }
func (fakePlatform) PreventSleep(string) (platform.KeepAwake, error) { return fakeKA{}, nil }
func (fakePlatform) DaemonLabel() string                             { return "test" }
func (fakePlatform) InstallAutostart(platform.AutostartSpec) error   { return nil }
func (fakePlatform) RemoveAutostart() error                          { return nil }
func (fakePlatform) AutostartStatus() (platform.AutostartState, error) {
	return platform.AutostartState{}, nil
}
func (fakePlatform) Elevate([]string, string) error  { return nil }
func (fakePlatform) PackageManager() platform.PkgMgr { return platform.PkgMgr{} }
func (fakePlatform) InstallPackages(...string) error { return nil }

type fakeKA struct{}

func (fakeKA) Release() error { return nil }

type fakeComp struct {
	name    string
	healthy bool
	repairs int
}

func (f *fakeComp) Name() string { return f.name }
func (f *fakeComp) Check() Health {
	if f.healthy {
		return ok("up")
	}
	return bad("down")
}
func (f *fakeComp) Repair() error {
	f.repairs++
	f.healthy = true
	return nil
}

// TestTickRepairsOneAtATime verifies the supervisor repairs only the first
// unhealthy component per tick (no thundering herd), converging over ticks.
func TestTickRepairsOneAtATime(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	a := &fakeComp{name: "a", healthy: false}
	b := &fakeComp{name: "b", healthy: false}
	s := &Supervisor{
		plat:  fakePlatform{},
		cfg:   config.DefaultConfig(),
		comps: []Component{a, b},
	}

	s.tick() // should repair only 'a'
	if a.repairs != 1 || b.repairs != 0 {
		t.Fatalf("after tick 1: a.repairs=%d b.repairs=%d, want 1,0", a.repairs, b.repairs)
	}
	if !a.healthy || b.healthy {
		t.Fatalf("after tick 1: a.healthy=%v b.healthy=%v, want true,false", a.healthy, b.healthy)
	}

	s.tick() // now 'a' is fine, should repair 'b'
	if b.repairs != 1 || !b.healthy {
		t.Fatalf("after tick 2: b.repairs=%d b.healthy=%v, want 1,true", b.repairs, b.healthy)
	}

	// State file should reflect a healthy system now.
	st, err := ReadState()
	if err != nil {
		t.Fatalf("ReadState: %v", err)
	}
	if !st.Healthy() {
		t.Fatalf("state not healthy: %+v", st.Components)
	}
}
