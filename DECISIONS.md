# Decisions

Autonomous engineering decisions made while building agentbox, with rationale.
These deviate from or sharpen the original plan; recorded here instead of asking.

## Zero external Go dependencies (stdlib only)
The plan suggested cobra/toml/go-qrcode. Instead agentbox uses **only the Go
standard library**:
- CLI dispatch is hand-rolled (`cmd/box/root.go`) instead of cobra.
- Config/state files are **JSON** (`encoding/json`) instead of TOML.
- QR codes are rendered via the system `qrencode` binary if present, with a
  typeable-code fallback, instead of a Go QR library.

**Why:** the product's whole pitch is "installs on any box." A binary with no
module dependencies builds offline, has no supply chain, and cross-compiles
trivially (`CGO_ENABLED=0`). The cost (a ~30-line flag dispatcher, JSON instead
of TOML) is trivial next to that benefit.

## Module path & binary name
- Module: `github.com/shariqnaiyer/agentbox`.
- Binary/command: **`box`** — short, and (unlike `cc`, the original idea) it does
  **not** collide with anything on a default Unix PATH. `cc` is the C compiler.

## One binary, host + client
A single `box` binary is both the host daemon (`box host`) and the client
(`box`, `box new`, …). Simpler distribution; the subcommand selects the role.

## v1 OS scope: Unix only (macOS + Linux)
Native Windows is deferred (no tmux / different PTY model). Linux covers
Raspberry Pi and BSD-likes in practice. The OS-specific surface is isolated to
`internal/platform/{darwin,linux}.go` behind one interface — adding an OS later
means adding one file.

## Platform adapter is the only OS-specific code
Three concerns differ by OS and nothing else: keep-awake, autostart, privileged
install. Verified by cross-compiling `GOOS=linux` on a Mac in CI — both
implementations always compile.

## Tailscale: bring-your-own account (v1)
`box host init` runs `tailscale up` with a user-pasted auth key rather than
embedding tailscaled under an agentbox-owned namespace. For an OSS/personal
tool this avoids operating a tailnet on users' behalf (a cost/ToS burden).
Embedded/namespaced Tailscale is a product-tier concern, not v1.

## Persistence respawn never uses `tailscale --reset`
The tailscale component's Repair only runs a plain `up`. `--reset` /
`--force-reauth` over a remote session would drop the very link in use.

## Supervisor owns no PTYs
The daemon is a reconciliation loop over components; tmux owns the agent
processes. So the supervisor can crash and be restarted by launchd/systemd with
zero agent interruption — it just re-reconciles from `managed.json` + live tmux.
One repair per tick avoids a thundering herd.

## Reuse `~/.agents`, don't fork it
`box ls` reads `~/.agents/status/*` directly; `box new` mirrors `spawn.sh`'s
worktree+session+status convention in Go (`internal/worktree`, `session`,
`agentstatus`). On a fresh host with no toolkit, `box host init` vendors minimal
copies (`internal/agents/toolkit/*.sh`) and wires the Claude hooks — but never
overwrites the user's existing scripts.

## Managed agents declared in `managed.json`
`box new` (a separate process from the daemon) records the agent in
`~/.config/agentbox/managed.json`; the supervisor reads it each tick and keeps
declared sessions alive. It never spawns agents the user didn't ask for.

## Claude auth: surface the footguns, don't hide them
`box host init` Bootstrap explicitly warns about the `ANTHROPIC_API_KEY` shadow
(metered API instead of subscription), the macOS Keychain-over-SSH lock, and
defaults to interactive `/login` (full scope, so the optional Remote Control
transport stays available). Spawned sessions strip `ANTHROPIC_API_KEY` by
default (`unset_anthropic_api_key`).

## Anthropic Remote Control is adopted, not depended on
Treated as an optional free phone transport where available; mosh+tmux is always
the guaranteed fallback. Note: the existence/behavior of Remote Control comes
from research dated after the build's knowledge cutoff — verify before relying
on it.

## Known limitations (documented, not bugs)
- macOS lid-close on **battery** can still sleep despite `caffeinate`; a Mac host
  should stay on AC. `box doctor` notes this.
- `eternal-terminal` (et) isn't in default apt/dnf/apk repos; the ladder skips it
  and falls back to mosh/ttyd/ssh. `box host init` reports what it couldn't install.
- The headless-Mac first-run permission question (can a monitor-less Mac grant
  all prompts?) is the plan's week-1 spike and is **not** resolved in code — it's
  an operational validation to run on real hardware.
