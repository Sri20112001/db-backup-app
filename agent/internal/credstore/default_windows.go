//go:build windows

package credstore

import (
	"os"
	"strings"
)

// NewDefaultStore returns the platform credential store. On Windows this is
// DPAPI with LocalMachine scope: the credential file is opaque at rest, and
// the LocalSystem service can decrypt what the setup GUI protected.
// DPAPI's crypto path is runtime-validated (dpapi_windows_test.go, real
// Windows); full service end-to-end still needs the manual Windows
// checklist. AGENT_CREDENTIAL_STORE=file opts back out to the plain store.
func NewDefaultStore(dir string) CredentialStore {
	if strings.EqualFold(os.Getenv("AGENT_CREDENTIAL_STORE"), "file") {
		return NewFileStore(dir)
	}
	return NewDPAPIStoreWithScope(dir, ScopeLocalMachine)
}
