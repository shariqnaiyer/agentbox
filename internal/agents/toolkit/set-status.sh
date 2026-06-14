#!/usr/bin/env bash
# Installed by agentbox on fresh hosts. Called by Claude Code hooks to record
# an agent's status, keyed by tmux session name. Mirrors the ~/.agents toolkit.
# Usage: set-status.sh <active|idle|done>
state="$1"
dir="$HOME/.agents/status"
mkdir -p "$dir"
name="$(tmux display -p '#S' 2>/dev/null)"
[ -z "$name" ] && name="${TMUX_PANE:-unknown}"
printf '%s\t%s\n' "$state" "$(date +%s)" > "$dir/$name"
