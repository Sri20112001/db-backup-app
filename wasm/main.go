package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"strings"
	"sync"
	"syscall/js"
)

// Active streaming hashers keyed by integer ID
var (
	hasherLock sync.Mutex
	hashers    = make(map[int]hash.Hash)
	nextHashID = 1
)

type InspectionResult struct {
	Format      string            `json:"format"`
	Metadata    map[string]string `json:"metadata"`
	Tables      []TableEntry      `json:"tables"`
	Files       []FileEntry       `json:"files,omitempty"`
	HeaderValid bool              `json:"header_valid"`
	Error       string            `json:"error,omitempty"`
}

type TableEntry struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
	Type   string `json:"type"` // TABLE, VIEW, INDEX, SEQUENCE
	Detail string `json:"detail,omitempty"`
}

type FileEntry struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	ModTime string `json:"mod_time"`
}

func main() {
	c := make(chan struct{}, 0)

	// Register JS callbacks on global 'VaultGuardWasm'
	js.Global().Set("VaultGuardWasm", js.ValueOf(map[string]interface{}{
		"initHasher":    js.FuncOf(initHasher),
		"updateHasher":  js.FuncOf(updateHasher),
		"finishHasher":  js.FuncOf(finishHasher),
		"quickSha256":   js.FuncOf(quickSha256),
		"inspectHeader": js.FuncOf(inspectHeader),
		"version":       "1.0.0",
	}))

	println("[VaultGuard Wasm] Core engine initialized")
	<-c
}

// initHasher returns a new hasher ID for chunked hashing
func initHasher(this js.Value, args []js.Value) interface{} {
	hasherLock.Lock()
	defer hasherLock.Unlock()

	id := nextHashID
	nextHashID++
	hashers[id] = sha256.New()
	return id
}

// updateHasher(id, uint8Array) feeds a chunk into the hasher
func updateHasher(this js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return false
	}
	id := args[0].Int()
	uint8Arr := args[1]

	hasherLock.Lock()
	h, exists := hashers[id]
	hasherLock.Unlock()

	if !exists {
		return false
	}

	buf := make([]byte, uint8Arr.Get("length").Int())
	js.CopyBytesToGo(buf, uint8Arr)
	h.Write(buf)
	return true
}

// finishHasher(id) returns the final hex SHA-256 digest
func finishHasher(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return ""
	}
	id := args[0].Int()

	hasherLock.Lock()
	h, exists := hashers[id]
	if exists {
		delete(hashers, id)
	}
	hasherLock.Unlock()

	if !exists {
		return ""
	}

	return hex.EncodeToString(h.Sum(nil))
}

// quickSha256(uint8Array) computes sha256 for a small byte buffer synchronously
func quickSha256(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return ""
	}
	uint8Arr := args[0]
	buf := make([]byte, uint8Arr.Get("length").Int())
	js.CopyBytesToGo(buf, uint8Arr)

	h := sha256.Sum256(buf)
	return hex.EncodeToString(h[:])
}

// inspectHeader(uint8Array, filename) inspects a buffer (e.g. first 1MB of a dump file)
func inspectHeader(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return `{"error":"No data provided"}`
	}

	uint8Arr := args[0]
	buf := make([]byte, uint8Arr.Get("length").Int())
	js.CopyBytesToGo(buf, uint8Arr)

	filename := ""
	if len(args) >= 2 {
		filename = args[1].String()
	}

	res := parseArchiveHeader(buf, filename)
	jsonBytes, err := json.Marshal(res)
	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err.Error())
	}
	return string(jsonBytes)
}

func parseArchiveHeader(data []byte, filename string) InspectionResult {
	res := InspectionResult{
		Metadata: make(map[string]string),
		Tables:   make([]TableEntry, 0),
		Files:    make([]FileEntry, 0),
	}

	if len(data) == 0 {
		res.Error = "Empty file buffer"
		return res
	}

	// 1. Check PostgreSQL Custom Dump: Magic "PGDMP"
	if len(data) >= 5 && string(data[:5]) == "PGDMP" {
		res.Format = "PostgreSQL Custom Dump (pg_dump -Fc)"
		res.HeaderValid = true
		res.Metadata["format_code"] = "PGDMP"
		if len(data) >= 8 {
			res.Metadata["pg_major"] = fmt.Sprintf("%d", data[5])
			res.Metadata["pg_minor"] = fmt.Sprintf("%d", data[6])
			res.Metadata["pg_rev"] = fmt.Sprintf("%d", data[7])
		}

		// Parse readable strings in the header block for database/server versions
		inspectPgCustomStrings(data, &res)
		return res
	}

	// 2. Check Gzip Archive (Magic 0x1f, 0x8b)
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		res.Format = "Gzip Compressed Archive"
		res.HeaderValid = true

		gzReader, err := gzip.NewReader(bytes.NewReader(data))
		if err == nil {
			defer gzReader.Close()
			if gzReader.Name != "" {
				res.Metadata["original_filename"] = gzReader.Name
			}
			if !gzReader.ModTime.IsZero() {
				res.Metadata["modified_time"] = gzReader.ModTime.Format("2006-01-02 15:04:05 MST")
			}

			// Try reading as TAR archive inside Gzip
			tr := tar.NewReader(gzReader)
			foundTar := false
			for {
				header, err := tr.Next()
				if err != nil {
					break
				}
				foundTar = true
				res.Files = append(res.Files, FileEntry{
					Path:    header.Name,
					Size:    header.Size,
					ModTime: header.ModTime.Format("2006-01-02 15:04:05"),
				})
				if len(res.Files) >= 100 {
					break // Cap preview to 100 entries
				}
			}

			if foundTar {
				res.Format = "Tar.Gz Archive (.tar.gz)"
				return res
			}

			// If not tar, check if decompressed content is SQL
			sampleDecompressed := make([]byte, 65536)
			n, _ := gzReader.Read(sampleDecompressed)
			if n > 0 {
				inspectSqlStatements(sampleDecompressed[:n], &res)
				if len(res.Tables) > 0 {
					res.Format = "Compressed SQL Dump (.sql.gz)"
				}
			}
		}
		return res
	}

	// 3. Check Plaintext SQL Dump
	if strings.HasSuffix(strings.ToLower(filename), ".sql") || bytes.Contains(data[:min(len(data), 512)], []byte("--")) || bytes.Contains(data[:min(len(data), 512)], []byte("CREATE")) {
		inspectSqlStatements(data, &res)
		if len(res.Tables) > 0 || strings.HasSuffix(strings.ToLower(filename), ".sql") {
			res.Format = "Plain SQL Dump (.sql)"
			res.HeaderValid = true
			return res
		}
	}

	res.Format = "Generic / Raw Binary"
	res.Metadata["size_analyzed"] = fmt.Sprintf("%d bytes", len(data))
	return res
}

func inspectPgCustomStrings(data []byte, res *InspectionResult) {
	// Simple string scanner in the header area for TOC entries and DB metadata
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Split(bufio.ScanWords)

	seen := make(map[string]bool)
	for scanner.Scan() {
		word := scanner.Text()
		word = strings.Trim(word, "\x00\"';")
		if len(word) < 3 || len(word) > 64 {
			continue
		}

		// Detect table definitions or TOC tag references
		if strings.HasPrefix(word, "public.") || strings.Contains(word, "_tbl") || strings.Contains(word, "_table") {
			parts := strings.Split(word, ".")
			schema := "public"
			name := word
			if len(parts) == 2 {
				schema = parts[0]
				name = parts[1]
			}
			key := schema + "." + name
			if !seen[key] && isIdentifier(name) {
				seen[key] = true
				res.Tables = append(res.Tables, TableEntry{
					Schema: schema,
					Name:   name,
					Type:   "TABLE",
				})
			}
		}
	}
}

func inspectSqlStatements(data []byte, res *InspectionResult) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	seen := make(map[string]bool)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		upper := strings.ToUpper(line)

		if strings.HasPrefix(upper, "CREATE TABLE") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				rawName := strings.Trim(parts[2], "(\"`';")
				nameParts := strings.Split(rawName, ".")
				schema := "public"
				name := rawName
				if len(nameParts) == 2 {
					schema = nameParts[0]
					name = nameParts[1]
				}
				key := schema + "." + name
				if !seen[key] && isIdentifier(name) {
					seen[key] = true
					res.Tables = append(res.Tables, TableEntry{
						Schema: schema,
						Name:   name,
						Type:   "TABLE",
					})
				}
			}
		} else if strings.HasPrefix(upper, "CREATE VIEW") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				rawName := strings.Trim(parts[2], "(\"`';")
				key := "view." + rawName
				if !seen[key] && isIdentifier(rawName) {
					seen[key] = true
					res.Tables = append(res.Tables, TableEntry{
						Schema: "public",
						Name:   rawName,
						Type:   "VIEW",
					})
				}
			}
		} else if strings.HasPrefix(upper, "CREATE INDEX") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				rawName := strings.Trim(parts[2], "(\"`';")
				key := "index." + rawName
				if !seen[key] && isIdentifier(rawName) {
					seen[key] = true
					res.Tables = append(res.Tables, TableEntry{
						Schema: "public",
						Name:   rawName,
						Type:   "INDEX",
					})
				}
			}
		}

		if len(res.Tables) >= 150 {
			break
		}
	}

	if len(res.Tables) > 0 {
		res.HeaderValid = true
	}
}

func isIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, ch := range s {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_') {
			return false
		}
	}
	return true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
