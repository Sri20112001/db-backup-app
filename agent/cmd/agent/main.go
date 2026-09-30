// Command agent is the VaultGuard backup agent.
//
// It runs on machines to be backed up and talks to the server over HTTP:
//
//  1. First run: POST /api/agents/register with AGENT_REGISTRATION_KEY
//     (generate the key in the UI: Agents -> token). Credentials are cached
//     in AGENT_STATE_FILE for subsequent runs.
//  2. Steady state: poll /api/agent/runs and /api/agent/restores, claim
//     work, execute FILESYSTEM, POSTGRES, MONGODB and MSSQL backups to LOCAL
//     storage targets,
//     and report progress/results. Polling doubles as the heartbeat that
//     keeps the agent ONLINE.
//
// Configuration via environment:
//
//	AGENT_SERVER            API base URL (default http://localhost:7541)
//	AGENT_REGISTRATION_KEY  one-time key, first run only
//	AGENT_STATE_FILE        credential cache (default <exe-dir>/agent-state.json)
//	AGENT_HOSTNAME          override detected hostname
//	AGENT_POLL_INTERVAL     e.g. 30s (default 30s, min 5s)
//	AGENT_PG_HOST           PostgreSQL host for POSTGRES jobs (default localhost)
//	AGENT_PG_PORT           PostgreSQL port (default 5432)
//	AGENT_PG_USER           PostgreSQL user (default postgres)
//	AGENT_PG_PASSWORD       PostgreSQL password (default empty)
//	AGENT_MSSQL_SERVER      SQL Server host[\INSTANCE] for MSSQL jobs (default localhost)
//	AGENT_MSSQL_USER        SQL login (default empty = Windows auth)
//	AGENT_MSSQL_PASSWORD    SQL password (default empty)
//	AGENT_MSSQL_BACKUP_DIR  .bak staging dir, engine-local (default OS temp dir)
//	AGENT_MONGO_URI         MongoDB connection string (default mongodb://localhost:27017)
//	AGENT_TLS_SKIP_VERIFY   accept self-signed certs, e.g. true (default false)
package main

import (
	"context"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	agent "github.com/backup-saas/agent/internal/agent"
)

// isLoopbackServer reports whether the configured server URL targets this
// machine (plain HTTP is acceptable there — packets never leave the host).
func isLoopbackServer(serverURL string) bool {
	u, err := url.Parse(serverURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "" || host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("agent: ")

	cfg := agent.LoadConfig()

	// TLS posture: skipping verification is a lab-only escape hatch. Loud
	// on purpose — a silent MITM on backup traffic would expose database
	// contents and the agent token. Production agents must trust the
	// server CA instead.
	if cfg.TLSSkipVerify {
		log.Print("WARNING: AGENT_TLS_SKIP_VERIFY=true — server identity is NOT verified; never use outside a lab")
	}
	if !strings.HasPrefix(cfg.Server, "https://") && !isLoopbackServer(cfg.Server) {
		log.Print("WARNING: AGENT_SERVER is plain HTTP to a non-loopback host — backup traffic is unencrypted")
	}

	if cfg.BrowseAddr != "" {
		agent.StartBrowseServer(cfg.BrowseAddr, cfg.BrowseToken, cfg.DashboardOrigin)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := agent.EnsureRegistered(cfg)
	if err != nil {
		if cfg.BrowseAddr != "" {
			// Stay alive as a folder-browser source for the dashboard
			// picker even before registration.
			log.Printf("not registered (%v); polling disabled, folder browser still serving", err)
			<-ctx.Done()
			log.Print("stopped")
			return
		}
		log.Fatalf("setup: %v", err)
	}

	client := agent.NewClientWithTLS(cfg.Server, st.AgentToken, cfg.TLSSkipVerify)
	client.SetAgentID(st.AgentID)

	// Live socket: instant cancels + log streaming to dashboards.
	// Polling keeps working if the socket drops.
	ws := agent.NewWSClient(cfg.Server, st.AgentID, st.AgentToken, cfg.TLSSkipVerify)
	go ws.Run(ctx, nil)
	runner := agent.NewRunner(cfg, client, ws)

	log.Printf("hostname=%s poll=%s", cfg.Hostname, cfg.PollInterval)
	runner.Run(ctx)
	log.Print("stopped")
}
