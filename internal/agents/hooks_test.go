package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstallHooksNestedAndMigrate verifies InstallHooks writes the nested
// {hooks:[...]} format Claude Code requires, migrates a legacy flat set-status
// entry, and preserves the user's own (foreign) hooks.
func TestInstallHooksNestedAndMigrate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	pre := `{
	  "hooks": {
	    "UserPromptSubmit": [
	      {"type":"command","command":"bash ` + home + `/.agents/set-status.sh active"}
	    ],
	    "SessionStart": [
	      {"matcher":"*","type":"command","command":"bash /x/herdr.sh session"}
	    ]
	  }
	}`
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(pre), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := InstallHooks(); err != nil {
		t.Fatal(err)
	}

	b, _ := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	var s map[string]any
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatalf("settings not valid JSON: %v", err)
	}
	hooks := s["hooks"].(map[string]any)

	for event := range hookEvents {
		arr, _ := hooks[event].([]any)
		nestedCount := 0
		for _, item := range arr {
			m := item.(map[string]any)
			if nested, ok := m["hooks"].([]any); ok && nestedHasSetStatus(nested) {
				nestedCount++
			}
			if cmd, ok := m["command"].(string); ok && strings.Contains(cmd, "set-status.sh") {
				t.Fatalf("event %s still has a flat set-status entry", event)
			}
		}
		if nestedCount != 1 {
			t.Fatalf("event %s has %d nested set-status entries, want 1", event, nestedCount)
		}
	}

	// Foreign herdr hook must survive migration.
	foundHerdr := false
	for _, item := range hooks["SessionStart"].([]any) {
		if cmd, ok := item.(map[string]any)["command"].(string); ok && strings.Contains(cmd, "herdr.sh") {
			foundHerdr = true
		}
	}
	if !foundHerdr {
		t.Fatal("foreign herdr hook was dropped")
	}

	if ok, _ := HooksInstalled(); !ok {
		t.Fatal("HooksInstalled = false after install")
	}
}
