package transport

import (
	"reflect"
	"testing"

	"github.com/shariqnaiyer/agentbox/internal/config"
)

func TestPromote(t *testing.T) {
	ks := []Kind{KindMosh, KindET, KindSSH}
	got := promote(ks, KindSSH)
	want := []Kind{KindSSH, KindMosh, KindET}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("promote = %v, want %v", got, want)
	}
	// Promoting an absent transport is a no-op.
	if got := promote(ks, KindTTYD); !reflect.DeepEqual(got, ks) {
		t.Fatalf("promote(absent) = %v, want %v", got, ks)
	}
}

func TestConnectArgs(t *testing.T) {
	h := config.Host{Name: "mac", TailscaleDNS: "mac.tail.ts.net", SSHUser: "me", TtydPort: 7681}
	sess := "work"

	bin, args, _, err := ConnectArgs(KindMosh, h, sess)
	if err != nil || bin != "mosh" {
		t.Fatalf("mosh: bin=%q err=%v", bin, err)
	}
	if args[0] != "me@mac.tail.ts.net" || args[1] != "--" {
		t.Fatalf("mosh args = %v", args)
	}

	bin, args, _, _ = ConnectArgs(KindET, h, sess)
	if bin != "et" || args[0] != "me@mac.tail.ts.net:2022" || args[1] != "-c" {
		t.Fatalf("et args = %v", args)
	}

	bin, args, _, _ = ConnectArgs(KindSSH, h, sess)
	if bin != "ssh" || args[0] != "-t" || args[1] != "me@mac.tail.ts.net" {
		t.Fatalf("ssh args = %v", args)
	}

	_, _, url, _ := ConnectArgs(KindTTYD, h, sess)
	if url != "http://mac.tail.ts.net:7681" {
		t.Fatalf("ttyd url = %q", url)
	}
}

func TestAddrFallsBackToIP(t *testing.T) {
	h := config.Host{TailscaleIP: "100.1.2.3"}
	if h.Addr() != "100.1.2.3" {
		t.Fatalf("Addr = %q", h.Addr())
	}
}
