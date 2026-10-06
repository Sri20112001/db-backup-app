package setup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/backup-saas/agent/internal/agent"
	"github.com/backup-saas/agent/internal/credstore"
)

// Installer performs the machine-side installation. All paths are explicit
// fields (no globals) so every step is unit-testable in temp dirs.
// Mutable data always lands in ConfigDir (ProgramData on Windows) — never
// inside Program Files.
type Installer struct {
	// AgentExeSource is the agent binary to install (usually next to setup).
	AgentExeSource string
	// InstallDir is the program directory (Windows: C:\Program Files\VaultGuard\Agent).
	InstallDir string
	// ConfigDir holds config.json + credentials (Windows: C:\ProgramData\VaultGuard\Agent).
	ConfigDir string
	// Store persists the permanent credential (DPAPI on Windows when opted in).
	Store credstore.CredentialStore
}

// DefaultInstallDir is the platform program directory.
func DefaultInstallDir() string {
	if p := installDirPlatform(); p != "" {
		return p
	}
	return agent.DefaultConfigDir()
}

// EnsureDirs creates install + config directories.
func (in *Installer) EnsureDirs() error {
	for _, d := range []string{in.InstallDir, in.ConfigDir} {
		if d == "" {
			return fmt.Errorf("installer directory not set")
		}
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", d, err)
		}
	}
	return nil
}

// InstalledExePath is the agent binary destination.
func (in *Installer) InstalledExePath() string {
	return filepath.Join(in.InstallDir, agentExeName())
}

// CopyExecutable installs the agent binary next to the service entrypoint.
func (in *Installer) CopyExecutable() error {
	if in.AgentExeSource == "" {
		return fmt.Errorf("agent executable source not set")
	}
	src, err := os.Open(in.AgentExeSource)
	if err != nil {
		return fmt.Errorf("open agent executable: %w", err)
	}
	defer src.Close()
	dst, err := os.OpenFile(in.InstalledExePath(), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("write agent executable: %w", err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy agent executable: %w", err)
	}
	return nil
}

// WriteAgentConfig writes non-secret settings (server URL, hostname,
// agent ID). The URL is canonicalized first so config.json always holds one
// comparable form. Credentials are never written here — see SaveCredential.
func (in *Installer) WriteAgentConfig(serverURL, hostname, agentID string) error {
	canonical, err := NormalizeServerURL(serverURL)
	if err != nil {
		return err
	}
	if err := ValidateServerURL(canonical); err != nil {
		return err
	}
	return agent.WriteConfigFile(filepath.Join(in.ConfigDir, "config.json"), agent.FileSettings{
		ServerURL:     canonical,
		Hostname:      hostname,
		AgentID:       agentID,
		ConfigVersion: agent.CurrentConfigVersion,
	})
}

// SaveCredential stores the permanent credential via the CredentialStore.
func (in *Installer) SaveCredential(agentID, agentToken string) error {
	if in.Store == nil {
		return fmt.Errorf("credential store not configured")
	}
	return in.Store.Save(credstore.Credentials{AgentID: agentID, AgentToken: agentToken})
}

// HasCredential reports whether a permanent credential already exists
// (upgrade path: preserve Agent ID, no new enrollment needed).
func (in *Installer) HasCredential() bool {
	if in.Store == nil {
		return false
	}
	creds, err := in.Store.Load()
	return err == nil && creds.AgentID != "" && creds.AgentToken != ""
}

// Uninstall performs a clean removal with best-effort semantics:
//
//	stop service → remove service → remove agent binary → remove credential
//
// config.json and logs are PRESERVED deliberately (diagnostics + easy
// reinstall). The server-side Agent record is never touched: without
// revocation the dashboard shows OFFLINE; with revocation it stays REVOKED.
// Each step reports; a failing step aborts with what was (and wasn't) done.
func (in *Installer) Uninstall() error {
	var done []string
	fail := func(step string, err error) error {
		return fmt.Errorf("uninstall failed at %s: %w (completed: %s)", step, err, strings.Join(done, ", "))
	}
	if err := in.StopServiceQuiet(); err != nil {
		return fail("stop service", err)
	}
	done = append(done, "service stopped")
	if err := in.UninstallService(); err != nil {
		return fail("remove service", err)
	}
	done = append(done, "service removed")
	if exe := in.InstalledExePath(); exe != "" {
		if err := os.Remove(exe); err != nil && !os.IsNotExist(err) {
			return fail("remove agent binary", err)
		}
		done = append(done, "binary removed")
	}
	if in.Store != nil {
		if err := in.Store.Delete(); err != nil {
			return fail("remove local credential", err)
		}
		done = append(done, "credential removed")
	}
	return nil
}
