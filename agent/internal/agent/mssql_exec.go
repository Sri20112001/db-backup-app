package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// isMssqlSource reports whether a job source type means SQL Server. Both the
// current client value (MSSQL_SERVER) and the legacy server constant
// (SQL_SERVER, stored by older jobs) are accepted.
func isMssqlSource(sourceType string) bool {
	return sourceType == "MSSQL_SERVER" || sourceType == "SQL_SERVER"
}

// mssqlQuote brackets a database/object name ([my db]], ] escaped).
func mssqlQuote(name string) string {
	return "[" + strings.ReplaceAll(name, "]", "]]") + "]"
}

// safeFileStem sanitizes a database name for use in a .bak file name.
func safeFileStem(name string) string {
	s := strings.ToLower(safeName.ReplaceAllString(strings.TrimSpace(name), "_"))
	if s == "" {
		s = "db"
	}
	return s
}

// mssqlString single-quotes a string literal for T-SQL.
func mssqlString(s string) string {
	return "N'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// sqlcmdAuth appends auth flags: Windows trusted connection when no user is
// configured, SQL login otherwise.
func sqlcmdAuth(args []string, cfg MssqlConfig) []string {
	if cfg.User == "" {
		return append(args, "-E")
	}
	args = append(args, "-U", cfg.User)
	if cfg.Password != "" {
		args = append(args, "-P", cfg.Password)
	}
	return args
}

// runSqlcmd executes sqlcmd -S server with the given query and returns
// trimmed stdout. Non-zero exit yields the server's error text.
func runSqlcmd(ctx context.Context, cfg MssqlConfig, timeout time.Duration, query string) (string, error) {
	sqlcmd, err := exec.LookPath("sqlcmd")
	if err != nil {
		return "", fmt.Errorf("sqlcmd not found (install SQL Server Command Line Utilities and ensure it is on PATH)")
	}
	c, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	args := sqlcmdAuth([]string{"-S", cfg.Server, "-h", "-1", "-W", "-Q", query}, cfg)
	cmd := exec.CommandContext(c, sqlcmd, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if out := strings.TrimSpace(stdout.String()); out != "" {
			msg = out + " " + msg
		}
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", firstLine(msg))
	}
	return strings.TrimSpace(stdout.String()), nil
}

// BackupMssql takes a native full backup (BACKUP DATABASE) to a .bak file
// and returns its path and size. The .bak is written by the SQL Server
// engine itself, so backupDir must be engine-local and writable by its
// service account (AGENT_MSSQL_BACKUP_DIR, default OS temp dir).
func BackupMssql(ctx context.Context, cfg MssqlConfig, dbName string, compressed bool) (string, int64, error) {
	dir := cfg.BackupDir
	if dir == "" {
		dir = os.TempDir()
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", 0, fmt.Errorf("create backup dir: %w", err)
	}
	// Unique name; SQL Server appends nothing, so include timestamp+pid.
	bakPath := filepath.Join(dir, fmt.Sprintf("vg_%s_%d_%d.bak",
		safeFileStem(dbName), time.Now().Unix(), os.Getpid()))

	withClause := "INIT, STATS = 10"
	if compressed {
		withClause = "COMPRESSION, " + withClause
	}
	query := fmt.Sprintf("BACKUP DATABASE %s TO DISK = %s WITH %s;",
		mssqlQuote(dbName), mssqlString(bakPath), withClause)

	if _, err := runSqlcmd(ctx, cfg, time.Hour, query); err != nil {
		os.Remove(bakPath)
		// Editions without backup compression (e.g. Express): retry plain.
		if compressed && strings.Contains(strings.ToLower(err.Error()), "compression") {
			plain := fmt.Sprintf("BACKUP DATABASE %s TO DISK = %s WITH INIT, STATS = 10;",
				mssqlQuote(dbName), mssqlString(bakPath))
			if _, err2 := runSqlcmd(ctx, cfg, time.Hour, plain); err2 != nil {
				os.Remove(bakPath)
				return "", 0, fmt.Errorf("BACKUP DATABASE: %w", err2)
			}
		} else {
			return "", 0, fmt.Errorf("BACKUP DATABASE: %w", err)
		}
	}
	st, err := os.Stat(bakPath)
	if err != nil {
		os.Remove(bakPath)
		return "", 0, fmt.Errorf("backup file missing after BACKUP DATABASE: %w", err)
	}
	return bakPath, st.Size(), nil
}

type mssqlFile struct {
	Logical  string
	Physical string
	Type     string // D = data, L = log
}

// runSqlcmdSep is runSqlcmd with a column separator (for machine parsing).
func runSqlcmdSep(ctx context.Context, cfg MssqlConfig, timeout time.Duration, sep, query string) (string, error) {
	sqlcmd, err := exec.LookPath("sqlcmd")
	if err != nil {
		return "", fmt.Errorf("sqlcmd not found (install SQL Server Command Line Utilities and ensure it is on PATH)")
	}
	c, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	args := sqlcmdAuth([]string{"-S", cfg.Server, "-h", "-1", "-W", "-s", sep, "-Q", query}, cfg)
	cmd := exec.CommandContext(c, sqlcmd, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if out := strings.TrimSpace(stdout.String()); out != "" {
			msg = out + " " + msg
		}
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", firstLine(msg))
	}
	return strings.TrimSpace(stdout.String()), nil
}

// RestoreMssql restores a .bak into targetDB (created when missing). Files
// are MOVEed under the instance default data/log directories so restoring
// under a different name never collides with the source's files.
func RestoreMssql(ctx context.Context, cfg MssqlConfig, targetDB, bakPath string) error {
	// 1. Logical files inside the backup.
	out, err := runSqlcmdSep(ctx, cfg, 5*time.Minute, "|",
		"SET NOCOUNT ON; RESTORE FILELISTONLY FROM DISK = "+mssqlString(bakPath)+";")
	if err != nil {
		return fmt.Errorf("RESTORE FILELISTONLY: %w", err)
	}
	var files []mssqlFile
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "(") {
			continue // skip "(N rows affected)"
		}
		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}
		logical := strings.TrimSpace(parts[0])
		physical := strings.TrimSpace(parts[1])
		typ := strings.TrimSpace(parts[2])
		if logical == "" || (typ != "D" && typ != "L") {
			continue
		}
		files = append(files, mssqlFile{Logical: logical, Physical: physical, Type: typ})
	}
	if len(files) == 0 {
		return fmt.Errorf("backup contains no data/log files")
	}

	// 2. Instance default directories.
	dataDir, err := runSqlcmd(ctx, cfg, time.Minute,
		"SET NOCOUNT ON; SELECT CAST(SERVERPROPERTY('InstanceDefaultDataPath') AS NVARCHAR(4000));")
	if err != nil || dataDir == "" || dataDir == "NULL" {
		return fmt.Errorf("cannot determine instance default data path: %v", err)
	}
	logDir, err := runSqlcmd(ctx, cfg, time.Minute,
		"SET NOCOUNT ON; SELECT CAST(SERVERPROPERTY('InstanceDefaultLogPath') AS NVARCHAR(4000));")
	if err != nil || logDir == "" || logDir == "NULL" {
		logDir = dataDir
	}

	// 3. Create target database when missing (model/files land via MOVE).
	exists, err := runSqlcmd(ctx, cfg, time.Minute,
		"SET NOCOUNT ON; SELECT CASE WHEN DB_ID("+mssqlString(targetDB)+") IS NULL THEN 0 ELSE 1 END;")
	if err != nil {
		return fmt.Errorf("check target database: %w", err)
	}
	if strings.TrimSpace(exists) == "0" {
		if _, err := runSqlcmd(ctx, cfg, 5*time.Minute,
			"CREATE DATABASE "+mssqlQuote(targetDB)+";"); err != nil {
			return fmt.Errorf("create target database: %w", err)
		}
	}

	// 4. Restore with MOVE for every file.
	var moves []string
	for _, f := range files {
		dir := dataDir
		if f.Type == "L" {
			dir = logDir
		}
		if !strings.HasSuffix(dir, `\`) {
			dir += `\`
		}
		base := f.Physical
		if i := strings.LastIndexAny(base, `\/`); i >= 0 {
			base = base[i+1:]
		}
		// Prefix with target name to avoid clobbering sibling databases'
		// files when several restores share the default directories.
		moves = append(moves, fmt.Sprintf("MOVE %s TO %s",
			mssqlString(f.Logical), mssqlString(dir+targetDB+"_"+base)))
	}
	query := fmt.Sprintf("RESTORE DATABASE %s FROM DISK = %s WITH REPLACE, %s, STATS = 10;",
		mssqlQuote(targetDB), mssqlString(bakPath), strings.Join(moves, ", "))
	if _, err := runSqlcmd(ctx, cfg, time.Hour, query); err != nil {
		return fmt.Errorf("RESTORE DATABASE: %w", err)
	}
	return nil
}
