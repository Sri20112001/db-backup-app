package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// mongoBin locates a MongoDB Database Tools binary (mongodump, mongorestore)
// or mongosh: PATH first, then the standard Windows install layouts.
func mongoBin(name string) (string, error) {
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}
	exe := name + ".exe"
	patterns := []string{
		`C:\Program Files\MongoDB\Tools\*\bin\` + exe,
		`C:\Program Files\MongoDB\Server\*\bin\` + exe,
		`C:\Program Files\mongosh\bin\` + exe,
		filepath.Join(os.Getenv("LOCALAPPDATA"), `Programs\mongosh\`+exe),
	}
	for _, pattern := range patterns {
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			return matches[len(matches)-1], nil
		}
	}
	return "", fmt.Errorf("%s not found (install MongoDB Database Tools / mongosh and ensure it is on PATH)", name)
}

var mongoURIScheme = regexp.MustCompile(`^mongodb(\+srv)?://`)

// sanitizeMongoErr strips credentials from error text (tools may echo the
// URI): mongodb://user:pass@host -> mongodb://***@host.
func sanitizeMongoErr(uri, msg string) string {
	at := strings.LastIndex(msg, "@")
	scheme := strings.Index(msg, "://")
	if at > scheme && scheme >= 0 {
		msg = msg[:scheme+3] + "***" + msg[at:]
	}
	_ = uri
	return firstLine(strings.TrimSpace(msg))
}

func validMongoURI(uri string) bool {
	return mongoURIScheme.MatchString(uri) && len(uri) <= 2048
}

// BackupMongo dumps one database with mongodump --archive --gzip into a temp
// file. Returns the archive path and its size.
func BackupMongo(ctx context.Context, uri, dbName string) (string, int64, error) {
	mongodump, err := mongoBin("mongodump")
	if err != nil {
		return "", 0, err
	}
	tmp, err := os.CreateTemp("", "vg-mongodump-*.archive.gz")
	if err != nil {
		return "", 0, fmt.Errorf("create temp archive: %w", err)
	}
	archivePath := tmp.Name()
	tmp.Close()

	c, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(c, mongodump,
		"--uri="+uri,
		"--db="+dbName,
		"--archive="+archivePath,
		"--gzip",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		os.Remove(archivePath)
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", 0, fmt.Errorf("mongodump %s: %s", dbName, sanitizeMongoErr(uri, msg))
	}
	st, err := os.Stat(archivePath)
	if err != nil {
		os.Remove(archivePath)
		return "", 0, err
	}
	return archivePath, st.Size(), nil
}

// RestoreMongo restores an archive into targetDB (same-name restore when
// sourceDB == targetDB, namespace remap otherwise). --drop makes restores
// idempotent.
func RestoreMongo(ctx context.Context, uri, sourceDB, targetDB, archivePath string) error {
	mongorestore, err := mongoBin("mongorestore")
	if err != nil {
		return err
	}
	c, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	args := []string{
		"--uri=" + uri,
		"--archive=" + archivePath,
		"--gzip",
		"--drop",
	}
	if sourceDB != "" && targetDB != "" && sourceDB != targetDB {
		args = append(args, "--nsFrom="+sourceDB+".*", "--nsTo="+targetDB+".*")
	}
	cmd := exec.CommandContext(c, mongorestore, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("mongorestore: %s", sanitizeMongoErr(uri, msg))
	}
	return nil
}

// MongoDatabasesRequest carries the connection string for one-shot discovery.
// Like the Postgres picker, it is used once and never stored.
type MongoDatabasesRequest struct {
	URI string `json:"uri"`
}

func handleMongoDatabases(w http.ResponseWriter, r *http.Request) {
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

	var req MongoDatabasesRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}
	req.URI = strings.TrimSpace(req.URI)
	if !validMongoURI(req.URI) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid MongoDB connection string (expected mongodb://...)"})
		return
	}

	out := DatabasesResponse{Databases: []SqlDatabase{}}
	dbs, err := discoverMongoDatabases(req.URI)
	if err != nil {
		out.Error = err.Error()
	} else {
		for _, name := range dbs {
			out.Databases = append(out.Databases, SqlDatabase{Name: name})
		}
		out.Source = "mongosh"
	}
	json.NewEncoder(w).Encode(out)
}

func discoverMongoDatabases(uri string) ([]string, error) {
	mongosh, err := mongoBin("mongosh")
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Print database names as one JSON array, excluding locals.
	script := `JSON.stringify(db.getMongo().getDBNames().filter(function(n){return n!=='admin'&&n!=='local'&&n!=='config'}))`
	cmd := exec.CommandContext(ctx, mongosh, uri, "--quiet", "--eval", script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s", sanitizeMongoErr(uri, msg))
	}
	var names []string
	dec := json.NewDecoder(strings.NewReader(strings.TrimSpace(stdout.String())))
	if err := dec.Decode(&names); err != nil || len(names) == 0 {
		return nil, fmt.Errorf("connected but no databases returned")
	}
	return names, nil
}
