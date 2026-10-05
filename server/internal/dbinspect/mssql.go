package dbinspect

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/backup-saas/server/internal/models"
	_ "github.com/microsoft/go-mssqldb"
)

// --- Microsoft SQL Server (go-mssqldb, database/sql) ---
// Best-effort: the server rarely shares the customer's LAN with SQL Server,
// so failures fall back to cached names + TCP reachability in the caller.

func mssqlDSN(host string, port int, user, password string) string {
	auth := ""
	if strings.TrimSpace(user) != "" {
		auth = userinfo(user, password)
	}
	db := "master"
	return fmt.Sprintf("sqlserver://%s%s:%d?database=%s&connection+timeout=10&dial+timeout=10", auth, host, port, db)
}

// quoteIdent brackets one identifier part for dynamic three-part names.
func quoteIdent(s string) string {
	return "[" + strings.ReplaceAll(s, "]", "]]") + "]"
}

func mssqlOpen(ctx context.Context, host string, port int, user, password string) (*sql.DB, error) {
	db, err := sql.Open("sqlserver", mssqlDSN(host, port, user, password))
	if err != nil {
		return nil, err
	}
	pctx, cancel := context.WithTimeout(ctx, OpTimeout)
	defer cancel()
	if err := db.PingContext(pctx); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func collectMssql(ctx context.Context, host string, port int, user, password string) ([]models.DatabaseInfo, error) {
	db, err := mssqlOpen(ctx, host, port, user, password)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT name FROM sys.databases WHERE state = 0 ORDER BY name`)
	if err != nil {
		return nil, err
	}
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			return nil, err
		}
		names = append(names, n)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Sizes in one shot (needs VIEW SERVER STATE; degrades to unknown).
	sizes := map[string]int64{}
	srows, err := db.QueryContext(ctx, `SELECT d.name, SUM(mf.size)*8192 FROM sys.master_files mf JOIN sys.databases d ON d.database_id = mf.database_id GROUP BY d.name`)
	if err == nil {
		for srows.Next() {
			var n string
			var s int64
			if err := srows.Scan(&n, &s); err == nil {
				sizes[n] = s
			}
		}
		srows.Close()
	}
	var out []models.DatabaseInfo
	for i, n := range names {
		info := models.DatabaseInfo{Name: n, SizeBytes: models.UnknownSize, TableCount: -1}
		if s, ok := sizes[n]; ok {
			info.SizeBytes = s
		}
		if i < MaxDatabasesForCounts {
			var c int
			q := fmt.Sprintf("SELECT COUNT(*) FROM %s.sys.tables", quoteIdent(n))
			if err := db.QueryRowContext(ctx, q).Scan(&c); err == nil {
				info.TableCount = c
			}
		}
		out = append(out, info)
	}
	if out == nil {
		out = []models.DatabaseInfo{}
	}
	return out, nil
}

func listMssqlTables(ctx context.Context, host string, port int, user, password, database string) ([]models.TableInfo, error) {
	db, err := mssqlOpen(ctx, host, port, user, password)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	q := fmt.Sprintf(`
		SELECT TOP %d s.name, t.name, SUM(p.rows), SUM(a.total_pages)*8192
		FROM %s.sys.tables t
		JOIN %s.sys.schemas s ON s.schema_id = t.schema_id
		JOIN %s.sys.partitions p ON p.object_id = t.object_id AND p.index_id IN (0,1)
		JOIN %s.sys.allocation_units a ON a.container_id = p.partition_id
		GROUP BY s.name, t.name
		ORDER BY SUM(a.total_pages) DESC`,
		MaxTables, quoteIdent(database), quoteIdent(database), quoteIdent(database), quoteIdent(database))
	rows, err := db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.TableInfo
	for rows.Next() {
		var t models.TableInfo
		if err := rows.Scan(&t.Schema, &t.Name, &t.Rows, &t.SizeBytes); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	if out == nil {
		out = []models.TableInfo{}
	}
	return out, rows.Err()
}
