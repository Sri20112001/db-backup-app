//go:build windows

package credstore

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

// DPAPIStore wraps FileStore: the JSON blob is additionally protected with
// Windows DPAPI (CryptProtectData) so a stolen disk or an offline copy of
// ProgramData is useless without this machine's keys. Scope matters:
//
//   - ScopeCurrentUser (default): only the protecting Windows user can
//     decrypt. Use for interactive/test credential handling.
//   - ScopeLocalMachine: any user on THIS machine can decrypt. Required for
//     the Windows Service (which runs as LocalSystem, not as the installing
//     user). The installer uses this scope. Tradeoff is documented: it
//     protects against offline-disk theft, not against other local users —
//     the standard, accepted DPAPI-for-services posture.
//
// Opt-in via AGENT_CREDENTIAL_STORE=dpapi until Windows runtime validation
// completes; then DPAPI becomes the default on Windows with no caller
// changes (NewDefaultStore flips).
type DPAPIStore struct {
	file  *FileStore
	scope Scope
}

// Scope selects the DPAPI protection scope.
type Scope int

const (
	ScopeCurrentUser Scope = iota
	ScopeLocalMachine
)

// NewDPAPIStore stores DPAPI-protected credentials at
// dir/agent-credential.json using the CurrentUser scope.
func NewDPAPIStore(dir string) *DPAPIStore {
	return &DPAPIStore{file: NewFileStore(dir), scope: ScopeCurrentUser}
}

// NewDPAPIStoreWithScope stores DPAPI-protected credentials with an explicit
// scope. The installer passes ScopeLocalMachine so the SYSTEM service can
// decrypt what the admin's setup GUI protected.
func NewDPAPIStoreWithScope(dir string, scope Scope) *DPAPIStore {
	return &DPAPIStore{file: NewFileStore(dir), scope: scope}
}

func (s *DPAPIStore) Save(c Credentials) error {
	if c.AgentID == "" || c.AgentToken == "" {
		return errors.New("refusing to save incomplete credentials")
	}
	plain := []byte(`{"agent_id":` + jsonQuote(c.AgentID) + `,"agent_token":` + jsonQuote(c.AgentToken) + `}`)
	protected, err := dpapiProtect(plain, s.scope)
	if err != nil {
		return fmt.Errorf("dpapi protect: %w", err)
	}
	// The protected blob is opaque to other users; file perms stay 0600.
	if err := ensureParentDir(s.file.Path); err != nil {
		return err
	}
	return writeFile0600(s.file.Path, protected)
}

func (s *DPAPIStore) Load() (Credentials, error) {
	protected, err := readFile(s.file.Path)
	if err != nil {
		return Credentials{}, err
	}
	plain, err := dpapiUnprotect(protected)
	if err != nil {
		// Upgrade path: pre-DPAPI installs stored plaintext JSON. Adopt it
		// and immediately re-protect at rest (self-healing), so a legacy
		// plaintext file does not linger on disk after first read.
		if c, perr := parseCredentials(protected); perr == nil {
			_ = s.reprotect(c) // best effort; credential itself is returned regardless
			return c, nil
		}
		return Credentials{}, fmt.Errorf("dpapi unprotect: %w", err)
	}
	return parseCredentials(plain)
}

// reprotect rewrites already-loaded credentials through DPAPI.
func (s *DPAPIStore) reprotect(c Credentials) error {
	plain := []byte(`{"agent_id":` + jsonQuote(c.AgentID) + `,"agent_token":` + jsonQuote(c.AgentToken) + `}`)
	protected, err := dpapiProtect(plain, s.scope)
	if err != nil {
		return err
	}
	return writeFile0600(s.file.Path, protected)
}

func (s *DPAPIStore) Delete() error {
	return s.file.Delete()
}

// --- minimal DPAPI bindings (stdlib only, no new dependencies) ---

type dataBlob struct {
	cbData uint32
	pbData *byte
}

var (
	crypt32       = syscall.NewLazyDLL("crypt32.dll")
	kernel32      = syscall.NewLazyDLL("kernel32.dll")
	procProtect   = crypt32.NewProc("CryptProtectData")
	procUnprotect = crypt32.NewProc("CryptUnprotectData")
	procLocalFree = kernel32.NewProc("LocalFree")
)

// CRYPTPROTECT_UI_FORBIDDEN: never prompt — services have no desktop.
// CRYPTPROTECT_LOCAL_MACHINE: machine scope (services); without it the scope
// is the current user.
const (
	cryptProtectUIFrobidden  = 0x1
	cryptProtectLocalMachine = 0x4
)

func dpapiProtect(plain []byte, scope Scope) ([]byte, error) {
	var flags uintptr = cryptProtectUIFrobidden
	if scope == ScopeLocalMachine {
		flags |= cryptProtectLocalMachine
	}
	var in, out dataBlob
	if len(plain) > 0 {
		in.cbData = uint32(len(plain))
		in.pbData = &plain[0]
	}
	r, _, err := procProtect.Call(
		uintptr(unsafe.Pointer(&in)),
		uintptr(0), // description
		uintptr(0), // optional entropy
		uintptr(0), // reserved
		uintptr(0), // prompt struct
		flags,
		uintptr(unsafe.Pointer(&out)),
	)
	if r == 0 {
		return nil, fmt.Errorf("CryptProtectData failed: %v", err)
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	protected := make([]byte, out.cbData)
	copy(protected, unsafe.Slice(out.pbData, out.cbData))
	return protected, nil
}

func dpapiUnprotect(protected []byte) ([]byte, error) {
	var in, out dataBlob
	if len(protected) > 0 {
		in.cbData = uint32(len(protected))
		in.pbData = &protected[0]
	}
	r, _, err := procUnprotect.Call(
		uintptr(unsafe.Pointer(&in)),
		uintptr(0), // description
		uintptr(0), // optional entropy
		uintptr(0), // reserved
		uintptr(0), // prompt struct
		uintptr(cryptProtectUIFrobidden),
		uintptr(unsafe.Pointer(&out)),
	)
	if r == 0 {
		return nil, fmt.Errorf("CryptUnprotectData failed: %v", err)
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	plain := make([]byte, out.cbData)
	copy(plain, unsafe.Slice(out.pbData, out.cbData))
	return plain, nil
}
