//go:build windows

// Package agentservice runs the VaultGuard agent as a Windows Service
// (VaultGuardAgent) using golang.org/x/sys/windows/svc — the established
// Go service implementation, already in the module graph. The service:
//
//   - starts automatically with Windows (StartAutomatic)
//   - runs headless with no user logged in
//   - stops cleanly on SCM Stop/Shutdown
//   - logs to the agent log file (no secrets; see cmd/agent)
//
// Service control (install/uninstall/start/stop) goes through the same
// package via the agent binary's -install/-uninstall/-start/-stop flags,
// which is also what the Fyne setup GUI invokes.
package agentservice

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// ServiceName is the SCM name; DisplayName is shown in services.msc.
const (
	ServiceName  = "VaultGuardAgent"
	DisplayName  = "VaultGuard Agent"
	Description  = "VaultGuard secure backup agent: executes backup and restore jobs from the VaultGuard server."
	startTimeout = 30 * time.Second
)

// ExePath resolves the current process executable (symlinks resolved).
// Used ONLY for self-install flows (agent -install registers itself);
// the setup installer must pass the INSTALLED agent path explicitly.
func ExePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	// Resolve symlinks so SCM always points at the real binary.
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return exe, nil
}


// Install creates the service with automatic start. exePath MUST be the
// installed agent binary — never os.Executable() of the caller (the setup
// GUI/console would otherwise register ITSELF as the service, and SCM would
// time out waiting for a status report that never comes). extraArgs (e.g.
// -config <path>) are baked into the service command line.
func Install(exePath string, extraArgs ...string) error {
	if exePath == "" {
		return fmt.Errorf("service executable path is required")
	}
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer m.Disconnect()
	s, err := m.OpenService(ServiceName)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", ServiceName)
	}
	cfg := mgr.Config{
		ServiceType:  0x10, // SERVICE_WIN32_OWN_PROCESS
		StartType:    mgr.StartAutomatic,
		ErrorControl: mgr.ErrorNormal,
		DisplayName:  DisplayName,
		Description:  Description,
	}
	args := append([]string{exePath}, extraArgs...)
	s, err = m.CreateService(ServiceName, exePath, cfg, args...)
	if err != nil {
		return fmt.Errorf("create service: %w", err)
	}
	defer s.Close()
	return nil
}

// Uninstall stops (best effort) and removes the service.
func Uninstall() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer m.Disconnect()
	s, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", ServiceName)
	}
	defer s.Close()
	_ = stopService(s) // best effort: deleting a running service is fine too
	if err := s.Delete(); err != nil {
		return fmt.Errorf("delete service: %w", err)
	}
	return nil
}

// Start starts the installed service.
func Start() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer m.Disconnect()
	s, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", ServiceName)
	}
	defer s.Close()
	return s.Start()
}

// Stop sends the Stop control and waits for the service to exit.
func Stop() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer m.Disconnect()
	s, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", ServiceName)
	}
	defer s.Close()
	return stopService(s)
}

// Restart stops (tolerated when already stopped) and starts the service.
// Used by upgrades: binary replaced on disk, then restarted into it.
func Restart() error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer m.Disconnect()
	s, err := m.OpenService(ServiceName)
	if err != nil {
		return fmt.Errorf("service %s is not installed", ServiceName)
	}
	defer s.Close()
	if !isServiceStopped(s) {
		if err := stopService(s); err != nil {
			return err
		}
	}
	return s.Start()
}

// Status describes the installed service state (Running/Stopped/…, plus
// whether the service exists at all).
func Status() (string, error) {
	m, err := mgr.Connect()
	if err != nil {
		return "", fmt.Errorf("connect to service manager: %w", err)
	}
	defer m.Disconnect()
	s, err := m.OpenService(ServiceName)
	if err != nil {
		return "NOT_INSTALLED", nil
	}
	defer s.Close()
	st, err := s.Query()
	if err != nil {
		return "", fmt.Errorf("query status: %w", err)
	}
	return svcStateString(st.State), nil
}

func svcStateString(s svc.State) string {
	switch s {
	case svc.Stopped:
		return "STOPPED"
	case svc.StartPending:
		return "START_PENDING"
	case svc.StopPending:
		return "STOP_PENDING"
	case svc.Running:
		return "RUNNING"
	case svc.ContinuePending:
		return "CONTINUE_PENDING"
	case svc.PausePending:
		return "PAUSE_PENDING"
	case svc.Paused:
		return "PAUSED"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", uint32(s))
	}
}

func isServiceStopped(s *mgr.Service) bool {
	st, err := s.Query()
	return err == nil && st.State == svc.Stopped
}

func stopService(s *mgr.Service) error {
	status, err := s.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("stop control: %w", err)
	}
	deadline := time.Now().Add(startTimeout)
	for status.State != svc.Stopped {
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for service to stop")
		}
		time.Sleep(300 * time.Millisecond)
		status, err = s.Query()
		if err != nil {
			return fmt.Errorf("query status: %w", err)
		}
	}
	return nil
}

// IsInteractive reports whether the process runs with a console (as opposed
// to under the Service Control Manager).
func IsInteractive() bool {
	is, err := svc.IsAnInteractiveSession()
	if err != nil {
		return true
	}
	return is
}

type serviceHandler struct {
	runFn func(stop <-chan struct{})
}

func (h *serviceHandler) Execute(_ []string, req <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	changes <- svc.Status{State: svc.StartPending}
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		h.runFn(stop)
	}()
	changes <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	for c := range req {
		switch c.Cmd {
		case svc.Interrogate:
			changes <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			close(stop)
			changes <- svc.Status{State: svc.StopPending}
			select {
			case <-done:
			case <-time.After(startTimeout):
			}
			return false, 0
		}
	}
	return false, 0
}

// Run executes runFn under the Service Control Manager. It blocks until SCM
// stops the service; runFn must return when stop is closed.
func Run(runFn func(stop <-chan struct{})) error {
	return svc.Run(ServiceName, &serviceHandler{runFn: runFn})
}
