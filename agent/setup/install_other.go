//go:build !windows

package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Linux service management via systemd. Requires root; errors surface to the
// caller (the setup GUI shows them sanitized). File-writing steps are shared
// and unit-testable; only service activation shells out.
func installDirPlatform() string { return "/opt/vaultguard/agent" }

func agentExeName() string { return "vaultguard-agent" }

// ConfigFilePath is the installed config.json path.
func (in *Installer) ConfigFilePath() string {
	return filepath.Join(in.ConfigDir, "config.json")
}

func systemdUnitPath() string { return "/etc/systemd/system/vaultguard-agent.service" }

// SystemdUnit renders the unit file (pure, testable).
func SystemdUnit(exePath, configPath string) string {
	return "[Unit]\nDescription=VaultGuard Agent\nAfter=network-online.target\nWants=network-online.target\n\n" +
		"[Service]\nType=simple\nExecStart=" + exePath + " -config " + configPath + "\nRestart=on-failure\nRestartSec=10\n\n" +
		"[Install]\nWantedBy=multi-user.target\n"
}

// InstallService writes the systemd unit (needs root for /etc/systemd).
func (in *Installer) InstallService() error {
	unit := SystemdUnit(in.InstalledExePath(), in.ConfigFilePath())
	if err := os.WriteFile(systemdUnitPath(), []byte(unit), 0644); err != nil {
		return fmt.Errorf("write systemd unit (need root?): %w", err)
	}
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %s: %w", string(out), err)
	}
	if out, err := exec.Command("systemctl", "enable", "vaultguard-agent").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl enable: %s: %w", string(out), err)
	}
	return nil
}

// StartService starts the systemd service.
func (in *Installer) StartService() error {
	if out, err := exec.Command("systemctl", "start", "vaultguard-agent").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl start: %s: %w", string(out), err)
	}
	return nil
}

// RestartService restarts the systemd service (upgrade path).
func (in *Installer) RestartService() error {
	if out, err := exec.Command("systemctl", "restart", "vaultguard-agent").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl restart: %s: %w", string(out), err)
	}
	return nil
}

// StopServiceQuiet stops the service, tolerating absence (uninstall path).
func (in *Installer) StopServiceQuiet() error {
	if out, err := exec.Command("systemctl", "stop", "vaultguard-agent").CombinedOutput(); err != nil {
		msg := strings.ToLower(string(out))
		if strings.Contains(msg, "not found") || strings.Contains(msg, "not loaded") || strings.Contains(msg, "could not be found") {
			return nil
		}
		return fmt.Errorf("systemctl stop: %s: %w", string(out), err)
	}
	return nil
}

// UninstallService disables and removes the systemd unit, tolerating absence.
func (in *Installer) UninstallService() error {
	out, err := exec.Command("systemctl", "disable", "vaultguard-agent").CombinedOutput()
	if err != nil {
		msg := strings.ToLower(string(out))
		if !strings.Contains(msg, "not found") && !strings.Contains(msg, "no such file") && !strings.Contains(msg, "does not exist") {
			return fmt.Errorf("systemctl disable: %s: %w", string(out), err)
		}
	}
	if err := os.Remove(systemdUnitPath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove systemd unit: %w", err)
	}
	_, _ = exec.Command("systemctl", "daemon-reload").CombinedOutput()
	return nil
}

// RequireAdmin fails fast without root (systemd + /etc writes need it).
func RequireAdmin() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("root privileges required — re-run with sudo")
	}
	return nil
}
