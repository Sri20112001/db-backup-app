package credstore

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Shared small helpers (used by the DPAPI backend; FileStore uses
// encoding/json+os directly in store.go).

func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func ensureParentDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0700)
}

func writeFile0600(path string, data []byte) error {
	return os.WriteFile(path, data, 0600)
}

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func parseCredentials(data []byte) (Credentials, error) {
	var c Credentials
	if err := json.Unmarshal(data, &c); err != nil {
		return Credentials{}, err
	}
	if c.AgentID == "" || c.AgentToken == "" {
		return Credentials{}, errIncomplete
	}
	return c, nil
}
