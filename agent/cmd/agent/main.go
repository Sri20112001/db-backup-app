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
//	AGENT_REGISTRATION_KEY  one-time key, first run only (legacy path)
//	AGENT_ENROLLMENT_TOKEN  one-time enrollment token (preferred first-run path)
//	AGENT_CONFIG_FILE       JSON config path, non-secret settings only
//	AGENT_CREDENTIAL_DIR    credential store directory (default platform config dir)
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
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	agent "github.com/backup-saas/agent/internal/agent"
	"github.com/backup-saas/agent/internal/agentservice"
	"github.com/backup-saas/agent/internal/credstore"
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

	// Service control flags (also invoked by the setup GUI installer).
	var (
		flagConfig    = flag.String("config", "", "JSON config file (non-secret settings only)")
		flagInstall   = flag.Bool("install", false, "install as a Windows service and exit")
		flagUninstall = flag.Bool("uninstall", false, "uninstall the Windows service and exit")
		flagStart     = flag.Bool("start", false, "start the Windows service and exit")
		flagStop      = flag.Bool("stop", false, "stop the Windows service and exit")
		flagRestart   = flag.Bool("restart", false, "restart the Windows service and exit")
		flagStatus    = flag.Bool("status", false, "print service + credential status and exit")
	)
	flag.Parse()
	if *flagConfig != "" {
		os.Setenv("AGENT_CONFIG_FILE", *flagConfig)
	}
	switch {
	case *flagInstall:
		configPath := *flagConfig
		if configPath == "" {
			configPath = agent.DefaultConfigFile()
		}
		// Self-install: this binary IS the agent, so registering it is correct
		// (unlike the setup GUI, which must pass the installed agent path).
		self, err := agentservice.ExePath()
		if err != nil {
			log.Fatalf("install: %v", err)
		}
		if err := agentservice.Install(self, "-config", configPath); err != nil {
			log.Fatalf("install: %v", err)
		}
		log.Print("service installed")
		return
	case *flagUninstall:
		if err := agentservice.Uninstall(); err != nil {
			log.Fatalf("uninstall: %v", err)
		}
		log.Print("service uninstalled")
		return
	case *flagStart:
		if err := agentservice.Start(); err != nil {
			log.Fatalf("start: %v", err)
		}
		log.Print("service started")
		return
	case *flagStop:
		if err := agentservice.Stop(); err != nil {
			log.Fatalf("stop: %v", err)
		}
		log.Print("service stopped")
		return
	case *flagRestart:
		if err := agentservice.Restart(); err != nil {
			log.Fatalf("restart: %v", err)
		}
		log.Print("service restarted")
		return
	case *flagStatus:
		printStatus()
		return
	}

	// Under the Service Control Manager there is no console: log to the
	// config-dir file (never secrets — the agent never logs credentials).
	if !agentservice.IsInteractive() {
		if f, err := os.OpenFile(filepath.Join(agent.CredentialDir(), "agent.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			log.SetOutput(f)
			defer f.Close()
		}
		runService()
		return
	}

	runConsole()
}

// printStatus reports service state + credential presence. Never prints
// secrets — only whether a credential exists and where it lives.
func printStatus() {
	svcState, err := agentservice.Status()
	if err != nil {
		log.Printf("service: %v", err)
	} else {
		log.Printf("service VaultGuardAgent: %s", svcState)
	}
	store := credstore.NewDefaultStore(agent.CredentialDir())
	if creds, err := store.Load(); err == nil && creds.AgentID != "" {
		log.Printf("credential: present for agent %s (store: %T)", creds.AgentID, store)
	} else {
		log.Printf("credential: none stored (run setup enrollment first)")
	}
	log.Printf("config: %s", agent.DefaultConfigFile())
}

// runService executes the agent under the Service Control Manager.
func runService() {
	if err := agentservice.Run(func(stop <-chan struct{}) {
		runAgent(stop)
	}); err != nil {
		log.Fatalf("service: %v", err)
	}
}

// runConsole executes the agent as a foreground process.
func runConsole() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan struct{})
	go func() {
		defer close(done)
		runAgent(nil)
	}()
	select {
	case <-ctx.Done():
	case <-done:
	}
	log.Print("stopped")
}

// runAgent is the shared agent body: configure, resolve credentials, poll.
// A nil stop channel means "run until the process is killed".

func runAgent(stop <-chan struct{}) {
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

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if stop != nil {
		// Service mode: SCM owns the lifetime.
		go func() {
			<-stop
			cancel()
		}()
	}

	store := credstore.NewDefaultStore(agent.CredentialDir())
	st, err := agent.ResolveCredentials(cfg, store)
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
