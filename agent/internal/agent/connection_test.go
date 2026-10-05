package agent

import (
	"strings"
	"testing"
)

// Per-claim connection credentials must win over environment config, and the
// secret must be clearable after use. Legacy jobs (nil connection) keep the
// env behavior (backward compatible).
func TestPgForPrefersConnection(t *testing.T) {
	fallback := PgConfig{Host: "localhost", Port: 5432, User: "postgres", Password: "env-pw"}
	job := &JobConfig{Connection: &ConnectionConfig{
		Type: "POSTGRES", Host: "192.168.1.50", Port: 5433, Username: "vaultguard_backup", Password: "claim-pw",
	}}
	got := pgFor(job, fallback)
	if got.Host != "192.168.1.50" || got.Port != 5433 || got.User != "vaultguard_backup" || got.Password != "claim-pw" {
		t.Errorf("connection not preferred: %+v", got)
	}

	legacy := &JobConfig{}
	if got := pgFor(legacy, fallback); got != fallback {
		t.Errorf("legacy job must use env config: %+v", got)
	}
}

func TestMongoURIFor(t *testing.T) {
	job := &JobConfig{Connection: &ConnectionConfig{
		Type: "MONGODB", Host: "db.internal", Port: 27017, Username: "app", Password: "s3cr3t",
	}}
	uri := mongoURIFor(job, "mongodb://localhost:27017")
	if !strings.HasPrefix(uri, "mongodb://app:") || !strings.Contains(uri, "db.internal:27017") {
		t.Errorf("bad mongo URI: %s", uri)
	}
	if strings.Contains(uri, "localhost") {
		t.Errorf("fallback leaked into claim URI: %s", uri)
	}
	legacy := &JobConfig{}
	if got := mongoURIFor(legacy, "mongodb://localhost:27017"); got != "mongodb://localhost:27017" {
		t.Errorf("legacy mongo must use env URI: %s", got)
	}
}

func TestMongoNoAuthConnection(t *testing.T) {
	// No-auth server (empty user/password): the saved connection must still
	// win (host/port), producing a bare URI — never env fallback, never "user:@".
	job := &JobConfig{Connection: &ConnectionConfig{
		Type: "MONGODB", Host: "db.internal", Port: 27018,
	}}
	uri := mongoURIFor(job, "mongodb://localhost:27017")
	if strings.Contains(uri, "@") || strings.Contains(uri, "localhost") {
		t.Errorf("no-auth URI must be bare and connection-scoped: %s", uri)
	}
	if !strings.Contains(uri, "db.internal:27018") {
		t.Errorf("connection host/port lost: %s", uri)
	}
}

func TestMssqlForPrefersConnection(t *testing.T) {
	fallback := MssqlConfig{Server: "localhost", User: "sa", Password: "env-pw"}
	job := &JobConfig{Connection: &ConnectionConfig{
		Type: "MSSQL", Host: "sql.internal", Port: 1434, Username: "backup", Password: "claim-pw",
	}}
	got := mssqlFor(job, fallback)
	if got.User != "backup" || got.Password != "claim-pw" || !strings.Contains(got.Server, "sql.internal") {
		t.Errorf("connection not preferred: %+v", got)
	}
}

func TestClearConnection(t *testing.T) {
	job := &JobConfig{Connection: &ConnectionConfig{Password: "claim-pw"}}
	clearConnection(job)
	if job.Connection.Password != "" {
		t.Error("password must be cleared from memory after use")
	}
	clearConnection(&JobConfig{}) // nil-safe
}

func TestIsRevokedError(t *testing.T) {
	for _, msg := range []string{
		"server 401: agent revoked",
		"Server 401: Agent Revoked",
	} {
		if !isRevokedError(errTest(msg)) {
			t.Errorf("must detect revoked: %q", msg)
		}
	}
	for _, msg := range []string{
		"server 401: invalid agent token",
		"connection refused",
		"server 500: boom",
		"",
	} {
		var err error
		if msg != "" {
			err = errTest(msg)
		}
		if isRevokedError(err) {
			t.Errorf("must NOT treat as revoked: %q", msg)
		}
	}
	if isRevokedError(nil) {
		t.Error("nil is not revoked")
	}
	// Latching: once revoked, the runner stops instead of retrying forever.
	r := &Runner{}
	if r.noteRevoked(errTest("server 401: invalid agent token")) || r.revoked {
		t.Error("non-revocation error must not latch")
	}
	if !r.noteRevoked(errTest("server 401: agent revoked")) || !r.revoked {
		t.Error("revocation must latch")
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }
