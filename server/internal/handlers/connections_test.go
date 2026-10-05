package handlers

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/backup-saas/server/internal/dbinspect"
	"github.com/backup-saas/server/internal/models"
	"github.com/gin-gonic/gin"
)

// --- connection type normalization ---

func TestNormalizeConnectionType(t *testing.T) {
	cases := []struct {
		in      string
		want    models.ConnectionType
		wantErr bool
	}{
		{"POSTGRES", models.ConnectionPostgres, false},
		{"postgres", models.ConnectionPostgres, false},
		{"MONGODB", models.ConnectionMongo, false},
		{"MSSQL", models.ConnectionMssql, false},
		{"MSSQL_SERVER", models.ConnectionMssql, false},
		{"SQL_SERVER", models.ConnectionMssql, false},
		{"MYSQL", "", true},
		{"", "", true},
	}
	for _, tt := range cases {
		got, err := normalizeConnectionType(tt.in)
		if tt.wantErr && err == nil {
			t.Errorf("normalizeConnectionType(%q): expected error", tt.in)
		}
		if !tt.wantErr && (err != nil || got != tt.want) {
			t.Errorf("normalizeConnectionType(%q) = %q, %v; want %q", tt.in, got, err, tt.want)
		}
	}
}

func TestDefaultPortFor(t *testing.T) {
	if defaultPortFor(models.ConnectionPostgres) != 5432 {
		t.Error("POSTGRES default port must be 5432")
	}
	if defaultPortFor(models.ConnectionMongo) != 27017 {
		t.Error("MONGODB default port must be 27017")
	}
	if defaultPortFor(models.ConnectionMssql) != 1433 {
		t.Error("MSSQL default port must be 1433")
	}
}

// --- connection DTO never exposes secrets ---

func TestConnectionDTONoSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	conn := models.DatabaseConnection{
		OrganizationID:    [16]byte{1},
		AgentID:           [16]byte{2},
		Name:              "PG Prod",
		Type:              models.ConnectionPostgres,
		Host:              "192.168.1.50",
		Port:              5432,
		Username:          "vaultguard_backup",
		EncryptedPassword: "ENCRYPTED-BYTES-MUST-NOT-LEAK",
		Status:            models.ConnectionConnected,
		DatabaseNames:     "postgres\ncompany_db",
	}
	dto := toConnectionDTO(conn)
	raw, _ := json.Marshal(dto)
	s := string(raw)
	for _, leak := range []string{"ENCRYPTED-BYTES-MUST-NOT-LEAK", "password"} {
		if strings.Contains(strings.ToLower(s), strings.ToLower(leak)) {
			t.Errorf("connection DTO leaks secret material %q: %s", leak, s)
		}
	}
	if dto["host"] != "192.168.1.50" || dto["port"] != 5432 {
		t.Errorf("DTO missing metadata: %s", s)
	}
	// Databases now carry details objects (names fall back from DatabaseNames
	// with unknown sentinels when no metadata was collected).
	dbs, ok := dto["databases"].([]models.DatabaseInfo)
	if !ok || len(dbs) != 2 || dbs[1].Name != "company_db" {
		t.Errorf("DTO databases wrong: %v", dto["databases"])
	}
	if dbs[1].SizeBytes != models.UnknownSize || dbs[1].TableCount != -1 {
		t.Errorf("fallback details must be unknown sentinels: %+v", dbs[1])
	}
}

// --- dbinspect.Sanitize never spans lines / stays bounded ---

func TestSanitizeConnError(t *testing.T) {
	err := dbinspect.Sanitize(mockErr2{"first line\nsecond line with postgres://admin:secret@host/db"})
	if strings.Contains(err, "secret") || strings.Contains(err, "\n") {
		t.Errorf("error not sanitized: %q", err)
	}
	long := dbinspect.Sanitize(mockErr2{strings.Repeat("x", 500)})
	if len(long) > 300 {
		t.Errorf("error not bounded: len=%d", len(long))
	}
}

type mockErr2 struct{ msg string }

func (e mockErr2) Error() string { return e.msg }

// --- restore connection resolution (same mechanism as backup) ---

func TestConnectionForRestoreFallback(t *testing.T) {
	// No connection anywhere → nil (filesystem / legacy, env fallback).
	empty := models.RestoreJob{}
	if got := connectionForRestore(empty, []byte("0123456789abcdef0123456789abcdef")); got != nil {
		t.Errorf("expected nil without connections, got %+v", got)
	}
}

// --- metadata merge: names-only saves preserve cached details ---

func TestMergeMetadataNames(t *testing.T) {
	cached := []models.DatabaseInfo{
		{Name: "company_db", SizeBytes: 12345, TableCount: 7},
		{Name: "gone_db", SizeBytes: 1, TableCount: 1},
	}
	merged := models.MergeMetadataNames(cached, []string{"company_db", "new_db"})
	if len(merged) != 2 {
		t.Fatalf("want 2, got %v", merged)
	}
	if merged[0].SizeBytes != 12345 || merged[0].TableCount != 7 {
		t.Errorf("cached details lost: %+v", merged[0])
	}
	if merged[1].Name != "new_db" || merged[1].SizeBytes != models.UnknownSize {
		t.Errorf("new db must be unknown: %+v", merged[1])
	}
	if s := models.MarshalDatabaseMetadata(nil); s != "" {
		t.Errorf("empty marshal must be blank, got %q", s)
	}
}

// --- database source classification (connection vs filesystem) ---

func TestIsDatabaseSource(t *testing.T) {
	for _, dt := range []models.BackupSourceType{models.SourcePostgres, models.SourceMongo, models.SourceSQLServer, models.SourceMssqlServer} {
		if !isDatabaseSource(dt) {
			t.Errorf("%s must be a database source", dt)
		}
	}
	for _, fs := range []models.BackupSourceType{models.SourceFilesystem, models.SourceDBF} {
		if isDatabaseSource(fs) {
			t.Errorf("%s must NOT be a database source", fs)
		}
	}
}
