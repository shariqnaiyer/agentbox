#!/usr/bin/env bash
# Installed by agentbox on fresh hosts. Renders agent statuses for the tmux
# status-right line. Mirrors the ~/.agents toolkit (300s staleness rule).
dir="$HOME/.agents/status"
[ -d "$dir" ] || exit 0
out=""
now="$(date +%s)"
for f in "$dir"/*; do
  [ -e "$f" ] || continue
  name="$(basename "$f")"
  state="$(cut -f1 "$f" 2>/dev/null)"
  ts="$(cut -f2 "$f" 2>/dev/null)"
  if [ "$state" = "active" ] && [ -n "$ts" ] && [ $((now - ts)) -gt 300 ]; then
    state="idle"
  fi
  case "$state" in
    active) icon="#[fg=green]●" ;;
    idle)   icon="#[fg=yellow]○" ;;
    done)   icon="#[fg=cyan]✓" ;;
    *)      icon="#[fg=red]✗" ;;
  esac
  out="$out $icon #[fg=white]$name#[default]"
done
echo "$out "
