package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/backup-saas/server/internal/realtime"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// errAgentRevoked aborts enrollment/rotation for revoked agents.
// errTokenConsumed aborts a concurrent/replayed enrollment that lost the
// single-use claim race. Both map to generic 401s (no oracle detail).
var (
	errAgentRevoked  = errors.New("agent revoked")
	errTokenConsumed = errors.New("token consumed")
)

// EnrollmentHandler implements the agent lifecycle milestone: one-time
// enrollment tokens, enrollment, heartbeat, and revocation. Permanent agent
// credentials reuse the existing bcrypt token_hash mechanism (shared with
// Register/RotateToken via mintAgentToken) — no second auth system.
type EnrollmentHandler struct {
	db   *gorm.DB
	grpc StreamDisconnector
}

func NewEnrollmentHandler(db *gorm.DB, grpc StreamDisconnector) *EnrollmentHandler {
	return &EnrollmentHandler{db: db, grpc: grpc}
}

// sha256Hex digests one-time tokens for storage. Plaintext tokens are shown
// once at creation, never stored, never logged.
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// mintAgentToken creates a permanent agent credential: the plaintext token
// (returned once to the agent) and its bcrypt hash (the only server-side
// representation).
func mintAgentToken() (token, hash string, err error) {
	token = randomHex(32)
	h, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}
	return token, string(h), nil
}

// enrollmentExpired is the pure expiry predicate (unit-tested).
func enrollmentExpired(expiresAt, now time.Time) bool {
	return !now.Before(expiresAt)
}

type generateEnrollmentRequest struct {
	Name           string `json:"name"`
	ExpiresMinutes int    `json:"expires_minutes"`
}

// GenerateEnrollmentToken creates a short-lived single-use enrollment token,
// optionally linked to a pre-created pending agent (named now, enrolled
// later by the setup GUI). Owner/Admin only (router-enforced).
func (h *EnrollmentHandler) GenerateEnrollmentToken(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	var req generateEnrollmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ttl := req.ExpiresMinutes
	if ttl == 0 {
		ttl = 30
	}
	if ttl < 1 || ttl > 1440 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "expires_minutes must be 1-1440"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "pending-" + randomHex(4)
	}
	token := randomHex(32)
	expiresAt := time.Now().Add(time.Duration(ttl) * time.Minute)

	var agentID uuid.UUID
	err := h.db.Transaction(func(tx *gorm.DB) error {
		agent := models.Agent{
			Base:           models.Base{ID: uuid.New()},
			OrganizationID: orgID,
			Name:           name,
			Status:         models.AgentOffline,
		}
		if err := tx.Create(&agent).Error; err != nil {
			return err
		}
		tok := models.EnrollmentToken{
			Base:           models.Base{ID: uuid.New()},
			OrganizationID: orgID,
			AgentID:        &agent.ID,
			TokenHash:      sha256Hex(token),
			ExpiresAt:      expiresAt,
		}
		if uid, err := uuid.Parse(c.GetString("user_id")); err == nil {
			tok.CreatedBy = &uid
		}
		if err := tx.Create(&tok).Error; err != nil {
			return err
		}
		agentID = agent.ID
		return nil
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create enrollment token"})
		return
	}
	publishEntity(orgID.String(), realtime.TypeAgents, "token-created", agentID.String())
	// The plaintext token is returned exactly once — never stored, never logged.
	c.JSON(http.StatusCreated, gin.H{
		"agent_id":         agentID,
		"enrollment_token": token,
		"expires_at":       expiresAt,
	})
}

type enrollRequest struct {
	EnrollmentToken string `json:"enrollment_token" binding:"required"`
	MachineName     string `json:"machine_name" binding:"required"`
	Platform        string `json:"platform"`
	Architecture    string `json:"architecture"`
	AgentVersion    string `json:"agent_version"`
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`
	IPAddress       string `json:"ip_address"`
}

// Enroll exchanges a one-time enrollment token for a permanent agent
// credential. Public endpoint (token IS the auth). Single-use enforced
// atomically; expired/used/unknown tokens are rejected without distinction
// in detail (all 401) to avoid oracle behavior.
func (h *EnrollmentHandler) Enroll(c *gin.Context) {
	var req enrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var tok models.EnrollmentToken
	if err := h.db.Where("token_hash = ?", sha256Hex(strings.TrimSpace(req.EnrollmentToken))).First(&tok).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid enrollment token"})
		return
	}
	if tok.UsedAt != nil || enrollmentExpired(tok.ExpiresAt, time.Now()) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid enrollment token"})
		return
	}
	if tok.AgentID == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid enrollment token"})
		return
	}

	agentToken, tokenHash, err := mintAgentToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}
	name := strings.TrimSpace(req.MachineName)
	if name == "" {
		name = strings.TrimSpace(req.Hostname)
	}
	now := time.Now()
	err = h.db.Transaction(func(tx *gorm.DB) error {
		// Claim the token: exactly one concurrent enrollment can flip
		// used_at from NULL. RowsAffected==0 means we lost the race (or a
		// replay) — abort without issuing a second credential.
		claimed := tx.Model(&models.EnrollmentToken{}).Where("id = ? AND used_at IS NULL", tok.ID).
			Update("used_at", time.Now())
		if claimed.Error != nil {
			return claimed.Error
		}
		if claimed.RowsAffected == 0 {
			return errTokenConsumed
		}
		var agent models.Agent
		if err := tx.Where("id = ? AND organization_id = ?", *tok.AgentID, tok.OrganizationID).First(&agent).Error; err != nil {
			return err
		}
		if agent.RevokedAt != nil {
			return errAgentRevoked
		}
		if err := tx.Model(&agent).Updates(map[string]interface{}{
			"name":             name,
			"machine_name":     strings.TrimSpace(req.MachineName),
			"platform":         strings.ToLower(strings.TrimSpace(req.Platform)),
			"architecture":     strings.ToLower(strings.TrimSpace(req.Architecture)),
			"version":          strings.TrimSpace(req.AgentVersion),
			"token_hash":       tokenHash,
			"status":           models.AgentOnline,
			"last_seen_at":     &now,
			"installed_at":     &now,
			"registration_key": nil,
		}).Error; err != nil {
			return err
		}
		machine := models.Machine{
			OrganizationID: agent.OrganizationID,
			AgentID:        agent.ID,
			Hostname:       name,
			OS:             req.OS,
			IPAddress:      req.IPAddress,
		}
		return tx.Where("agent_id = ?", agent.ID).Assign(machine).FirstOrCreate(&machine).Error
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid enrollment token"})
		return
	}

	var agent models.Agent
	h.db.Where("id = ?", *tok.AgentID).First(&agent)
	publishPresence(agent)
	c.JSON(http.StatusOK, gin.H{
		"agent_id":                 agent.ID,
		"agent_token":              agentToken,
		"poll_interval_seconds":    30,
		"heartbeat_interval_seconds": 60,
	})
}

// Heartbeat is the explicit liveness signal (agent-token auth). It updates
// last_seen_at and the reported agent version; ONLINE/OFFLINE is derived
// from recency, never set blindly. Empty/missing body is accepted (legacy
// agents post no payload).
func (h *EnrollmentHandler) Heartbeat(c *gin.Context) {
	agent := c.MustGet("agent").(*models.Agent)
	var req struct {
		AgentVersion string `json:"agent_version"`
	}
	_ = c.ShouldBindJSON(&req) // optional payload; ignore bind errors
	now := time.Now()
	updates := map[string]interface{}{
		"status":       models.AgentOnline,
		"last_seen_at": &now,
	}
	if v := strings.TrimSpace(req.AgentVersion); v != "" && len(v) <= 32 {
		updates["version"] = v
	}
	h.db.Model(agent).Updates(updates)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "server_time": now})
}

// Revoke permanently disables an agent: authentication, heartbeat, and job
// claiming are all rejected afterwards. The record is kept for audit.
func (h *EnrollmentHandler) Revoke(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var agent models.Agent
	if err := h.db.Where("id = ? AND organization_id = ?", agentID, orgID).First(&agent).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	now := time.Now()
	h.db.Model(&agent).Updates(map[string]interface{}{
		"revoked_at": &now,
		"status":     models.AgentOffline,
	})
	h.grpc.DisconnectAgent(agent.ID.String())
	publishEntity(orgID.String(), realtime.TypeAgents, "revoked", agent.ID.String())
	c.JSON(http.StatusOK, gin.H{"revoked": true})
}

// ListTokens lists enrollment tokens for the org — metadata only (expiry,
// usage, linked agent). Hashes never leave the server; the plaintext token
// exists only in the create response.
func (h *EnrollmentHandler) ListTokens(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	var toks []models.EnrollmentToken
	h.db.Where("organization_id = ?", orgID).Order("created_at DESC").Find(&toks)
	out := make([]gin.H, 0, len(toks))
	for _, t := range toks {
		out = append(out, gin.H{
			"id":              t.ID.String(),
			"organization_id": t.OrganizationID.String(),
			"agent_id":        agentIDString(t.AgentID),
			"expires_at":      t.ExpiresAt,
			"used_at":         t.UsedAt,
			"created_at":      t.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, out)
}

// RevokeToken deletes an unused enrollment token so it can never enroll.
// Used, expired, missing, or other-org tokens all answer 404 (no oracle).
func (h *EnrollmentHandler) RevokeToken(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	res := h.db.Where("id = ? AND organization_id = ? AND used_at IS NULL", id, orgID).
		Delete(&models.EnrollmentToken{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "revoke failed"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	publishEntity(orgID.String(), realtime.TypeAgents, "token-revoked", id.String())
	c.JSON(http.StatusNoContent, nil)
}

func agentIDString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

// agentWithLifecycle enriches an agent with its effective lifecycle state.
// Additive only — existing fields are untouched (backward compatible).
func agentWithLifecycle(a models.Agent) gin.H {
	return gin.H{
		"id":              a.ID.String(),
		"organization_id": a.OrganizationID.String(),
		"name":            a.Name,
		"platform":        a.Platform,
		"architecture":    a.Architecture,
		"machine_name":    a.MachineName,
		"status":          string(a.Status),
		"lifecycle":       a.Lifecycle(),
		"version":         a.Version,
		"last_seen_at":    a.LastSeenAt,
		"installed_at":    a.InstalledAt,
		"revoked_at":      a.RevokedAt,
		"created_at":      a.CreatedAt,
	}
}
