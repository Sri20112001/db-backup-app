package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// BrowseEntry is one directory listing row.
type BrowseEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
}

// BrowseResponse is the payload of GET /browse.
type BrowseResponse struct {
	CWD     string       `json:"cwd"`
	Parent  string       `json:"parent"`
	Roots   []string     `json:"roots"`
	Entries []BrowseEntry `json:"entries"`
}

// StartBrowseServer serves a read-only directory browser on a loopback
// address (default 127.0.0.1:7546). It intentionally binds loopback only:
// the dashboard running on the same machine uses it for the folder picker.
// Remote browsers cannot reach it; for remote agents type the path manually.
func StartBrowseServer(addr string) {
	if addr == "" {
		addr = "127.0.0.1:7546"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/browse", handleBrowse)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})
	go func() {
		log.Printf("folder browser listening on %s (loopback only)", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Printf("folder browser stopped: %v", err)
		}
	}()
}

func driveRoots() []string {
	if runtime.GOOS != "windows" {
		return []string{"/"}
	}
	var roots []string
	for c := 'A'; c <= 'Z'; c++ {
		root := fmt.Sprintf("%c:\\", c)
		if st, err := os.Stat(root); err == nil && st.IsDir() {
			roots = append(roots, root)
		}
	}
	return roots
}

func handleBrowse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Allow the dashboard (any local origin/port) to call this endpoint.
	w.Header().Set("Access-Control-Allow-Origin", "*")

	reqPath := strings.TrimSpace(r.URL.Query().Get("path"))
	var dir string
	if reqPath == "" {
		roots := driveRoots()
		// Start at the first usable root (usually C:\), fall back to cwd.
		dir = ""
		if len(roots) > 0 {
			dir = roots[0]
		} else if cwd, err := os.Getwd(); err == nil {
			dir = cwd
		}
	} else {
		dir = filepath.Clean(reqPath)
	}

	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "not a readable directory"})
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "cannot list directory"})
		return
	}

	out := BrowseResponse{CWD: dir, Roots: driveRoots(), Entries: []BrowseEntry{}}
	if parent := filepath.Dir(dir); parent != dir {
		out.Parent = parent
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue // folders only — backups take directories (or a file typed manually)
		}
		out.Entries = append(out.Entries, BrowseEntry{
			Name:  e.Name(),
			Path:  filepath.Join(dir, e.Name()),
			IsDir: true,
		})
	}
	sort.Slice(out.Entries, func(i, j int) bool { return out.Entries[i].Name < out.Entries[j].Name })
	json.NewEncoder(w).Encode(out)
}
