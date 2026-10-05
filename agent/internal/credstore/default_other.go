//go:build !windows

package credstore

// NewDefaultStore returns the 0600 file store. DPAPI is Windows-only;
// systemd-credential / keyring backends can plug in here later behind the
// same CredentialStore interface.
func NewDefaultStore(dir string) CredentialStore {
	return NewFileStore(dir)
}
