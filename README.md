# agentbox

**Run a coding agent on any always-on Unix box. Reach it from anywhere. Never manually reconnect.**

`agentbox` turns a spare Mac, a Linux VPS, a Raspberry Pi, or an old laptop into
an always-on, agent-agnostic coding-agent host. One command sets it up — deps,
Tailscale, keep-awake, autostart, agent auth, pairing — and from then on you just
run `box` from your laptop or phone and you're attached to your agent, mid-task,
even after your laptop slept and your phone switched to cellular.

It wraps proven tools (Tailscale, mosh, eternal-terminal, tmux, ttyd) and your
existing `~/.agents` workflow instead of reinventing them. The value is the glue
and the reliability, not new protocols.

> v1 supports **macOS and Linux** (Raspberry Pi / BSD included). Native Windows
> is deferred. Claude Code is the first-class agent; Codex and Gemini are wired
> in with stubbed auth.

## The mental model: three independent layers

| Layer | What | Tool |
|-------|------|------|
| **Reachability** | dial the box from anywhere, no port-forwarding | Tailscale |
| **Transport** | survive sleep / network changes / roaming | mosh → eternal-terminal → ttyd → ssh |
| **Persistence** | the agent keeps running across disconnects | tmux |

The transport ladder is probed automatically; every rung attaches the **same**
named tmux session, so a reconnect from any device lands you exactly where you
were.

## Quick start

### On the always-on box

```sh
curl -fsSL https://raw.githubusercontent.com/shariqnaiyer/agentbox/main/installer/install.sh | sh
box host init        # deps, tailnet, autostart, agent login, pairing QR
```

`box host init` is idempotent — re-run it any time to repair.

### On your laptop / phone

Install `box` (same installer), then pair by scanning the QR or pasting the code
that `host init` printed:

```sh
box pair box1_XXXX…   # records the host
box                   # connect + attach (the money command)
```

### Day to day

```sh
box new fix-auth --repo ~/code/app   # spawn an agent in its own worktree+session
box ls                               # see all agent sessions + status
box                                  # attach the most-recent agent
box fix-auth                         # attach a specific session
box host status                      # host health, without attaching
box doctor                           # diagnose
```

## Commands

```
box                       Connect to your box and attach the agent
box <host> [session]      Connect to a specific paired host / session

box host init             One-time host setup
box host                  Run the supervisor daemon (autostart launches this)
box host status [--json]  Host health without attaching

box new <name> [--agent claude|codex|gemini] [--repo PATH] [--branch B] [--attach]
box ls [--json]           Agent sessions + status
box kill <name> [--worktree]

box connect [host] [session] [--list]
box pair                  (host) print pairing code + QR
box pair <code>           (client) record a host
box hosts [rm <name>]
box web                   Enable the browser (ttyd) transport
box doctor [--json]
```

## How it works

- **Host side (`box host`)** is a supervisor: a reconciliation loop over
  components (tmux, Tailscale, agents, transport). It owns no terminals — tmux
  does — so if it crashes, autostart restarts it and it re-reconciles with **zero
  agent interruption**. It holds a keep-awake assertion for its whole life.
- **Agents** run in tmux sessions named after the agent, in a git worktree
  (`wt-<name>`), following the `~/.agents` convention. Claude Code hooks write
  status to `~/.agents/status/<name>`, which `box ls` and `box host status` read.
- **Pairing** trusts the tailnet: both ends are on your Tailscale account, so a
  pairing code just carries the host's address and transports — no key exchange.

See [DECISIONS.md](DECISIONS.md) for the engineering choices and known limits.

## Auth footguns (Claude)

`box host init` and `box doctor` will warn you about these, but to be explicit:

- **`ANTHROPIC_API_KEY` set** → Claude uses the metered API, *not* your
  subscription. agentbox strips it from spawned sessions by default; remove it
  from your shell profile to be safe.
- **macOS Keychain over SSH** → initializing over SSH on a Mac can't read the
  login Keychain. Run once on the console, or
  `security unlock-keychain ~/Library/Keychains/login.keychain-db`.
- **Token vs login** → the optional Remote Control phone transport needs a
  full-scope interactive `/login` (which `host init` walks you through), not just
  an inference-only token.

## Build from source

```sh
make build      # -> bin/box
make test
make cross      # all four release targets
```

Requires Go 1.24+. Zero module dependencies.

## License

MIT — see [LICENSE](LICENSE).
