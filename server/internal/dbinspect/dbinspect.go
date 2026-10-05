// Package dbinspect collects per-database details (sizes, table/collection
// counts, table lists) using the connection's STORED credentials, decrypted
// by the caller. It powers the cached sizes on connections and the live
// schema drill-in. All functions honor ctx timeouts and cap per-database
// fan-out so one huge instance can't stall the API.
//
// Returned names/sizes are metadata, never secrets; errors are sanitized
// (driver messages can echo DSN fragments) via Sanitize.
package dbinspect

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/jackc/pgx/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Bounds: details are best-effort enrichment, never a full inventory scan.
const (
	MaxDatabasesForCounts = 50
	MaxTables             = 500
	MaxMongoCounts        = 100
	OpTimeout             = 10 * time.Second
)

// Sanitize bounds driver errors (single line, 300 chars) — drivers can echo
// DSN fragments, and these messages reach API responses.
func Sanitize(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if i := strings.Index(msg, "\n"); i >= 0 {
		msg = msg[:i]
	}
	msg = strings.TrimSpace(msg)
	if len(msg) > 300 {
		msg = msg[:300]
	}
	if msg == "" {
		return "database inspection failed"
	}
	return msg
}

func userinfo(user, password string) string {
	if strings.TrimSpace(user) == "" {
		return ""
	}
	return url.QueryEscape(user) + ":" + url.QueryEscape(password) + "@"
}

// CollectDatabases returns sizes + table/collection counts per database.
// connType is one of POSTGRES/MONGODB/MSSQL. Unknown engines yield an error
// (callers fall back to names-only + TCP reachability).
func CollectDatabases(ctx context.Context, connType models.ConnectionType, host string, port int, user, password string) ([]models.DatabaseInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*OpTimeout)
	defer cancel()
	switch connType {
	case models.ConnectionPostgres:
		return collectPostgres(ctx, host, port, user, password)
	case models.ConnectionMongo:
		return collectMongo(ctx, host, port, user, password)
	case models.ConnectionMssql:
		return collectMssql(ctx, host, port, user, password)
	}
	return nil, fmt.Errorf("unsupported engine %q", connType)
}

// ListTables returns tables/collections of one database with row estimates.
func ListTables(ctx context.Context, connType models.ConnectionType, host string, port int, user, password, database string) ([]models.TableInfo, error) {
	if strings.TrimSpace(database) == "" {
		return nil, fmt.Errorf("database is required")
	}
	ctx, cancel := context.WithTimeout(ctx, 3*OpTimeout)
	defer cancel()
	switch connType {
	case models.ConnectionPostgres:
		return listPostgresTables(ctx, host, port, user, password, database)
	case models.ConnectionMongo:
		return listMongoCollections(ctx, host, port, user, password, database)
	case models.ConnectionMssql:
		return listMssqlTables(ctx, host, port, user, password, database)
	}
	return nil, fmt.Errorf("unsupported engine %q", connType)
}

// --- PostgreSQL (pgx, already vendored) ---

func pgDSN(host string, port int, user, password, db string) string {
	ui := userinfo(user, password)
	if db == "" {
		db = "postgres"
	}
	return fmt.Sprintf("postgres://%s%s:%d/%s?sslmode=prefer&connect_timeout=10", ui, host, port, db)
}

func collectPostgres(ctx context.Context, host string, port int, user, password string) ([]models.DatabaseInfo, error) {
	conn, err := pgx.Connect(ctx, pgDSN(host, port, user, password, "postgres"))
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)
	rows, err := conn.Query(ctx, `SELECT datname, pg_database_size(datname) FROM pg_database WHERE datistemplate = false ORDER BY datname`)
	if err != nil {
		return nil, err
	}
	type entry struct {
		name string
		size int64
	}
	var entries []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.name, &e.size); err != nil {
			rows.Close()
			return nil, err
		}
		entries = append(entries, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]models.DatabaseInfo, 0, len(entries))
	for i, e := range entries {
		info := models.DatabaseInfo{Name: e.name, SizeBytes: e.size, TableCount: -1}
		if i < MaxDatabasesForCounts {
			var n int
			cctx, cancel := context.WithTimeout(ctx, OpTimeout)
			dbConn, derr := pgx.Connect(cctx, pgDSN(host, port, user, password, e.name))
			if derr == nil {
				derr = dbConn.QueryRow(cctx, `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog','information_schema')`).Scan(&n)
				dbConn.Close(cctx)
			}
			cancel()
			if derr == nil {
				info.TableCount = n
			}
		}
		out = append(out, info)
	}
	return out, nil
}

func listPostgresTables(ctx context.Context, host string, port int, user, password, database string) ([]models.TableInfo, error) {
	conn, err := pgx.Connect(ctx, pgDSN(host, port, user, password, database))
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)
	rows, err := conn.Query(ctx, `
		SELECT n.nspname, c.relname, pg_total_relation_size(c.oid), COALESCE(s.n_live_tup, -1)
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		LEFT JOIN pg_stat_user_tables s ON s.relid = c.oid
		WHERE c.relkind IN ('r','p') AND n.nspname NOT IN ('pg_catalog','information_schema')
		ORDER BY pg_total_relation_size(c.oid) DESC
		LIMIT $1`, MaxTables)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.TableInfo
	for rows.Next() {
		var t models.TableInfo
		if err := rows.Scan(&t.Schema, &t.Name, &t.SizeBytes, &t.Rows); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	if out == nil {
		out = []models.TableInfo{}
	}
	return out, rows.Err()
}

// --- MongoDB (mongo-driver) ---

func mongoURI(host string, port int, user, password string) string {
	if strings.TrimSpace(user) == "" {
		return fmt.Sprintf("mongodb://%s:%d/?directConnection=true", host, port)
	}
	return fmt.Sprintf("mongodb://%s/?authSource=admin", userinfo(user, password)+fmt.Sprintf("%s:%d", host, port))
}

func mongoClient(ctx context.Context, host string, port int, user, password string) (*mongo.Client, error) {
	return mongo.Connect(ctx, options.Client().ApplyURI(mongoURI(host, port, user, password)))
}

func collectMongo(ctx context.Context, host string, port int, user, password string) ([]models.DatabaseInfo, error) {
	client, err := mongoClient(ctx, host, port, user, password)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(ctx)
	pctx, cancel := context.WithTimeout(ctx, OpTimeout)
	defer cancel()
	if err := client.Ping(pctx, nil); err != nil {
		return nil, err
	}
	names, err := client.ListDatabaseNames(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	var out []models.DatabaseInfo
	for _, name := range names {
		if name == "admin" || name == "local" || name == "config" {
			continue
		}
		info := models.DatabaseInfo{Name: name, SizeBytes: models.UnknownSize, TableCount: -1}
		var stats bson.M
		if err := client.Database(name).RunCommand(ctx, bson.D{{Key: "dbStats", Value: 1}}).Decode(&stats); err == nil {
			if v, ok := stats["storageSize"]; ok {
				info.SizeBytes = bsonNumber(v)
			}
			if cols, err := client.Database(name).ListCollectionNames(ctx, bson.D{}); err == nil {
				info.TableCount = len(cols)
			}
		}
		out = append(out, info)
	}
	if out == nil {
		out = []models.DatabaseInfo{}
	}
	return out, nil
}

func listMongoCollections(ctx context.Context, host string, port int, user, password, database string) ([]models.TableInfo, error) {
	client, err := mongoClient(ctx, host, port, user, password)
	if err != nil {
		return nil, err
	}
	defer client.Disconnect(ctx)
	db := client.Database(database)
	cols, err := db.ListCollections(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cols.Close(ctx)
	type colDef struct {
		Name string `bson:"name"`
		Type string `bson:"type"`
	}
	var out []models.TableInfo
	counted := 0
	for cols.Next(ctx) {
		if len(out) >= MaxTables {
			break
		}
		var c colDef
		if err := cols.Decode(&c); err != nil {
			continue
		}
		t := models.TableInfo{Name: c.Name, Schema: c.Type, Rows: -1, SizeBytes: models.UnknownSize}
		if counted < MaxMongoCounts {
			if n, err := db.Collection(c.Name).EstimatedDocumentCount(ctx); err == nil {
				t.Rows = n
				counted++
			}
		}
		out = append(out, t)
	}
	if out == nil {
		out = []models.TableInfo{}
	}
	return out, cols.Err()
}

// bsonNumber coerces BSON numerics (int32/int64/double from dbStats).
func bsonNumber(v interface{}) int64 {
	switch n := v.(type) {
	case int32:
		return int64(n)
	case int64:
		return n
	case float64:
		return int64(n)
	}
	return models.UnknownSize
}
