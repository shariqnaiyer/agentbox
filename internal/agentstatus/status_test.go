package agentstatus

import (
	"testing"
	"time"
)

func TestStaleness(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	cases := []struct {
		name    string
		content string
		want    State
	}{
		{"fresh active stays active", "active\t999990", Active}, // 10s old
		{"stale active becomes idle", "active\t999600", Idle},   // 400s old > 300
		{"idle stays idle", "idle\t999000", Idle},               // age irrelevant for idle
		{"done stays done", "done\t1", Done},                    // very old, still done
		{"garbage is unknown", "weird\t999999", State("weird")}, // passthrough unknown state
		{"empty is unknown", "", Unknown},                       // no content
		{"missing ts active stays active", "active", Active},    // no timestamp → not staled
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parse("x", c.content, now)
			if got.State != c.want {
				t.Fatalf("parse(%q) state = %q, want %q", c.content, got.State, c.want)
			}
		})
	}
}

func TestSeedReadList(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	now := time.Unix(2_000_000, 0)
	if err := Seed("alpha", Idle, now); err != nil {
		t.Fatal(err)
	}
	a, err := Read("alpha", now)
	if err != nil {
		t.Fatal(err)
	}
	if a.State != Idle || a.Name != "alpha" {
		t.Fatalf("Read = %+v", a)
	}
	if !a.Updated.Equal(now) {
		t.Fatalf("Updated = %v want %v", a.Updated, now)
	}
	list, err := List(now)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "alpha" {
		t.Fatalf("List = %+v", list)
	}
}
