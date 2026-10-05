package setup

import "github.com/backup-saas/agent/internal/agent"

// defaultSetupConfigDir reuses the agent's platform config directory so the
// installer and the service agree on paths by construction.
func defaultSetupConfigDir() string {
	return agent.DefaultConfigDir()
}
