// Command agent is the VaultGuard backup agent.
//
// It runs on machines to be backed up and talks to the server over HTTP:
//
//  1. First run: POST /api/agents/register with AGENT_REGISTRATION_KEY
//     (generate the key in the UI: Agents -> token). Credentials are cached
//     in AGENT_STATE_FILE for subsequent runs.
//  2. Steady state: poll /api/agent/runs and /api/agent/restores, claim
//     work, execute FILESYSTEM and POSTGRES backups to LOCAL storage targets,
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
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	agent "github.com/backup-saas/agent/internal/agent"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("agent: ")

	cfg := agent.LoadConfig()

	if cfg.BrowseAddr != "" {
		agent.StartBrowseServer(cfg.BrowseAddr)
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

	client := agent.NewClient(cfg.Server, st.AgentToken)
	client.SetAgentID(st.AgentID)
	runner := agent.NewRunner(cfg, client)

	log.Printf("hostname=%s poll=%s", cfg.Hostname, cfg.PollInterval)
	runner.Run(ctx)
	log.Print("stopped")
}
