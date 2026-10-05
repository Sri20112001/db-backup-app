package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/backup-saas/server/internal/dbinspect"
	"github.com/backup-saas/server/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ConnectionHandler manages reusable database connection metadata.
// Secrets are write-only: the API accepts `password` on create/update,
// stores only AES-256-GCM ciphertext, and never returns it. Test/databases
// endpoints never echo credentials in responses or logs.
type ConnectionHandler struct {
	db     *gorm.DB
	encKey []byte
}

func NewConnectionHandler(db *gorm.DB, encKey []byte) *ConnectionHandler {
	return &ConnectionHandler{db: db, encKey: encKey}
}

type connectionRequest struct {
	Name      string `json:"name" binding:"required"`
	Type      string `json:"type" binding:"required"`
	AgentID   string `json:"agent_id" binding:"required"`
	Host      string `json:"host" binding:"required"`
	Port      int    `json:"port"`
	// Username/Password are optional: MongoDB without access control and
	// MSSQL with Windows integrated auth legitimately have none. Empty
	// means "connect without credentials" — never "skip validation".
	Username  string `json:"username"`
	Password  string `json:"password"`
	Databases []string `json:"databases"`
}

func defaultPortFor(connType models.ConnectionType) int {
	switch connType {
	case models.ConnectionPostgres:
		return 5432
	case models.ConnectionMongo:
		return 27017
	case models.ConnectionMssql:
		return 1433
	}
	return 0
}

// normalizeConnectionType accepts canonical POSTGRES/MONGODB/MSSQL plus the
// backup-job aliases MSSQL_SERVER and SQL_SERVER (both map to MSSQL).
func normalizeConnectionType(raw string) (models.ConnectionType, error) {
	t := models.ConnectionType(strings.ToUpper(strings.TrimSpace(raw)))
	switch t {
	case models.ConnectionPostgres, models.ConnectionMongo, models.ConnectionMssql:
		return t, nil
	case "MSSQL_SERVER", "SQL_SERVER":
		return models.ConnectionMssql, nil
	}
	return "", fmt.Errorf("unsupported connection type %q (supported: POSTGRES, MONGODB, MSSQL)", raw)
}

// connectionDTO is the safe list/get shape: metadata only, never secrets.
// Databases carry cached details (sizes, table counts); unknown details
// arrive as -1 sentinels the UI renders as "—".
func toConnectionDTO(c models.DatabaseConnection) gin.H {
	meta := c.MetadataList()
	return gin.H{
		"id":              c.ID.String(),
		"organization_id": c.OrganizationID.String(),
		"agent_id":        c.AgentID.String(),
		"name":            c.Name,
		"type":            string(c.Type),
		"host":            c.Host,
		"port":            c.Port,
		"username":        c.Username,
		"status":          string(c.Status),
		"last_checked_at": c.LastCheckedAt,
		"databases":       meta,
		"database_count":  len(meta),
		"agent":           c.Agent,
		"created_at":      c.CreatedAt,
		"updated_at":      c.UpdatedAt,
	}
}

func (h *ConnectionHandler) loadOwned(c *gin.Context) (*models.DatabaseConnection, bool) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return nil, false
	}
	var conn models.DatabaseConnection
	if err := h.db.Preload("Agent").Where("id = ? AND organization_id = ?", id, orgID).First(&conn).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "connection not found"})
		return nil, false
	}
	return &conn, true
}

func (h *ConnectionHandler) List(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	var conns []models.DatabaseConnection
	h.db.Preload("Agent").Where("organization_id = ?", orgID).Order("created_at ASC").Find(&conns)
	out := make([]gin.H, 0, len(conns))
	for _, cn := range conns {
		out = append(out, toConnectionDTO(cn))
	}
	c.JSON(http.StatusOK, out)
}

func (h *ConnectionHandler) Get(c *gin.Context) {
	conn, ok := h.loadOwned(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, toConnectionDTO(*conn))
}

func (h *ConnectionHandler) Create(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	var req connectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	connType, err := normalizeConnectionType(req.Type)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
		return
	}
	var agent models.Agent
	if err := h.db.Where("id = ? AND organization_id = ?", agentID, orgID).First(&agent).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent not found in this organization"})
		return
	}
	port := req.Port
	if port == 0 {
		port = defaultPortFor(connType)
	}
	if port < 1 || port > 65535 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "port must be 1-65535"})
		return
	}
	// Empty password = no-auth endpoint (or Windows integrated auth). It is
	// still envelope-encrypted for uniform handling, never stored plaintext.
	enc, err := encrypt(h.encKey, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
		return
	}
	conn := models.DatabaseConnection{
		Base:              models.Base{ID: uuid.New()},
		OrganizationID:    orgID,
		AgentID:           agentID,
		Name:              strings.TrimSpace(req.Name),
		Type:              connType,
		Host:              strings.TrimSpace(req.Host),
		Port:              port,
		Username:          strings.TrimSpace(req.Username),
		EncryptedPassword: enc,
		Status:            models.ConnectionUnknown,
		DatabaseNames:     strings.Join(req.Databases, "\n"),
	}
	if err := h.db.Create(&conn).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "idx_database_connections_org_name") {
			c.JSON(http.StatusConflict, gin.H{"error": "a connection with this name already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	h.db.Preload("Agent").First(&conn, conn.ID)
	c.JSON(http.StatusCreated, toConnectionDTO(conn))
}

func (h *ConnectionHandler) Update(c *gin.Context) {
	conn, ok := h.loadOwned(c)
	if !ok {
		return
	}
	// Read the raw body so password semantics are presence-based: a "password"
	// key (even "") rotates/clears the credential, while an absent key keeps
	// it. This allows authed → no-auth conversion, which empty-means-keep
	// could never express. (ShouldBindJSON would erase that distinction.)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unreadable request"})
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	var req connectionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(body, &raw)
	_, wantPasswordChange := raw["password"]
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Type) == "" ||
		strings.TrimSpace(req.AgentID) == "" || strings.TrimSpace(req.Host) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, type, agent_id and host are required"})
		return
	}
	orgID := c.MustGet("org_id").(uuid.UUID)
	connType, err := normalizeConnectionType(req.Type)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	agentID, err := uuid.Parse(req.AgentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
		return
	}
	var agent models.Agent
	if err := h.db.Where("id = ? AND organization_id = ?", agentID, orgID).First(&agent).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent not found in this organization"})
		return
	}
	port := req.Port
	if port == 0 {
		port = defaultPortFor(connType)
	}
	updates := map[string]interface{}{
		"name":     strings.TrimSpace(req.Name),
		"type":     string(connType),
		"agent_id": agentID,
		"host":     strings.TrimSpace(req.Host),
		"port":     port,
		"username": strings.TrimSpace(req.Username),
	}
	if wantPasswordChange {
		// Present (even empty = convert to no-auth): rotate and invalidate.
		enc, err := encrypt(h.encKey, req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
			return
		}
		updates["encrypted_password"] = enc
		// Credential rotation invalidates the last check.
		updates["status"] = string(models.ConnectionUnknown)
	}
	if req.Databases != nil {
		updates["database_names"] = strings.Join(req.Databases, "\n")
		// Preserve cached sizes/counts across names-only saves by name match.
		merged := models.MergeMetadataNames(conn.MetadataList(), req.Databases)
		updates["database_metadata"] = models.MarshalDatabaseMetadata(merged)
	}
	if err := h.db.Model(conn).Updates(updates).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "idx_database_connections_org_name") {
			c.JSON(http.StatusConflict, gin.H{"error": "a connection with this name already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	h.db.Preload("Agent").First(conn, conn.ID)
	c.JSON(http.StatusOK, toConnectionDTO(*conn))
}

func (h *ConnectionHandler) Delete(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	conn, ok := h.loadOwned(c)
	if !ok {
		return
	}
	var jobCount int64
	h.db.Model(&models.BackupJob{}).Where("connection_id = ? AND organization_id = ?", conn.ID, orgID).Count(&jobCount)
	if jobCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("connection is used by %d backup job(s); reassign or delete them first", jobCount)})
		return
	}
	h.db.Delete(conn)
	c.JSON(http.StatusNoContent, nil)
}

// Test verifies reachability with the stored encrypted credentials and
// refreshes the cached database details (sizes, table counts) best-effort.
// The password is decrypted in memory only, never logged or returned.
func (h *ConnectionHandler) Test(c *gin.Context) {
	conn, ok := h.loadOwned(c)
	if !ok {
		return
	}
	password, err := decrypt(h.encKey, conn.EncryptedPassword)
	if err != nil {
		h.markStatus(conn, models.ConnectionError)
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": "stored credential is unusable; re-enter the password"})
		return
	}
	// Empty password is legitimate (no-auth MongoDB, Windows integrated
	// auth) — the live connection attempt below is the real validation.
	start := time.Now()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()

	meta, testErr := dbinspect.CollectDatabases(ctx, conn.Type, conn.Host, conn.Port, conn.Username, password)
	if testErr != nil && conn.Type != models.ConnectionPostgres {
		// The server rarely shares the customer's LAN: fall back to TCP
		// reachability for non-Postgres engines (full auth is validated
		// agent-side via the loopback helper before save).
		if terr := testTCP(ctx, conn.Host, conn.Port); terr == nil {
			testErr = nil
			meta = conn.MetadataList()
		} else {
			testErr = terr
		}
	}
	if testErr != nil {
		h.markStatus(conn, models.ConnectionError)
		c.JSON(http.StatusBadGateway, gin.H{"status": "error", "error": dbinspect.Sanitize(testErr)})
		return
	}
	names := make([]string, 0, len(meta))
	for _, d := range meta {
		names = append(names, d.Name)
	}
	updates := map[string]interface{}{
		"status":            string(models.ConnectionConnected),
		"last_checked_at":   time.Now(),
		"database_names":    strings.Join(names, "\n"),
		"database_metadata": models.MarshalDatabaseMetadata(meta),
	}
	h.db.Model(conn).Updates(updates)
	h.db.Preload("Agent").First(conn, conn.ID)
	c.JSON(http.StatusOK, gin.H{
		"status":       "ok",
		"latency_ms":   time.Since(start).Milliseconds(),
		"databases":    conn.MetadataList(),
		"agent_status": string(conn.Agent.Status),
	})
}

// Databases returns cached database details for the job wizard and detail
// views, refreshing live when reachable. Never includes credentials.
func (h *ConnectionHandler) Databases(c *gin.Context) {
	conn, ok := h.loadOwned(c)
	if !ok {
		return
	}
	if password, err := decrypt(h.encKey, conn.EncryptedPassword); err == nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
		defer cancel()
		if live, err := dbinspect.CollectDatabases(ctx, conn.Type, conn.Host, conn.Port, conn.Username, password); err == nil && len(live) > 0 {
			names := make([]string, 0, len(live))
			for _, d := range live {
				names = append(names, d.Name)
			}
			h.db.Model(conn).Updates(map[string]interface{}{
				"database_names":    strings.Join(names, "\n"),
				"database_metadata": models.MarshalDatabaseMetadata(live),
				"status":            string(models.ConnectionConnected),
				"last_checked_at":   time.Now(),
			})
			c.JSON(http.StatusOK, gin.H{"databases": live, "cached": false, "agent_status": string(conn.Agent.Status)})
			return
		}
	}
	cached := conn.MetadataList()
	c.JSON(http.StatusOK, gin.H{"databases": cached, "cached": true, "agent_status": string(conn.Agent.Status)})
}

// Tables drills into one database (tables/collections with row estimates)
// using the stored credential. Live query, never cached, never secrets.
func (h *ConnectionHandler) Tables(c *gin.Context) {
	conn, ok := h.loadOwned(c)
	if !ok {
		return
	}
	dbName := strings.TrimSpace(c.Param("dbname"))
	if dbName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "database name is required"})
		return
	}
	// Guard against cross-database probing outside the discovered list.
	known := false
	for _, d := range conn.MetadataList() {
		if d.Name == dbName {
			known = true
			break
		}
	}
	if !known {
		for _, d := range conn.DatabaseList() {
			if d == dbName {
				known = true
				break
			}
		}
	}
	if !known {
		c.JSON(http.StatusNotFound, gin.H{"error": "database not found on this connection"})
		return
	}
	password, err := decrypt(h.encKey, conn.EncryptedPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "stored credential is unusable; re-enter the password"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 45*time.Second)
	defer cancel()
	tables, err := dbinspect.ListTables(ctx, conn.Type, conn.Host, conn.Port, conn.Username, password, dbName)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": dbinspect.Sanitize(err)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"database": dbName, "tables": tables})
}

func (h *ConnectionHandler) markStatus(conn *models.DatabaseConnection, st models.ConnectionStatus) {
	now := time.Now()
	h.db.Model(conn).Updates(map[string]interface{}{"status": string(st), "last_checked_at": &now})
}

func testTCP(ctx context.Context, host string, port int) error {
	d := net.Dialer{Timeout: 10 * time.Second}
	nc, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return err
	}
	nc.Close()
	return nil
}
