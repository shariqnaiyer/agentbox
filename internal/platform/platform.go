// Package platform is the only place in agentbox with OS-specific code.
// It isolates the three things that genuinely differ between a Mac host and
// a Linux host: keeping the box awake, autostarting the supervisor on boot,
// and installing packages / elevating privilege. Everything else in agentbox
// is identical Go that calls this interface.
package platform

// Platform abstracts the host OS. Detect() returns the right implementation
// at compile time via build tags (darwin.go / linux.go).
type Platform interface {
	// OS returns "darwin" or "linux".
	OS() string

	// PreventSleep asserts that the machine must stay awake. The returned
	// handle holds the assertion until Release() is called. The supervisor
	// holds one for its entire lifetime — this is what makes "close the
	// laptop, the agent keeps running" work.
	PreventSleep(reason string) (KeepAwake, error)

	// DaemonLabel is the OS-native identifier for the supervisor service
	// ("ai.agentbox.host" on macOS, "agentbox-host" on Linux).
	DaemonLabel() string

	// InstallAutostart installs (idempotently) a boot/login service that runs
	// the supervisor and restarts it on crash.
	InstallAutostart(spec AutostartSpec) error
	RemoveAutostart() error
	AutostartStatus() (AutostartState, error)

	// Elevate runs a command with root privilege when required (sudo on
	// Linux). Stdio is inherited so an interactive password prompt works.
	Elevate(args []string, reason string) error

	// PackageManager reports the detected package manager.
	PackageManager() PkgMgr
	// InstallPackages installs logical packages ("mosh", "et", "ttyd"),
	// mapping each to the manager's real package name and skipping any the
	// manager cannot provide (reported, not fatal).
	InstallPackages(logical ...string) error
}

// KeepAwake is a held keep-awake assertion.
type KeepAwake interface {
	Release() error
}

// AutostartSpec describes the supervisor service to install.
type AutostartSpec struct {
	Exec        []string // absolute path to the box binary + its args, e.g. {"/usr/local/bin/box","host"}
	Description string
	KeepAlive   bool // restart on crash
	RunAtLoad   bool // start immediately and on every boot/login
}

// AutostartState reports whether the service is installed and running.
type AutostartState struct {
	Installed bool
	Running   bool
	PID       int
}

// PkgMgr identifies the host package manager.
type PkgMgr struct {
	Name      string // brew | apt | dnf | apk | pacman | ""
	Available bool
}

// Detect returns the Platform implementation for the current OS.
func Detect() Platform { return newPlatform() }
