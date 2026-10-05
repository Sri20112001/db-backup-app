//go:build windows

package setup

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/backup-saas/agent/internal/agentservice"
)

func installDirPlatform() string {
	return `C:\Program Files\VaultGuard\Agent`
}

func agentExeName() string { return "VaultGuard-Agent.exe" }

// RequireAdmin fails fast when the setup process is not elevated. Service
// creation, Program Files/ProgramData writes, and service start all need
// administrator rights; the app manifest requests elevation at launch, but
// if the user cancelled UAC (or the manifest wasn't embedded) this produces
// the actionable error instead of a confusing mid-install failure. No
// partial installation exists yet at this point, so nothing needs rollback.
func RequireAdmin() error {
	// `net session` succeeds only elevated. Stdlib-only, no new deps.
	if err := exec.Command("net", "session").Run(); err != nil {
		return fmt.Errorf("administrator rights required — right-click VaultGuard-Agent-Setup.exe and choose 'Run as administrator'")
	}
	return nil
}

// InstallService creates the VaultGuardAgent Windows service pointing at the
// installed agent binary with its config file (never the setup executable).
func (in *Installer) InstallService() error {
	if err := agentservice.Install(in.InstalledExePath(), "-config", in.ConfigFilePath()); err != nil {
		return fmt.Errorf("install windows service: %w", err)
	}
	return nil
}

// StartService starts the installed Windows service.
func (in *Installer) StartService() error {
	if err := agentservice.Start(); err != nil {
		return fmt.Errorf("start windows service: %w", err)
	}
	return nil
}

// RestartService restarts the installed Windows service (upgrade path:
// binary replaced on disk, then restarted into it).
func (in *Installer) RestartService() error {
	if err := agentservice.Restart(); err != nil {
		return fmt.Errorf("restart windows service: %w", err)
	}
	return nil
}

// StopServiceQuiet stops the service, tolerating not-installed/not-running
// (uninstall path: absence is the desired end state).
func (in *Installer) StopServiceQuiet() error {
	if err := agentservice.Stop(); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not installed") {
			return nil
		}
		return err
	}
	return nil
}

// UninstallService removes the Windows service, tolerating absence.
func (in *Installer) UninstallService() error {
	if err := agentservice.Uninstall(); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not installed") {
			return nil
		}
		return err
	}
	return nil
}

// ConfigFilePath is the installed config.json path.
func (in *Installer) ConfigFilePath() string {
	return filepath.Join(in.ConfigDir, "config.json")
}

// ServiceLogPath is where the service-mode agent log file lives.
func (in *Installer) ServiceLogPath() string {
	return filepath.Join(in.ConfigDir, "agent.log")
}
