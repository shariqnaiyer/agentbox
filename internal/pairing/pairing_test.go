package pairing

import "testing"

func TestEncodeDecodeRoundTrip(t *testing.T) {
	p := Payload{
		HostName:     "mini",
		TailscaleDNS: "mini.tail.ts.net",
		TailscaleIP:  "100.1.2.3",
		SSHUser:      "shariq",
		Transports:   []string{"mosh", "et", "ssh"},
		TtydPort:     7681,
	}
	code := Encode(p)
	got, err := Decode(code)
	if err != nil {
		t.Fatal(err)
	}
	if got.HostName != p.HostName || got.TailscaleIP != p.TailscaleIP || got.TtydPort != 7681 {
		t.Fatalf("round trip mismatch: %+v", got)
	}
	if len(got.Transports) != 3 || got.Transports[0] != "mosh" {
		t.Fatalf("transports = %v", got.Transports)
	}
}

func TestDecodeRejectsGarbage(t *testing.T) {
	if _, err := Decode("not-a-code"); err == nil {
		t.Fatal("expected error for non-prefixed code")
	}
	if _, err := Decode("box1_!!!notbase64"); err == nil {
		t.Fatal("expected error for bad base64")
	}
}
