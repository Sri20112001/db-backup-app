package db

import (
	"embed"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/rs/zerolog/log"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// MigrateUp applies pending versioned SQL migrations from the embedded
// migrations/ directory. It is safe on databases previously created by
// AutoMigrate: the baseline (000001) is fully idempotent, so adopting
// databases simply record version 1 without touching data.
//
// Add future schema changes as NEW numbered files (000002_*.up.sql with a
// matching .down.sql) — never edit a baseline that has already run.
func MigrateUp(dsn string) error {
	databaseURL, err := dsnToPostgresURL(dsn)
	if err != nil {
		return fmt.Errorf("parse DATABASE_URL: %w", err)
	}

	src, err := iofs.New(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("load embedded migrations: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, databaseURL)
	if err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil || dbErr != nil {
			log.Warn().Err(srcErr).Err(dbErr).Msg("migrator close reported errors")
		}
	}()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info().Msg("database migrations: already up to date")
			return nil
		}
		return fmt.Errorf("apply migrations: %w", err)
	}
	version, dirty, err := m.Version()
	if err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}
	log.Info().Uint("version", version).Bool("dirty", dirty).Msg("database migrations applied")
	return nil
}

// dsnToPostgresURL converts the lib/pq-style key=value DSN the server
// config uses (host=… user=… password=… dbname=… port=… sslmode=…) into the
// postgres:// URL the migrate driver requires. A value that already parses
// as a URL is passed through untouched.
func dsnToPostgresURL(dsn string) (string, error) {
	if u, err := url.Parse(strings.TrimSpace(dsn)); err == nil && u.Scheme != "" {
		return dsn, nil
	}
	kv := map[string]string{}
	for _, part := range strings.Fields(dsn) {
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		kv[k] = strings.Trim(v, `'"`)
	}
	if kv["dbname"] == "" || kv["user"] == "" || kv["host"] == "" {
		return "", fmt.Errorf("need host, user and dbname (got %q)", dsn)
	}
	port := kv["port"]
	if port == "" {
		port = "5432"
	}
	sslmode := kv["sslmode"]
	if sslmode == "" {
		sslmode = "disable"
	}
	u := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(kv["user"], kv["password"]),
		Host:     kv["host"] + ":" + port,
		Path:     "/" + kv["dbname"],
		RawQuery: "sslmode=" + url.QueryEscape(sslmode),
	}
	return u.String(), nil
}
