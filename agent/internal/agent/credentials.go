package agent

import (
	"fmt"
	"log"
	"runtime"

	"github.com/backup-saas/agent/internal/credstore"
)

// ResolveCredentials loads the permanent agent credential or obtains one.
// Order: CredentialStore → legacy state file (adopted into the store) →
// one-time enrollment token (preferred for setup installs) → legacy
// registration key. The plaintext token only ever exists in memory and in
// the store — never in logs, config files, or error strings.
func ResolveCredentials(cfg Config, store credstore.CredentialStore) (*state, error) {
	if creds, err := store.Load(); err == nil {
		return &state{AgentID: creds.AgentID, AgentToken: creds.AgentToken}, nil
	}
	if s, err := loadState(cfg.StateFile); err == nil {
		// Adopt the legacy cache into the store (best effort; the file
		// itself stays as-is for rollback safety).
		_ = store.Save(credstore.Credentials{AgentID: s.AgentID, AgentToken: s.AgentToken})
		return s, nil
	}
	c := NewClientWithTLS(cfg.Server, "", cfg.TLSSkipVerify)
	if cfg.EnrollmentToken != "" {
		out, err := c.Enroll(EnrollRequest{
			EnrollmentToken: cfg.EnrollmentToken,
			MachineName:     cfg.Hostname,
			Platform:        runtime.GOOS,
			Architecture:    runtime.GOARCH,
			AgentVersion:    Version,
			Hostname:        cfg.Hostname,
			OS:              runtime.GOOS,
			IPAddress:       outboundIP(),
		})
		if err != nil {
			return nil, fmt.Errorf("enrollment failed: %w", err)
		}
		s := &state{AgentID: out.AgentID, AgentToken: out.AgentToken}
		if err := store.Save(credstore.Credentials{AgentID: s.AgentID, AgentToken: s.AgentToken}); err != nil {
			return nil, fmt.Errorf("save credential: %w", err)
		}
		log.Printf("enrolled as agent %s (credential stored securely)", out.AgentID)
		return s, nil
	}
	if cfg.RegistrationKey != "" {
		out, err := c.Register(cfg.RegistrationKey, cfg.Hostname, Version)
		if err != nil {
			return nil, fmt.Errorf("registration failed: %w", err)
		}
		s := &state{AgentID: out.AgentID, AgentToken: out.AgentToken}
		if err := saveState(cfg.StateFile, s); err != nil {
			return nil, fmt.Errorf("save state: %w", err)
		}
		log.Printf("registered as agent %s (credentials saved to %s)", out.AgentID, cfg.StateFile)
		return s, nil
	}
	return nil, fmt.Errorf("no credential stored and neither AGENT_ENROLLMENT_TOKEN nor AGENT_REGISTRATION_KEY is set; enroll this machine from the VaultGuard dashboard first")
}
