package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// claudeSettingsPath is ~/.claude/settings.json.
func claudeSettingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "settings.json")
}

// setStatusPath is the script the hooks call.
func setStatusPath() string {
	return filepath.Join(AgentsDir(), "set-status.sh")
}

// hookEvents maps a Claude Code hook event to the status it records.
var hookEvents = map[string]string{
	"UserPromptSubmit": "active",
	"Stop":             "done",
	"Notification":     "idle",
	"SessionStart":     "idle",
}

// HooksInstalled reports whether the set-status.sh hooks are wired for every
// tracked event in settings.json.
func HooksInstalled() (bool, error) {
	b, err := os.ReadFile(claudeSettingsPath())
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var s map[string]any
	if err := json.Unmarshal(b, &s); err != nil {
		return false, err
	}
	hooks, _ := s["hooks"].(map[string]any)
	if hooks == nil {
		return false, nil
	}
	for event := range hookEvents {
		if !eventHasSetStatus(hooks[event]) {
			return false, nil
		}
	}
	return true, nil
}

// InstallHooks merges the four set-status.sh hooks into settings.json without
// disturbing existing hooks, creating the file if needed.
func InstallHooks() error {
	path := claudeSettingsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	s := map[string]any{}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &s)
	}
	hooks, _ := s["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	script := setStatusPath()
	for event, state := range hookEvents {
		// Drop any prior set-status entries (including legacy flat-format ones
		// that newer Claude Code rejects) and add a correct nested entry.
		arr := withoutSetStatus(asArray(hooks[event]))
		entry := map[string]any{
			"hooks": []any{
				map[string]any{"type": "command", "command": "bash " + script + " " + state},
			},
		}
		hooks[event] = append(arr, entry)
	}
	s["hooks"] = hooks
	out, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0o644)
}

// eventHasSetStatus reports whether any hook entry under an event references
// set-status.sh (handles both the flat {command} and nested {hooks:[...]} forms).
func eventHasSetStatus(v any) bool {
	arr, ok := v.([]any)
	if !ok {
		return false
	}
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if cmd, ok := m["command"].(string); ok && strings.Contains(cmd, "set-status.sh") {
			return true
		}
		if nested, ok := m["hooks"].([]any); ok && nestedHasSetStatus(nested) {
			return true
		}
	}
	return false
}

// asArray coerces a hooks-map value to a slice (nil-safe).
func asArray(v any) []any {
	if arr, ok := v.([]any); ok {
		return arr
	}
	return nil
}

// withoutSetStatus returns arr with every entry referencing set-status.sh
// removed — both the legacy flat {type,command} form and the nested
// {hooks:[...]} form — preserving all other hooks (e.g. the user's own).
func withoutSetStatus(arr []any) []any {
	out := make([]any, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			out = append(out, item)
			continue
		}
		if cmd, ok := m["command"].(string); ok && strings.Contains(cmd, "set-status.sh") {
			continue
		}
		if nested, ok := m["hooks"].([]any); ok && nestedHasSetStatus(nested) {
			continue
		}
		out = append(out, item)
	}
	return out
}

func nestedHasSetStatus(nested []any) bool {
	for _, h := range nested {
		if hm, ok := h.(map[string]any); ok {
			if cmd, ok := hm["command"].(string); ok && strings.Contains(cmd, "set-status.sh") {
				return true
			}
		}
	}
	return false
}
