package setup

import (
	"strings"

	"github.com/backup-saas/agent/internal/agent"
)

// defaultSetupConfigDir reuses the agent's platform config directory so the
// installer and the service agree on paths by construction.
func defaultSetupConfigDir() string {
	return agent.DefaultConfigDir()
}

// SavedServerURL returns the server URL from a previous install, if any.
// Wizards use it to prefill the field so the URL is asked once, then kept.
// Empty means no previous install (or an unreadable one).
func SavedServerURL() string {
	return savedServerURLFrom(agent.DefaultConfigFile())
}

func savedServerURLFrom(path string) string {
	fs, err := agent.ReadConfigFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(fs.ServerURL)
}
