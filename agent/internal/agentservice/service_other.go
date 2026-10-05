//go:build !windows

package agentservice

import "errors"

// Service identity (shared with the Windows implementation).
const (
	ServiceName = "VaultGuardAgent"
	DisplayName = "VaultGuard Agent"
	Description = "VaultGuard secure backup agent: executes backup and restore jobs from the VaultGuard server."
)

// Windows-only service management: Linux installs use systemd (see the
// setup installer), other platforms run the agent as a plain process.
func Install(_ ...string) error {
	return errors.New("windows service management is only available on Windows")
}
func Uninstall() error { return errors.New("windows service management is only available on Windows") }
func Start() error     { return errors.New("windows service management is only available on Windows") }
func Stop() error      { return errors.New("windows service management is only available on Windows") }
func Restart() error   { return errors.New("windows service management is only available on Windows") }
func Status() (string, error) {
	return "UNSUPPORTED", errors.New("windows service management is only available on Windows")
}
func IsInteractive() bool { return true }
func Run(_ func(stop <-chan struct{})) error {
	return errors.New("windows service execution is only available on Windows")
}
