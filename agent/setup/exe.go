package setup

import (
	"os"
	"path/filepath"
)

// AgentExeNextTo locates the agent binary shipped next to the setup
// executable (the release layout ships both). Empty = skip binary copy
// (service registration still proceeds if the binary is pre-staged).
// Shared by the Fyne GUI and the console setup so both resolve the same way.
func AgentExeNextTo(setupExePath string) string {
	if setupExePath == "" {
		if exe, err := os.Executable(); err == nil {
			setupExePath = exe
		} else {
			return ""
		}
	}
	for _, name := range []string{"VaultGuard-Agent.exe", "vaultguard-agent", "vaultguard-agent.exe"} {
		candidate := filepath.Join(filepath.Dir(setupExePath), name)
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return candidate
		}
	}
	return ""
}
