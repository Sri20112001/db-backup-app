package setup

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/backup-saas/agent/internal/credstore"
)

// Progress reports setup milestones to the UI (Fyne screens, CLI, tests).
// step is one of: connect, register, install, done. Secrets never appear in
// detail strings — the controller never forwards tokens or credentials.
type Progress func(step, detail string)

// SetupResult summarizes a successful setup for the completion screen.
type SetupResult struct {
	AgentID     string
	MachineName string
}

// SetupController orchestrates enrollment + installation. The Fyne wizard
// collects input and calls Run; all logic lives here, fully testable
// without a GUI.
type SetupController struct {
	Client    *EnrollmentClient
	Installer *Installer
	Info      MachineInfo
	OnStep    Progress
	// Result is populated on successful Run (completion screen data).
	Result *SetupResult
}

func (s *SetupController) progress(step, detail string) {
	if s.OnStep != nil {
		s.OnStep(step, detail)
	}
}

// Run executes fresh install (token given) or upgrade (no token, existing
// credential preserved). Fresh install rolls back partial work on failure
// (service → binary → config, in reverse); the saved credential is
// deliberately PRESERVED so a retry needs no new token (upgrade path picks
// it up). Upgrade never generates a new Agent: ID and credential survive.
//
// Server identity is compared canonically. A changed URL forces the token
// path: the existing credential belongs to the old server and is never sent
// to the new one; on failed re-enrollment the previous credential is
// restored so the old setup keeps working.
func (s *SetupController) Run(serverURL, token string) error {
	if s.Client == nil || s.Installer == nil {
		return fmt.Errorf("setup controller not configured")
	}
	canonical, err := NormalizeServerURL(serverURL)
	if err != nil {
		return err
	}
	if err := ValidateServerURL(canonical); err != nil {
		return err
	}
	// Privilege preflight before touching anything: no partial installs.
	if err := RequireAdmin(); err != nil {
		return err
	}
	s.Client.ServerURL = canonical

	saved := SavedServerURL()
	hasToken := strings.TrimSpace(token) != ""
	fresh, err := planRun(saved, canonical, hasToken, s.Installer.HasCredential())
	if err != nil {
		return err
	}
	if !fresh {
		return s.runUpgrade(canonical)
	}
	// Fresh path with a prior credential (server change or token retry):
	// a failed run must restore — never strand — the old setup.
	if s.Installer.HasCredential() {
		return s.withCredentialBackup(func() error {
			return s.runFreshInstall(canonical, strings.TrimSpace(token))
		})
	}
	return s.runFreshInstall(canonical, strings.TrimSpace(token))
}

// withCredentialBackup runs fn; on failure it restores the pre-existing
// credential (best effort) so a failed re-enrollment never destroys the
// working setup it was meant to replace.
func (s *SetupController) withCredentialBackup(fn func() error) error {
	var backup *credstore.Credentials
	if c, err := s.Installer.Store.Load(); err == nil {
		backup = &c
	} else {
		return fn()
	}
	if err := fn(); err != nil {
		_ = s.Installer.Store.Save(*backup) // best effort
		return fmt.Errorf("%v Previous credential restored — the agent still points at the old server", err)
	}
	return nil
}

// planRun decides fresh-install vs upgrade without side effects (unit
// tested). savedCanonical is "" on first install.
func planRun(savedCanonical, enteredCanonical string, hasToken, hasCredential bool) (fresh bool, err error) {
	if savedCanonical != "" && savedCanonical != enteredCanonical {
		if !hasToken {
			return false, fmt.Errorf("the server URL has changed — this agent must be re-enrolled with the new server, so an enrollment token is required (the existing credential belongs to the previous server and was left untouched)")
		}
		return true, nil
	}
	if hasToken {
		return true, nil
	}
	if hasCredential {
		return false, nil
	}
	return false, fmt.Errorf("no existing agent credential found — paste an enrollment token for first-time setup")
}

// runUpgrade refreshes the binary + config and restarts the service,
// preserving Agent ID and credential. Fails clearly when there is nothing
// to upgrade (fresh machines need an enrollment token).
func (s *SetupController) runUpgrade(serverURL string) error {
	if !s.Installer.HasCredential() {
		return fmt.Errorf("no existing agent credential found — paste an enrollment token for first-time setup")
	}
	s.progress("install", "Existing credential found — upgrading in place (Agent ID preserved)…")
	if err := s.Installer.EnsureDirs(); err != nil {
		return err
	}
	if s.Installer.AgentExeSource != "" {
		// Stop before replacing: a running service locks its binary.
		// Verified (not just requested) — see stopAndWait.
		if err := s.stopAndWait(); err != nil {
			return err
		}
		s.progress("install", "Updating agent executable…")
		if err := s.Installer.CopyExecutable(); err != nil {
			return wrapCopyError(err)
		}
	}
	s.progress("install", "Refreshing configuration…")
	host := s.Info.MachineName
	if host == "" {
		host = "unknown-host"
	}
	// Preserve the stored agent ID in config (diagnostics, non-secret).
	agentID := ""
	if creds, err := s.Installer.Store.Load(); err == nil {
		agentID = creds.AgentID
	}
	if err := s.Installer.WriteAgentConfig(serverURL, host, agentID); err != nil {
		return err
	}
	s.progress("install", "Restarting service…")
	if err := s.Installer.RestartService(); err != nil {
		if !isNotInstalled(err) {
			return err
		}
		// Rolled-back or never-installed machine: install fresh instead of
		// failing the retry (credential already preserved, no token needed).
		s.progress("install", "Service not present — installing fresh…")
		if err := s.Installer.InstallService(); err != nil {
			return err
		}
		if err := s.Installer.StartService(); err != nil {
			return err
		}
	}
	s.Result = &SetupResult{AgentID: agentID, MachineName: host}
	s.progress("done", "VaultGuard Agent upgraded and running.")
	return nil
}

// runFreshInstall enrolls with the one-time token, then installs. Rollback
// undoes, in reverse: service (if installed by this run) → binary (if
// copied by this run) → config (if written by this run). Pre-existing files
// are never deleted. The credential is preserved on purpose (see Run).
func (s *SetupController) runFreshInstall(serverURL, token string) error {
	s.progress("connect", "Connecting to VaultGuard…")
	if err := s.Client.CheckConnectivity(); err != nil {
		return err
	}
	s.progress("connect", "Connected.")

	s.progress("register", "Validating enrollment token…")
	agentID, agentToken, err := s.Client.Enroll(token, s.Info)
	if err != nil {
		return err
	}
	s.progress("register", "This computer is registered.")

	type undo struct {
		name string
		fn   func() error
	}
	var rollbacks []undo
	rollback := func(cause error) error {
		var undone []string
		for i := len(rollbacks) - 1; i >= 0; i-- {
			if err := rollbacks[i].fn(); err == nil {
				undone = append(undone, rollbacks[i].name)
			}
		}
		msg := "installation failed: " + sanitizeError(cause.Error())
		if len(undone) > 0 {
			msg += " (rolled back: " + strings.Join(undone, ", ") + "; credential preserved — re-run setup to retry without a new token)"
		}
		return fmt.Errorf("%s", msg)
	}
	fileExisted := func(path string) bool {
		_, err := statFile(path)
		return err == nil
	}

	s.progress("install", "Creating directories…")
	if err := s.Installer.EnsureDirs(); err != nil {
		return err
	}
	if s.Installer.AgentExeSource != "" {
		// A previous install's service may still be running and locking the
		// executable (Windows cannot overwrite a running binary). Stop it
		// and VERIFY the lock is gone — SCM acks stop before the process
		// fully exits, and AV/indexer holds transient locks.
		s.progress("install", "Stopping any previous service…")
		if err := s.stopAndWait(); err != nil {
			return rollback(err)
		}
		s.progress("install", "Installing agent executable…")
		exeExisted := fileExisted(s.Installer.InstalledExePath())
		if err := s.Installer.CopyExecutable(); err != nil {
			return rollback(wrapCopyError(err))
		}
		if !exeExisted {
			exe := s.Installer.InstalledExePath()
			rollbacks = append(rollbacks, undo{"agent binary", func() error { return removeFile(exe) }})
		}
	}
	s.progress("install", "Writing configuration…")
	host := s.Info.MachineName
	if host == "" {
		host = "unknown-host"
	}
	cfgPath := s.Installer.ConfigFilePath()
	cfgExisted := fileExisted(cfgPath)
	if err := s.Installer.WriteAgentConfig(serverURL, host, agentID); err != nil {
		return rollback(err)
	}
	if !cfgExisted {
		rollbacks = append(rollbacks, undo{"config file", func() error { return removeFile(cfgPath) }})
	}
	s.progress("install", "Storing agent credential securely…")
	if err := s.Installer.SaveCredential(agentID, agentToken); err != nil {
		return rollback(err)
	}
	// Credential intentionally has no rollback entry (see Run).
	s.progress("install", "Installing service…")
	if err := s.Installer.InstallService(); err != nil {
		if !isAlreadyExists(err) {
			return rollback(err)
		}
		// Retry over a previous install: the service is already registered
		// (its binary was just replaced above) — proceed to starting it.
		s.progress("install", "Service already registered — continuing…")
	}
	rollbacks = append(rollbacks, undo{"windows service", func() error { return s.Installer.UninstallService() }})
	s.progress("install", "Starting service…")
	if err := s.Installer.StartService(); err != nil {
		return rollback(err)
	}
	machine := s.Info.MachineName
	if machine == "" {
		machine = host
	}
	s.Result = &SetupResult{AgentID: agentID, MachineName: machine}
	s.progress("done", "VaultGuard Agent installed and running.")
	return nil
}

// NewInstallerForMachine builds the platform-default installer wiring the
// default credential store. agentExeSource may be "" to skip binary copy
// (e.g. tests or pre-staged installs).
func NewInstallerForMachine(agentExeSource string) *Installer {
	configDir := defaultSetupConfigDir()
	return &Installer{
		AgentExeSource: agentExeSource,
		InstallDir:     DefaultInstallDir(),
		ConfigDir:      configDir,
		Store:          credstore.NewDefaultStore(configDir),
	}
}

func statFile(path string) (os.FileInfo, error) { return os.Stat(path) }

func removeFile(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// stopAndWait stops any previous service and waits until the installed
// binary is actually replaceable. Best-effort stop (absence is fine), then
// a bounded wait: SCM acknowledges stop before the process fully exits, and
// antivirus/indexer locks are transient. Returns an actionable error naming
// the blocker when the lock persists (e.g. a manually-started agent the
// service stop cannot reach).
func (s *SetupController) stopAndWait() error {
	_ = s.Installer.StopServiceQuiet()
	return waitReplaceable(s.Installer.InstalledExePath(), 30*time.Second)
}

// waitReplaceable polls until path can be opened for writing. A missing file
// is replaceable (the copy will create it).
func waitReplaceable(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		f, err := os.OpenFile(path, os.O_RDWR, 0)
		if err == nil {
			f.Close()
			return nil
		}
		if os.IsNotExist(err) {
			return nil
		}
		if time.Now().After(deadline) {
			return err
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// isNotInstalled detects the no-service case across platforms so upgrade can
// fall through to fresh install instead of failing the retry.
func isNotInstalled(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not installed") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "does not exist")
}

// isAlreadyExists detects an already-registered service so a retry over a
// previous install proceeds to starting it instead of failing.
func isAlreadyExists(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "already exists")
}

// wrapCopyError turns OS file-lock failures into actionable guidance. A
// running agent locks its own binary on Windows; the pre-copy stop usually
// prevents this, but a manually-started copy needs explicit action.
func wrapCopyError(err error) error {
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "being used by another process") ||
		strings.Contains(msg, "access is denied") {
		return fmt.Errorf("the installed agent is still running and holding its binary. Stop it first — services.msc → VaultGuardAgent → Stop (or Task Manager → End task on VaultGuard-Agent.exe) — then retry")
	}
	return err
}
