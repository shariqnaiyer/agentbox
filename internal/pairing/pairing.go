// Package pairing handles client/host pairing. Trust is rooted in Tailscale:
// both ends are already on the same tailnet (BYO account in v1), so pairing is
// discovery + naming, not key exchange. The host emits a short code (and a QR
// for phones) carrying its address and transports; the client records it.
package pairing

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Payload is the information a client needs to connect to a host.
type Payload struct {
	HostName     string   `json:"h"`
	TailscaleDNS string   `json:"d"`
	TailscaleIP  string   `json:"i"`
	SSHUser      string   `json:"u"`
	Transports   []string `json:"t"`
	TtydPort     int      `json:"p,omitempty"`
}

// Encode renders a payload as a compact, copy-pasteable code.
func Encode(p Payload) string {
	b, _ := json.Marshal(p)
	return "box1_" + base64.RawURLEncoding.EncodeToString(b)
}

// Decode parses a pairing code.
func Decode(code string) (Payload, error) {
	var p Payload
	code = strings.TrimSpace(code)
	if !strings.HasPrefix(code, "box1_") {
		return p, fmt.Errorf("not an agentbox pairing code")
	}
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(code, "box1_"))
	if err != nil {
		return p, fmt.Errorf("bad pairing code: %w", err)
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return p, fmt.Errorf("bad pairing code: %w", err)
	}
	return p, nil
}

// RenderQR returns a terminal QR for the code using the system `qrencode` if
// present (brew/apt: qrencode). If unavailable, returns "" and callers fall
// back to showing the typeable code — a deliberate zero-Go-dependency choice.
func RenderQR(text string) string {
	if _, err := exec.LookPath("qrencode"); err != nil {
		return ""
	}
	out, err := exec.Command("qrencode", "-t", "ANSIUTF8", "-m", "1", text).Output()
	if err != nil {
		return ""
	}
	return string(out)
}
