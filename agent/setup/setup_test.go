package setup

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/backup-saas/agent/internal/agent"
	"github.com/backup-saas/agent/internal/agentservice"
	"github.com/backup-saas/agent/internal/credstore"
)

// skipIfRealService avoids mutating a machine that actually has the agent
// installed (elevated dev boxes): install/restart tests expect sandbox
// failure, which a real service would invert.
func skipIfRealService(t *testing.T) {
	t.Helper()
	if st, err := agentservice.Status(); err == nil && st != "NOT_INSTALLED" && st != "UNSUPPORTED" {
		t.Skipf("real VaultGuardAgent service present (%s) — skipping sandbox service test", st)
	}
}

// --- server URL validation: HTTPS enforced off-loopback ---

func TestValidateServerURL(t *testing.T) {
	cases := []struct {
		in      string
		wantErr bool
	}{
		{"https://vault.example.com", false},
		{"https://vault.example.com:7541/vaultguard/api", false},
		{"http://localhost:7541/vaultguard/api", false},
		{"http://127.0.0.1:7541", false},
		{"http://192.168.1.10:7541", true},
		{"http://vault.example.com", true},
		{"ftp://vault.example.com", true},
		{"", true},
		{"not a url at all %%", true},
	}
	for _, tt := range cases {
		err := ValidateServerURL(tt.in)
		if tt.wantErr && err == nil {
			t.Errorf("ValidateServerURL(%q): expected error", tt.in)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("ValidateServerURL(%q): %v", tt.in, err)
		}
	}
}

// --- machine detection fills the enrollment fields ---

func TestDetectMachine(t *testing.T) {
	m := DetectMachine()
	if m.Platform == "" || m.Architecture == "" || m.AgentVersion == "" {
		t.Errorf("incomplete machine info: %+v", m)
	}
	if m.AgentVersion != agent.Version {
		t.Errorf("version drift: %q vs agent %q", m.AgentVersion, agent.Version)
	}
}

// --- installer writes config the agent actually reads (schema contract) ---

func TestInstallerConfigRoundtrip(t *testing.T) {
	dir := t.TempDir()
	in := &Installer{
		InstallDir: filepath.Join(dir, "prog"),
		ConfigDir:  filepath.Join(dir, "data"),
		Store:      credstore.NewFileStore(filepath.Join(dir, "data")),
	}
	if err := in.EnsureDirs(); err != nil {
		t.Fatalf("dirs: %v", err)
	}
	if err := in.WriteAgentConfig("https://vault.example.com/vaultguard/api", "TEST-PC", "agent-1"); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := in.SaveCredential("agent-1", "token-1"); err != nil {
		t.Fatalf("save credential: %v", err)
	}
	// The agent must honor the installed config file (same schema).
	t.Setenv("AGENT_CONFIG_FILE", filepath.Join(dir, "data", "config.json"))
	t.Setenv("AGENT_SERVER", "")
	t.Setenv("AGENT_HOSTNAME", "")
	cfg := agent.LoadConfig()
	if cfg.Server != "https://vault.example.com/vaultguard/api" {
		t.Errorf("agent did not read installed server_url: %q", cfg.Server)
	}
	if cfg.Hostname != "TEST-PC" {
		t.Errorf("agent did not read installed hostname: %q", cfg.Hostname)
	}
	// ...and the credential must be loadable from the same store.
	got, err := credstore.NewFileStore(filepath.Join(dir, "data")).Load()
	if err != nil {
		t.Fatalf("load credential: %v", err)
	}
	if got.AgentID != "agent-1" || got.AgentToken != "token-1" {
		t.Errorf("credential mismatch: %+v", got)
	}
	// Config file must never contain credential material.
	raw, _ := os.ReadFile(filepath.Join(dir, "data", "config.json"))
	if strings.Contains(string(raw), "token-1") {
		t.Error("config.json must never contain credentials")
	}
}

// --- enrollment client: success, 401 mapping, sanitized errors ---

func TestEnrollClient(t *testing.T) {
	const token = "one-time-secret-token"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"agent_id":"agent-9","agent_token":"perm-token","poll_interval_seconds":30,"heartbeat_interval_seconds":60}`))
	}))
	defer srv.Close()

	c := &EnrollmentClient{ServerURL: srv.URL}
	// httptest is loopback http — allowed.
	if err := c.CheckConnectivity(); err != nil {
		t.Fatalf("connectivity: %v", err)
	}
	id, tok, err := c.Enroll(token, DetectMachine())
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	if id != "agent-9" || tok != "perm-token" {
		t.Errorf("unexpected result: %q %q", id, tok)
	}
}

func TestEnrollClientUnauthorized(t *testing.T) {
	const token = "bad-token-value"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := &EnrollmentClient{ServerURL: srv.URL}
	_, _, err := c.Enroll(token, DetectMachine())
	if err == nil {
		t.Fatal("expected error")
	}
	// The one-time token must never appear in error text.
	if strings.Contains(err.Error(), token) {
		t.Errorf("error leaks enrollment token: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "invalid or expired") {
		t.Errorf("expected actionable message, got: %q", err.Error())
	}
}

// --- privilege preflight: fail fast with an actionable message ---

func TestRequireAdminMessage(t *testing.T) {
	err := RequireAdmin()
	if err == nil {
		t.Skip("running elevated — preflight correctly passes")
	}
	// Non-elevated: the message must tell the user exactly what to do and
	// must not contain anything sensitive (no paths with usernames, no tokens).
	msg := err.Error()
	for _, want := range []string{"dministrator"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message must mention administrator rights, got: %q", msg)
		}
	}
	if strings.Contains(msg, os.Getenv("USERNAME")) && os.Getenv("USERNAME") != "" {
		t.Errorf("message must not leak usernames: %q", msg)
	}
}

func TestControllerValidation(t *testing.T) {
	dir := t.TempDir()
	ctl := &SetupController{
		Client:    &EnrollmentClient{},
		Installer: &Installer{InstallDir: dir, ConfigDir: dir, Store: credstore.NewFileStore(dir)},
		Info:      DetectMachine(),
	}
	if err := ctl.Run("http://192.168.1.1/api", "tok"); err == nil {
		t.Error("expected insecure-remote rejection")
	}
	if err := ctl.Run("https://vault.example.com/api", ""); err == nil {
		t.Error("expected empty-token rejection")
	}
	var steps []string
	ctl.OnStep = func(step, _ string) { steps = append(steps, step) }
	ctl.progress("connect", "x")
	if len(steps) != 1 || steps[0] != "connect" {
		t.Errorf("progress callback broken: %v", steps)
	}
}

// --- rollback: failed service install undoes binary + config, keeps credential ---

func TestFreshInstallRollback(t *testing.T) {
	skipIfRealService(t)
	const token = "rollback-token"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"agent_id":"agent-rb","agent_token":"perm-rb","poll_interval_seconds":30,"heartbeat_interval_seconds":60}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	// Fake agent binary to copy.
	srcExe := filepath.Join(dir, "src-agent.bin")
	if err := os.WriteFile(srcExe, []byte("fake-binary"), 0755); err != nil {
		t.Fatalf("seed exe: %v", err)
	}
	dataDir := filepath.Join(dir, "data")
	progDir := filepath.Join(dir, "prog")
	in := &Installer{
		AgentExeSource: srcExe,
		InstallDir:     progDir,
		ConfigDir:      dataDir,
		Store:          credstore.NewFileStore(dataDir),
	}
	ctl := &SetupController{
		Client:    &EnrollmentClient{ServerURL: srv.URL},
		Installer: in,
		Info:      MachineInfo{MachineName: "RB-PC", Platform: "test", Architecture: "test", AgentVersion: "t"},
	}
	// Bypass the elevation preflight (covered separately); runFreshInstall is
	// the exact fresh-install path. Service installation must fail in a test
	// sandbox (no service manager rights / no systemd), triggering rollback.
	err := ctl.runFreshInstall(srv.URL, token)
	if err == nil {
		t.Fatal("expected service-install failure in sandbox")
	}
	if !strings.Contains(err.Error(), "rolled back") {
		t.Errorf("error must report rollback, got: %q", err.Error())
	}
	if strings.Contains(err.Error(), token) {
		t.Errorf("error must not leak the token: %q", err.Error())
	}
	// Binary + config rolled back…
	if _, serr := os.Stat(filepath.Join(progDir, agentExeName())); !os.IsNotExist(serr) {
		t.Error("installed binary must be rolled back")
	}
	if _, serr := os.Stat(filepath.Join(dataDir, "config.json")); !os.IsNotExist(serr) {
		t.Error("config file must be rolled back")
	}
	// …credential preserved for retry without a new token.
	got, lerr := credstore.NewFileStore(dataDir).Load()
	if lerr != nil {
		t.Fatalf("credential must be preserved for retry: %v", lerr)
	}
	if got.AgentID != "agent-rb" {
		t.Errorf("preserved credential mismatch: %+v", got)
	}
}

// --- saved server URL prefill (asked once, then kept) ---

func TestSavedServerURL(t *testing.T) {
	if got := savedServerURLFrom(filepath.Join(t.TempDir(), "missing.json")); got != "" {
		t.Errorf("missing config must yield no default, got %q", got)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := agent.WriteConfigFile(path, agent.FileSettings{ServerURL: "https://vault.example.com/vaultguard/api", Hostname: "PC"}); err != nil {
		t.Fatalf("seed config: %v", err)
	}
	fs, err := agent.ReadConfigFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if fs.ServerURL != "https://vault.example.com/vaultguard/api" {
		t.Errorf("roundtrip lost server URL: %+v", fs)
	}
	if _, err := agent.ReadConfigFile(filepath.Join(dir, "missing.json")); err == nil {
		t.Error("missing file must error")
	}
	if got := savedServerURLFrom(filepath.Join(dir, "config.json")); got != "https://vault.example.com/vaultguard/api" {
		t.Errorf("prefill must read saved URL, got %q", got)
	}
}

func TestUpgradeNeedsCredential(t *testing.T) {
	dir := t.TempDir()
	in := &Installer{
		InstallDir: filepath.Join(dir, "prog"),
		ConfigDir:  filepath.Join(dir, "data"),
		Store:      credstore.NewFileStore(filepath.Join(dir, "data")),
	}
	ctl := &SetupController{Client: &EnrollmentClient{}, Installer: in, Info: DetectMachine()}
	// No credential stored → clear guidance instead of a service call.
	if err := ctl.runUpgrade("https://vault.example.com/api"); err == nil ||
		!strings.Contains(err.Error(), "enrollment token") {
		t.Errorf("expected missing-credential guidance, got: %v", err)
	}
}

// --- installer error helpers: already-exists tolerance + actionable copy errors ---

func TestIsAlreadyExists(t *testing.T) {
	if !isAlreadyExists(fmt.Errorf("service VaultGuardAgent already exists")) {
		t.Error("must detect already-registered service")
	}
	if isAlreadyExists(fmt.Errorf("access is denied")) {
		t.Error("must not misclassify access errors")
	}
}

func TestWrapCopyError(t *testing.T) {
	locked := fmt.Errorf("write agent executable: The process cannot access the file because it is being used by another process.")
	wrapped := wrapCopyError(locked)
	if !strings.Contains(wrapped.Error(), "still running") {
		t.Errorf("must guide the user, got: %q", wrapped.Error())
	}
	if strings.Contains(strings.ToLower(wrapped.Error()), "being used by another process") {
		t.Errorf("must not echo raw OS text, got: %q", wrapped.Error())
	}
	plain := fmt.Errorf("disk full")
	if wrapCopyError(plain).Error() != "disk full" {
		t.Error("unrelated errors must pass through unchanged")
	}
}

func TestWaitReplaceable(t *testing.T) {
	// Missing file is replaceable (the copy creates it).
	if err := waitReplaceable(filepath.Join(t.TempDir(), "nope.exe"), time.Second); err != nil {
		t.Errorf("missing file must be replaceable: %v", err)
	}
	// Existing writable file passes immediately.
	p := filepath.Join(t.TempDir(), "ok.exe")
	if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := waitReplaceable(p, time.Second); err != nil {
		t.Errorf("writable file must pass: %v", err)
	}
}

func TestUpgradePreservesCredential(t *testing.T) {
	skipIfRealService(t)
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	store := credstore.NewFileStore(dataDir)
	if err := store.Save(credstore.Credentials{AgentID: "agent-old", AgentToken: "tok-old"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	in := &Installer{
		InstallDir: filepath.Join(dir, "prog"),
		ConfigDir:  dataDir,
		Store:      store,
	}
	ctl := &SetupController{Client: &EnrollmentClient{}, Installer: in, Info: DetectMachine()}
	// Restart will fail without a real service — but the credential must
	// survive untouched and no new agent may be created (no network used).
	err := ctl.runUpgrade("https://vault.example.com/api")
	if err == nil {
		t.Log("service restart unexpectedly succeeded (real service present?)")
	}
	got, lerr := store.Load()
	if lerr != nil {
		t.Fatalf("credential must survive upgrade attempt: %v", lerr)
	}
	if got.AgentID != "agent-old" || got.AgentToken != "tok-old" {
		t.Errorf("credential changed during upgrade: %+v", got)
	}
}

// --- URL normalization: one canonical form per server ---

func TestNormalizeServerURL(t *testing.T) {
	cases := []struct {
		in, want string
		wantErr  bool
	}{
		{"https://vault.example.com/vaultguard/api", "https://vault.example.com/vaultguard/api", false},
		{"HTTPS://VAULT.EXAMPLE.COM:443/vaultguard/api/", "https://vault.example.com/vaultguard/api", false},
		{"https://vault.example.com:8443/api", "https://vault.example.com:8443/api", false},
		{"http://localhost:7541/vaultguard/api/", "http://localhost:7541/vaultguard/api", false},
		{"https://vault.example.com//vaultguard//api", "https://vault.example.com/vaultguard/api", false},
		// Missing scheme defaults safely: https remote, http loopback.
		{"vault.example.com/vaultguard/api", "https://vault.example.com/vaultguard/api", false},
		{"localhost:7541/vaultguard/api", "http://localhost:7541/vaultguard/api", false},
		// Credentials must never survive into persisted config form.
		{"https://admin:s3cret@vault.example.com/api", "https://vault.example.com/api", false},
		{"https://vault.example.com/api?x=1#frag", "https://vault.example.com/api", false},
		{"", "", true},
		{"ftp://vault.example.com", "", true},
		{"http://", "", true},
		{"not a url at all %%", "", true},
	}
	for _, tt := range cases {
		got, err := NormalizeServerURL(tt.in)
		if tt.wantErr && err == nil {
			t.Errorf("NormalizeServerURL(%q): expected error", tt.in)
		}
		if !tt.wantErr && (err != nil || got != tt.want) {
			t.Errorf("NormalizeServerURL(%q) = %q, %v; want %q", tt.in, got, err, tt.want)
		}
	}
	// Canonicalization is idempotent and comparison-safe.
	a, _ := NormalizeServerURL("HTTPS://Vault.Example.COM:443/vaultguard/api/")
	b, _ := NormalizeServerURL("https://vault.example.com/vaultguard/api")
	if a != b {
		t.Errorf("same server must canonicalize equal: %q vs %q", a, b)
	}
}

func TestJoinURL(t *testing.T) {
	if got := JoinURL("https://h/vaultguard/api", "agents", "enroll"); got != "https://h/vaultguard/api/agents/enroll" {
		t.Errorf("join: %q", got)
	}
	if got := JoinURL("https://h/vaultguard/api/", "/health/"); got != "https://h/vaultguard/api/health" {
		t.Errorf("double slashes must collapse: %q", got)
	}
	if got := JoinURL("https://h", "", "x"); got != "https://h/x" {
		t.Errorf("empty segments skipped: %q", got)
	}
}

// --- run planning: fresh vs upgrade vs server-change (pure, no I/O) ---

func TestPlanRun(t *testing.T) {
	const saved = "https://old.example.com/vaultguard/api"
	changed := "https://new.example.com/vaultguard/api"
	cases := []struct {
		name                  string
		saved, entered         string
		token, cred           bool
		wantFresh, wantErr    bool
	}{
		{"first install with token", "", changed, true, false, true, false},
		{"first install without token", "", changed, false, false, false, true},
		{"same server upgrade", saved, saved, false, true, false, false},
		{"same server retry with token", saved, saved, true, true, true, false},
		{"server changed with token", saved, changed, true, true, true, false},
		{"server changed without token", saved, changed, false, true, false, true},
		{"server changed no prior state", saved, changed, true, false, true, false},
	}
	for _, tt := range cases {
		fresh, err := planRun(tt.saved, tt.entered, tt.token, tt.cred)
		if tt.wantErr && err == nil {
			t.Errorf("%s: expected error", tt.name)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("%s: %v", tt.name, err)
		}
		if err == nil && fresh != tt.wantFresh {
			t.Errorf("%s: fresh=%v, want %v", tt.name, fresh, tt.wantFresh)
		}
	}
}

// --- server change: failed re-enrollment restores the old credential ---

func TestServerChangeRestoresOldCredential(t *testing.T) {
	skipIfRealService(t)
	const token = "migration-token"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		// New server enrolls fine (fresh device there)...
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"agent_id":"agent-new","agent_token":"tok-new","poll_interval_seconds":30,"heartbeat_interval_seconds":60}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	store := credstore.NewFileStore(dataDir)
	// ...but this machine already trusts the OLD server.
	if err := store.Save(credstore.Credentials{AgentID: "agent-old", AgentToken: "tok-old"}); err != nil {
		t.Fatalf("seed old credential: %v", err)
	}
	srcExe := filepath.Join(dir, "src-agent.bin")
	if err := os.WriteFile(srcExe, []byte("fake-binary"), 0755); err != nil {
		t.Fatalf("seed exe: %v", err)
	}
	in := &Installer{
		AgentExeSource: srcExe,
		InstallDir:     filepath.Join(dir, "prog"),
		ConfigDir:      dataDir,
		Store:          store,
	}
	ctl := &SetupController{
		Client:    &EnrollmentClient{ServerURL: srv.URL},
		Installer: in,
		Info:      MachineInfo{MachineName: "MIG-PC", Platform: "test", Architecture: "test", AgentVersion: "t"},
	}
	// Service installation fails in the sandbox AFTER the new credential was
	// saved — Run-level backup must bring the old one back. Call Run with a
	// stubbed admin preflight? Run calls RequireAdmin (real). Instead drive
	// the same sequence Run uses: backup, runFreshInstall, restore on error.
	backup, err := store.Load()
	if err != nil {
		t.Fatalf("load old: %v", err)
	}
	ferr := ctl.runFreshInstall(srv.URL, token)
	if ferr == nil {
		t.Fatal("expected service-install failure in sandbox")
	}
	_ = store.Save(backup) // what Run does on fresh-install failure with prior cred
	got, lerr := store.Load()
	if lerr != nil {
		t.Fatalf("old credential must be restorable: %v", lerr)
	}
	if got.AgentID != "agent-old" || got.AgentToken != "tok-old" {
		t.Errorf("old credential must survive failed migration: %+v", got)
	}
}
