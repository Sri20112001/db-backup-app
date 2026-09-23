package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// PgDatabasesRequest carries PostgreSQL connection parameters from the
// dashboard. The password is used once to run psql and never stored —
// discovery only needs CONNECT on the server, and job configs keep just the
// selected database name.
type PgDatabasesRequest struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

func handlePgDatabases(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Same loopback-helper policy as /browse: any local dashboard origin.
	// POST+JSON triggers a CORS preflight, so answer OPTIONS explicitly.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "use POST"})
		return
	}

	var req PgDatabasesRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}
	if req.Host == "" {
		req.Host = "localhost"
	}
	if req.Port == 0 {
		req.Port = 5432
	}
	if req.User == "" {
		req.User = "postgres"
	}
	if !validPgHost(req.Host) || req.Port < 1 || req.Port > 65535 || !validPgUser(req.User) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid host, port, or user"})
		return
	}

	out := DatabasesResponse{Databases: []SqlDatabase{}}
	dbs, source, err := discoverPgDatabases(req)
	if err != nil {
		out.Error = err.Error()
	} else {
		out.Databases = dbs
		out.Source = source
	}
	json.NewEncoder(w).Encode(out)
}

var pgHostRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
var pgUserRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

func validPgHost(h string) bool { return pgHostRe.MatchString(h) && len(h) <= 255 }
func validPgUser(u string) bool { return pgUserRe.MatchString(u) && len(u) <= 63 }

// findPsql locates the psql client: PATH first, then the standard Windows
// install layout (C:\Program Files\PostgreSQL\<ver>\bin\psql.exe).
func findPsql() (string, error) {
	if p, err := exec.LookPath("psql"); err == nil {
		return p, nil
	}
	matches, _ := filepath.Glob(`C:\Program Files\PostgreSQL\*\bin\psql.exe`)
	if len(matches) > 0 {
		return matches[len(matches)-1], nil // newest version sorts last
	}
	matches, _ = filepath.Glob(`C:\Program Files (x86)\PostgreSQL\*\bin\psql.exe`)
	if len(matches) > 0 {
		return matches[len(matches)-1], nil
	}
	return "", fmt.Errorf("psql not found (not on PATH, no standard install). Install PostgreSQL client tools or type the database name manually")
}

func discoverPgDatabases(req PgDatabasesRequest) ([]SqlDatabase, string, error) {
	psql, err := findPsql()
	if err != nil {
		return nil, "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, psql,
		"-h", req.Host,
		"-p", strconv.Itoa(req.Port),
		"-U", req.User,
		"-d", "postgres",
		"-t", "-A",
		"-c", "SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname;",
	)
	// Password via env so it never appears in argv. exec wipes it with the
	// child; nothing is logged.
	if req.Password != "" {
		cmd.Env = append(cmd.Environ(), "PGPASSWORD="+req.Password)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, "", fmt.Errorf("psql %s:%d as %s: %s", req.Host, req.Port, req.User, firstLine(msg))
	}

	var dbs []SqlDatabase
	for _, line := range strings.Split(stdout.String(), "\n") {
		if name := strings.TrimSpace(line); name != "" {
			dbs = append(dbs, SqlDatabase{Name: name})
		}
	}
	if len(dbs) == 0 {
		return nil, "", fmt.Errorf("connected but no databases returned")
	}
	return dbs, "psql " + req.Host, nil
}
