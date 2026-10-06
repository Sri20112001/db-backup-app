// Package setup implements the VaultGuard agent setup logic WITHOUT any UI
// dependency, so it is fully buildable and testable on every platform:
//
//	SetupController → EnrollmentClient → Installer → CredentialStore
//
// The Fyne setup GUI (setup/ module) is a thin binding over SetupController:
// wizard screens collect input, the controller performs enrollment and
// installation with progress callbacks. No business logic lives in button
// callbacks.
package setup

import (
	"os"
	"runtime"

	"github.com/backup-saas/agent/internal/agent"
)

// MachineInfo is auto-detected by the setup GUI (no personal data beyond
// what enrollment needs).
type MachineInfo struct {
	MachineName  string
	Platform     string
	Architecture string
	AgentVersion string
	OS           string
}

// DetectMachine fills MachineInfo from the local host.
func DetectMachine() MachineInfo {
	host, _ := os.Hostname()
	return MachineInfo{
		MachineName:  host,
		Platform:     runtime.GOOS,
		Architecture: runtime.GOARCH,
		AgentVersion: agent.Version,
		OS:           runtime.GOOS,
	}
}
