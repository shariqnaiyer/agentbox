package box

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/shariqnaiyer/agentbox/internal/agents"
	"github.com/shariqnaiyer/agentbox/internal/config"
	"github.com/shariqnaiyer/agentbox/internal/pairing"
	"github.com/shariqnaiyer/agentbox/internal/platform"
	"github.com/shariqnaiyer/agentbox/internal/reach"
	"github.com/shariqnaiyer/agentbox/internal/supervisor"
)

func cmdHost(args []string) error {
	if len(args) == 0 {
		return runDaemon()
	}
	switch args[0] {
	case "init":
		return hostInit(args[1:])
	case "status":
		return hostStatus(args[1:])
	case "run":
		return runDaemon()
	default:
		return fmt.Errorf("unknown: box host %s (try: init, status)", args[0])
	}
}

// runDaemon runs the supervisor until SIGINT/SIGTERM.
func runDaemon() error {
	plat := platform.Detect()
	reg := agents.NewRegistry()
	cfg, _ := config.Load()
	sup := supervisor.New(plat, reg, cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	fmt.Fprintf(os.Stderr, "agentbox supervisor up (pid %d)\n", os.Getpid())
	err := sup.Run(ctx)
	if err == context.Canceled {
		return nil
	}
	return err
}

// hostInit is the one-time bootstrap that turns this box into an agent host.
func hostInit(args []string) error {
	fs := flag.NewFlagSet("host init", flag.ContinueOnError)
	authKey := fs.String("authkey", "", "Tailscale auth key (else prompted)")
	hostName := fs.String("hostname", hostnameOr("agentbox"), "host name (MagicDNS + display)")
	noInstall := fs.Bool("no-install", false, "skip installing transports")
	web := fs.Bool("web", false, "enable the ttyd browser transport")
	withEt := fs.Bool("with-et", false, "also install Eternal Terminal (et) — a source build needing Xcode CLT")
	yes := fs.Bool("yes", false, "assume yes; non-interactive where possible")
	if err := fs.Parse(args); err != nil {
		return err
	}

	plat := platform.Detect()
	if err := config.EnsureDir(); err != nil {
		return err
	}
	fmt.Printf("Setting up this %s box as an agent host.\n\n", plat.OS())

	// 1. Dependencies.
	step("Checking dependencies")
	if !*noInstall {
		missing := missingDeps(*web, *withEt)
		if len(missing) > 0 {
			if *yes || confirm(fmt.Sprintf("  Install %v via %s?", missing, plat.PackageManager().Name)) {
				if err := plat.InstallPackages(missing...); err != nil {
					warn("package install: %v (continuing; transports may be limited)", err)
				}
			}
		} else {
			okmsg("core dependencies present")
		}
	}

	// 2. Toolkit + hooks (so box ls / status work on a fresh host).
	step("Installing status toolkit and Claude hooks")
	if err := agents.InstallToolkit(); err != nil {
		warn("toolkit: %v", err)
	}
	if err := agents.InstallHooks(); err != nil {
		warn("hooks: %v", err)
	} else {
		okmsg("~/.agents toolkit + status hooks ready")
	}

	// 3. Tailscale.
	step("Joining your tailnet")
	if reach.IsRunning() {
		okmsg("already on tailnet as %s (%s)", reach.DNSName(), reach.IPv4())
	} else {
		key := *authKey
		if key == "" && !*yes {
			fmt.Println("  Create an auth key at https://login.tailscale.com/admin/settings/keys")
			key = prompt("  Paste a Tailscale auth key (blank to log in interactively)", "")
		}
		if err := reach.Up(key, *hostName); err != nil {
			warn("tailscale up: %v — you can re-run `box host init` after fixing this", err)
		} else {
			okmsg("joined as %s (%s)", reach.DNSName(), reach.IPv4())
		}
	}

	// 4. Agent auth.
	step("Authenticating the default agent (Claude)")
	if err := agents.NewRegistry().Default().Bootstrap(); err != nil {
		warn("agent bootstrap: %v", err)
	}

	// 5. Autostart.
	step("Installing autostart + keep-awake")
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate self: %w", err)
	}
	if err := plat.InstallAutostart(platform.AutostartSpec{
		Exec:        []string{exe, "host"},
		Description: "agentbox host supervisor",
		KeepAlive:   true,
		RunAtLoad:   true,
	}); err != nil {
		warn("autostart: %v (run init interactively for the sudo step on Linux)", err)
	} else {
		okmsg("supervisor will start on boot and stay alive")
	}

	// 6. Persist config.
	cfg, _ := config.Load()
	cfg.HostName = *hostName
	cfg.DefaultAgent = "claude"
	cfg.WebEnabled = *web
	cfg.UnsetAnthropicAPIKey = true
	if err := config.Save(cfg); err != nil {
		return err
	}

	// 7. Pairing.
	fmt.Println()
	step("Pair a client")
	printPairing(cfg)
	fmt.Println("\nDone. From a client on the same tailnet: scan the QR or run `box pair <code>`, then `box`.")
	return nil
}

func hostStatus(args []string) error {
	fs := flag.NewFlagSet("host status", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	st, err := supervisor.ReadState()
	if err != nil {
		return fmt.Errorf("no host state (is the supervisor running? `box host`): %w", err)
	}
	if *asJSON {
		b, _ := json.MarshalIndent(st, "", "  ")
		fmt.Println(string(b))
		return nil
	}
	health := "healthy"
	if !st.Healthy() {
		health = "DEGRADED"
	}
	fmt.Printf("%s  [%s]  pid %d  up since %s\n", st.Host, health, st.PID, st.StartedAt.Format("15:04:05"))
	fmt.Printf("tailnet: %s (%s)  keep-awake: %v\n\n", st.TailscaleDNS, st.TailscaleIP, st.KeepAwake)
	for _, c := range st.Components {
		mark := "✓"
		if !c.OK {
			mark = "✗"
		}
		fmt.Printf("  %s %-10s %s\n", mark, c.Name, c.Detail)
	}
	if len(st.Agents) > 0 {
		fmt.Println("\nagents:")
		for _, a := range st.Agents {
			fmt.Printf("  %s %-16s %s\n", a.Icon(), a.Name, a.State)
		}
	}
	return nil
}

func missingDeps(web, withEt bool) []string {
	// et is intentionally NOT a default: on macOS it has no Homebrew bottle and
	// builds from source (needs Xcode CLT), which is fragile across OS versions.
	// mosh + ssh + ttyd cover the transport ladder; et is opt-in via --with-et.
	want := []string{"tmux", "mosh"}
	if web {
		want = append(want, "ttyd")
	}
	if withEt {
		want = append(want, "et")
	}
	var missing []string
	for _, b := range want {
		if !onPath(b) {
			missing = append(missing, b)
		}
	}
	return missing
}

func printPairing(cfg config.Config) {
	p := pairing.Payload{
		HostName:     cfg.HostName,
		TailscaleDNS: reach.DNSName(),
		TailscaleIP:  reach.IPv4(),
		SSHUser:      currentUsername(),
		Transports:   availableTransports(cfg),
		TtydPort:     cfg.TtydPort,
	}
	code := pairing.Encode(p)
	if qr := pairing.RenderQR(code); qr != "" {
		fmt.Println(qr)
	}
	fmt.Printf("pairing code:\n  %s\n", code)
}

func availableTransports(cfg config.Config) []string {
	var t []string
	if onPath("mosh") {
		t = append(t, "mosh")
	}
	if onPath("et") {
		t = append(t, "et")
	}
	if cfg.WebEnabled && onPath("ttyd") {
		t = append(t, "ttyd")
	}
	t = append(t, "ssh")
	return t
}
