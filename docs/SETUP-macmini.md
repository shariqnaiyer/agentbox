# Setting up agentbox on an Apple Silicon Mac mini (host)

This is the first real-hardware setup. Do it at the mini's console (monitor +
keyboard) — that sidesteps the macOS Keychain-over-SSH issue for the Claude
login, and lets you approve permission prompts directly.

## 0. Prerequisites on the mini

Install these first (skip any you already have):

- **Homebrew** — `/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`
- **Tailscale** — install the app from <https://tailscale.com/download> (or `brew install tailscale`), open it, and **sign in to your account via the GUI**. Confirm with `tailscale status` (should say Running). Signing in via the GUI means agentbox won't need an auth key.
- **Claude Code** — install it the same way you did on your main Mac. You're already on Max, so you'll just `/login` during setup.

## 1. Get the `box` binary onto the mini

The binary is built at `agentbox/dist/box_darwin_arm64` on your main Mac.
**AirDrop** it to the mini (lands in `~/Downloads`), then:

```sh
sudo mkdir -p /usr/local/bin
sudo mv ~/Downloads/box_darwin_arm64 /usr/local/bin/box
sudo chmod +x /usr/local/bin/box
# Clear the Gatekeeper quarantine AirDrop adds to the unsigned binary:
sudo xattr -d com.apple.quarantine /usr/local/bin/box 2>/dev/null || true
box version          # confirm it runs
```

(If you `scp` it instead of AirDrop, the quarantine step isn't needed — scp
doesn't set the quarantine attribute.)

## 2. Run setup

```sh
box host init
```

This will, in order:
1. `brew install` the missing transports (mosh, eternalterminal, ttyd?, tmux) — approve it.
2. Detect Tailscale is already running and skip the auth-key step.
3. Launch Claude for you to run `/login` (browser opens at the console — clean, no Keychain lock). Sign in, then exit Claude.
4. Install the LaunchAgent + keep-awake, and print a **pairing code** — copy it.

Add `--web` if you also want the browser (ttyd) transport: `box host init --web`.

## 3. Make it truly always-on (headless survival)

```sh
sudo pmset -a sleep 0 disablesleep 1 powernap 0 womp 1 autorestart 1
```

- `sleep 0 disablesleep 1` — never sleep.
- `womp 1` — wake on network.
- `autorestart 1` — power back on after a power cut.

Then **System Settings → Users & Groups → Automatically log in as → (your user)**
so the supervisor starts after a reboot without anyone logging in.

> Note: auto-login requires **FileVault off**. If you keep FileVault on (more
> disk encryption), the mini won't auto-start the agent after a reboot until you
> log in once at the console. Your call on that tradeoff.

## 4. Verify on the mini

```sh
box doctor        # should be all green now
box host status   # supervisor health + tailnet address
```

## 5. Pair from your main Mac (your interface)

On your main Mac:

```sh
sudo cp ~/Documents/dev/agentbox/dist/box_darwin_arm64 /usr/local/bin/box
sudo chmod +x /usr/local/bin/box
box pair <code-from-step-2>
box                     # connects to the mini and attaches
```

## 6. Day-to-day

```sh
box new myproject --repo ~/code/myproject   # start an agent on the mini
box ls                                      # see agents + status
box                                         # attach the most-recent agent
box myproject                               # attach a specific one
```

The agent runs on the mini in tmux; disconnect/reconnect (laptop sleep, Wi-Fi→
cellular) and `box` drops you right back into it.

## If something breaks

This is the first real run, so `box host init` may hit a rough edge (a brew
formula name, Tailscale CLI path, or the Claude login). If you enable
**System Settings → General → Sharing → Remote Login** on the mini, your main
Mac can SSH in and help debug / re-run steps. `box doctor` is the first thing to
check — it names what's wrong and the fix.
