// Package credstore abstracts the agent's permanent credential storage.
//
// The agent's (agent_id, agent_token) pair is the keys-to-the-kingdom: it
// authenticates backup/restore claims AND decrypts per-claim database
// credentials. It must never live in plaintext config.json, env, logs, or
// the installer. This package is the single choke point:
//
//	CredentialStore: Save / Load / Delete.
//
// Backends:
//   - FileStore (all platforms, default): JSON file with 0600 permissions
//     inside the agent config dir. Same format as the legacy
//     agent-state.json, which is adopted on first run.
//   - DPA PIStore (Windows, opt-in via AGENT_CREDENTIAL_STORE=dpapi):
//     the same JSON blob additionally wrapped with Windows DPAPI
//     (CryptProtectData, CurrentUser scope). DPAPI becomes the default
//     once it has Windows runtime validation; the abstraction means no
//     caller changes when that flips.
package credstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Credentials is the permanent agent credential issued at enrollment.
type Credentials struct {
	AgentID    string `json:"agent_id"`
	AgentToken string `json:"agent_token"`
}

// CredentialStore persists Credentials. Implementations must never log or
// expose the token.
type CredentialStore interface {
	Save(c Credentials) error
	Load() (Credentials, error)
	Delete() error
}

var errIncomplete = errors.New("incomplete credential file")

const credentialFileName = "agent-credential.json"

// FileStore keeps credentials in a 0600 JSON file under dir.
type FileStore struct {
	Path string
}

// NewFileStore stores credentials at dir/agent-credential.json.
func NewFileStore(dir string) *FileStore {
	return &FileStore{Path: filepath.Join(dir, credentialFileName)}
}

func (s *FileStore) Save(c Credentials) error {
	if c.AgentID == "" || c.AgentToken == "" {
		return errors.New("refusing to save incomplete credentials")
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0700); err != nil {
		return fmt.Errorf("create credential dir: %w", err)
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	// 0600: owner-only. The payload is written via FileStore only — callers
	// must not copy the token elsewhere.
	return os.WriteFile(s.Path, data, 0600)
}

func (s *FileStore) Load() (Credentials, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		return Credentials{}, err
	}
	var c Credentials
	if err := json.Unmarshal(data, &c); err != nil {
		return Credentials{}, err
	}
	if c.AgentID == "" || c.AgentToken == "" {
		return Credentials{}, errIncomplete
	}
	return c, nil
}

func (s *FileStore) Delete() error {
	err := os.Remove(s.Path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
