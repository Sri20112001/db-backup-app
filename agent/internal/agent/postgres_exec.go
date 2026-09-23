package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// pgBin locates a PostgreSQL client binary: PATH first, then next to the
// psql found by discovery, then the standard Windows install layout.
func pgBin(name string) (string, error) {
	if p, err := exec.LookPath(name); err == nil {
		return p, nil
	}
	if psql, err := findPsql(); err == nil {
		candidate := filepath.Join(filepath.Dir(psql), name+".exe")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		candidate = filepath.Join(filepath.Dir(psql), name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	for _, pattern := range []string{
		`C:\Program Files\PostgreSQL\*\bin\` + name + `.exe`,
		`C:\Program Files (x86)\PostgreSQL\*\bin\` + name + `.exe`,
	} {
		if matches, _ := filepath.Glob(pattern); len(matches) > 0 {
			return matches[len(matches)-1], nil
		}
	}
	return "", fmt.Errorf("%s not found (not on PATH, no standard install). Point PATH at the PostgreSQL bin directory", name)
}

func pgEnv(base []string, pg PgConfig) []string {
	if pg.Password != "" {
		return append(base, "PGPASSWORD="+pg.Password)
	}
	return base
}

// pgEscape quotes a value for embedding in SQL text (single-quote doubling).
func pgEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// GzipFile compresses src into dst with best compression.
func GzipFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	gz, err := gzip.NewWriterLevel(out, gzip.BestCompression)
	if err != nil {
		return err
	}
	if _, err := io.Copy(gz, in); err != nil {
		gz.Close()
		return err
	}
	return gz.Close()
}

// GunzipFile decompresses src into dst.
func GunzipFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	gz, err := gzip.NewReader(in)
	if err != nil {
		return err
	}
	defer gz.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, gz)
	return err
}

// BackupPostgres dumps one database with pg_dump in plain SQL format (.sql)
// — replayable with a single psql -f, human-readable and diffable — into a
// temp file. When compressed is true the dump is additionally gzipped
// (.sql.gz). Returns the dump path and its size.
func BackupPostgres(pg PgConfig, dbName string, compressed bool) (string, int64, error) {
	pgDump, err := pgBin("pg_dump")
	if err != nil {
		return "", 0, err
	}
	tmp, err := os.CreateTemp("", "vg-pgdump-*.sql")
	if err != nil {
		return "", 0, fmt.Errorf("create temp dump file: %w", err)
	}
	dumpPath := tmp.Name()
	tmp.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, pgDump,
		"-h", pg.Host,
		"-p", strconv.Itoa(pg.Port),
		"-U", pg.User,
		"-d", dbName,
		"-f", dumpPath,
	)
	cmd.Env = pgEnv(os.Environ(), pg)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		os.Remove(dumpPath)
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", 0, fmt.Errorf("pg_dump %s: %s", dbName, firstLine(msg))
	}

	if compressed {
		gzPath := dumpPath + ".gz"
		if err := GzipFile(dumpPath, gzPath); err != nil {
			os.Remove(dumpPath)
			return "", 0, fmt.Errorf("compress dump: %w", err)
		}
		os.Remove(dumpPath)
		dumpPath = gzPath
	}

	st, err := os.Stat(dumpPath)
	if err != nil {
		os.Remove(dumpPath)
		return "", 0, err
	}
	return dumpPath, st.Size(), nil
}

// RestorePostgres replays a plain-SQL dump into targetDB (gunzipping first
// when needed), creating the database first when it does not exist.
func RestorePostgres(pg PgConfig, targetDB, dumpPath string) error {
	psql, err := pgBin("psql")
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	run := func(name string, args ...string) (string, error) {
		c, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(c, name, args...)
		cmd.Env = pgEnv(os.Environ(), pg)
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = err.Error()
			}
			return "", fmt.Errorf("%s", firstLine(msg))
		}
		return strings.TrimSpace(stdout.String()), nil
	}

	baseArgs := []string{"-h", pg.Host, "-p", strconv.Itoa(pg.Port), "-U", pg.User}
	out, err := run(psql, append(append([]string{}, baseArgs...),
		"-d", "postgres", "-t", "-A",
		"-c", "SELECT 1 FROM pg_database WHERE datname = "+pgEscape(targetDB)+";")...)
	if err != nil {
		return fmt.Errorf("check target database: %w", err)
	}
	if out != "1" {
		createdb, err := pgBin("createdb")
		if err != nil {
			return fmt.Errorf("target database %q does not exist and createdb is unavailable: %w", targetDB, err)
		}
		if _, err := run(createdb, append(append([]string{}, baseArgs...), targetDB)...); err != nil {
			return fmt.Errorf("create target database: %w", err)
		}
	}

	// Gunzip first when the stored dump is compressed.
	sqlPath := dumpPath
	if strings.HasSuffix(dumpPath, ".gz") {
		tmp, err := os.CreateTemp("", "vg-pgrestore-*.sql")
		if err != nil {
			return fmt.Errorf("create temp sql file: %w", err)
		}
		tmpPath := tmp.Name()
		tmp.Close()
		defer os.Remove(tmpPath)
		if err := GunzipFile(dumpPath, tmpPath); err != nil {
			return fmt.Errorf("gunzip dump: %w", err)
		}
		sqlPath = tmpPath
	}

	c, cancel := context.WithTimeout(ctx, 28*time.Minute)
	defer cancel()
	// ON_ERROR_STOP aborts on the first error; without it psql replays the
	// whole file and still exits 0, hiding failures.
	cmd := exec.CommandContext(c, psql,
		"-h", pg.Host, "-p", strconv.Itoa(pg.Port), "-U", pg.User,
		"-d", targetDB, "-v", "ON_ERROR_STOP=1", "-f", sqlPath,
	)
	cmd.Env = pgEnv(os.Environ(), pg)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("psql restore: %s", firstLine(strings.TrimSpace(stderr.String())))
	}
	return nil
}
