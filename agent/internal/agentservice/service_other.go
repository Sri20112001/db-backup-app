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
//
// NOTE: this file must export the same API surface as service_windows.go
// (Linux CI vets this variant). Keep both in sync.
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

// ExePath has no meaning off Windows (there is no service entrypoint to
// resolve): the -install path reports this instead of proceeding.
func ExePath() (string, error) {
	return "", errors.New("service executable resolution is only available on Windows")
}
func IsInteractive() bool { return true }
func Run(_ func(stop <-chan struct{})) error {
	return errors.New("windows service execution is only available on Windows")
}
