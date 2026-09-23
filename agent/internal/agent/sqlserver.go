package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// SqlDatabase is one row from sys.databases.
type SqlDatabase struct {
	Name  string `json:"name"`
	State string `json:"state,omitempty"`
}

// DatabasesResponse is the payload of GET /databases. Databases is never
// null so the dashboard can always .map over it. Error is set when discovery
// was not possible (e.g. sqlcmd missing, no local instance reachable) — the
// dashboard then falls back to manual database-name entry.
type DatabasesResponse struct {
	Databases []SqlDatabase `json:"databases"`
	Source    string        `json:"source,omitempty"`
	Error     string        `json:"error,omitempty"`
}

func handleDatabases(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Same loopback-helper policy as /browse: any local dashboard origin.
	w.Header().Set("Access-Control-Allow-Origin", "*")

	out := DatabasesResponse{Databases: []SqlDatabase{}}
	dbs, source, err := discoverDatabases()
	if err != nil {
		out.Error = err.Error()
	} else {
		out.Databases = dbs
		out.Source = source
	}
	json.NewEncoder(w).Encode(out)
}

// discoverDatabases tries, in order:
//  1. sqlcmd (Windows auth) against the usual local instance names.
//  2. PowerShell + built-in .NET SqlClient — ships with Windows, needs no
//     extra tools installed.
// It intentionally shells out instead of adding a TDS driver.
func discoverDatabases() ([]SqlDatabase, string, error) {
	if _, err := exec.LookPath("sqlcmd"); err == nil {
		for _, server := range []string{"localhost", `localhost\SQLEXPRESS`} {
			if dbs, err := querySQLCmd(server); err == nil {
				return dbs, "sqlcmd " + server, nil
			}
		}
	}
	for _, server := range []string{"localhost", `localhost\SQLEXPRESS`} {
		if dbs, err := queryPowerShell(server); err == nil {
			return dbs, "powershell " + server, nil
		}
	}
	return nil, "", fmt.Errorf("no local SQL Server reachable (tried sqlcmd and the built-in .NET client against localhost, localhost\\SQLEXPRESS). Install SQL Server Command Line Utilities, or type the database name manually")
}

func querySQLCmd(server string) ([]SqlDatabase, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// -h -1: no headers/separators. -W: trim spaces. -s "|": field separator.
	// -E: trusted (Windows) auth — the agent runs as a service account.
	cmd := exec.CommandContext(ctx, "sqlcmd",
		"-S", server, "-E", "-h", "-1", "-W", "-s", "|",
		"-Q", "SET NOCOUNT ON; SELECT name + '|' + state_desc FROM sys.databases ORDER BY name;",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s: %s", server, msg)
	}

	var dbs []SqlDatabase
	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "(") {
			continue // skip blank lines and "(N rows affected)"
		}
		name, state, _ := strings.Cut(line, "|")
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		dbs = append(dbs, SqlDatabase{Name: name, State: strings.TrimSpace(state)})
	}
	if len(dbs) == 0 {
		return nil, fmt.Errorf("%s: query succeeded but returned no databases", server)
	}
	return dbs, nil
}

// queryPowerShell lists databases via the .NET SqlClient built into Windows
// PowerShell — no SQL tooling required. Uses Windows (trusted) auth, so the
// agent's service account needs CONNECT + VIEW ANY DATABASE on the instance.
func queryPowerShell(server string) ([]SqlDatabase, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	script := `$c = New-Object System.Data.SqlClient.SqlConnection('Server=` +
		server + `;Integrated Security=True;Connection Timeout=5');` +
		` $c.Open();` +
		` $r = (New-Object System.Data.SqlClient.SqlCommand('SELECT name FROM sys.databases ORDER BY name', $c)).ExecuteReader();` +
		` while ($r.Read()) { $r.GetString(0) }; $c.Close()`

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s: %s", server, firstLine(msg))
	}

	var dbs []SqlDatabase
	for _, line := range strings.Split(stdout.String(), "\n") {
		if name := strings.TrimSpace(line); name != "" {
			dbs = append(dbs, SqlDatabase{Name: name})
		}
	}
	if len(dbs) == 0 {
		return nil, fmt.Errorf("%s: query succeeded but returned no databases", server)
	}
	return dbs, nil
}

func firstLine(s string) string {
	if i := strings.Index(s, "\n"); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}
