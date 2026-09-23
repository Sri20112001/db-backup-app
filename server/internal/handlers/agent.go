package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// StreamDisconnector is satisfied by grpcserver.Server.
type StreamDisconnector interface {
	DisconnectAgent(agentID string)
}

type AgentHandler struct {
	db   *gorm.DB
	grpc StreamDisconnector
}

func NewAgentHandler(db *gorm.DB, grpc StreamDisconnector) *AgentHandler {
	return &AgentHandler{db: db, grpc: grpc}
}

func (h *AgentHandler) List(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	var agents []models.Agent
	h.db.Where("organization_id = ?", orgID).Find(&agents)
	c.JSON(http.StatusOK, asArray(agents))
}

// GenerateRegistrationToken creates a one-time token the agent uses to register.
func (h *AgentHandler) GenerateRegistrationToken(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)

	token := randomHex(32)
	agent := models.Agent{
		Base:            models.Base{ID: uuid.New()},
		OrganizationID:  orgID,
		Name:            "pending-" + randomHex(4),
		RegistrationKey: &token,
		Status:          models.AgentOffline,
	}
	if err := h.db.Create(&agent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create agent"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"agent_id":         agent.ID,
		"registration_key": token,
	})
}

// Register is called by the agent binary with the registration key.
func (h *AgentHandler) Register(c *gin.Context) {
	var req struct {
		RegistrationKey string `json:"registration_key" binding:"required"`
		Hostname        string `json:"hostname" binding:"required"`
		OS              string `json:"os"`
		Version         string `json:"version"`
		IPAddress       string `json:"ip_address"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var agent models.Agent
	if err := h.db.Where("registration_key = ?", req.RegistrationKey).First(&agent).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid registration key"})
		return
	}

	agentToken := randomHex(32)
	tokenHash, err := bcrypt.GenerateFromPassword([]byte(agentToken), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}
	updates := map[string]interface{}{
		"name":             req.Hostname,
		"token_hash":       string(tokenHash),
		"version":          req.Version,
		"status":           models.AgentOnline,
		"registration_key": nil,
	}
	h.db.Model(&agent).Updates(updates)

	// Create or update machine record
	machine := models.Machine{
		OrganizationID: agent.OrganizationID,
		AgentID:        agent.ID,
		Hostname:       req.Hostname,
		OS:             req.OS,
		IPAddress:      req.IPAddress,
	}
	h.db.Where("agent_id = ?", agent.ID).Assign(machine).FirstOrCreate(&machine)

	c.JSON(http.StatusOK, gin.H{
		"agent_id":    agent.ID,
		"agent_token": agentToken,
	})

	// Audit: agent registered
	userID, _ := uuid.Parse(c.GetString("user_id"))
	h.db.Create(&models.AuditLog{
		OrganizationID: agent.OrganizationID,
		UserID:         userID,
		Action:         "AGENT_REGISTERED",
		Resource:       "agent",
		ResourceID:     agent.ID.String(),
		IPAddress:      c.ClientIP(),
	})
}

func (h *AgentHandler) Get(c *gin.Context) {
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
	c.JSON(http.StatusOK, agent)
}

func (h *AgentHandler) Delete(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.db.Where("id = ? AND organization_id = ?", agentID, orgID).Delete(&models.Agent{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

// RotateToken issues a new agent token, immediately invalidating the old one.
// Uses SELECT FOR UPDATE to serialize concurrent rotation requests.
// Disconnects any active gRPC stream so the old token cannot keep a stream alive.
func (h *AgentHandler) RotateToken(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	agentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	newToken := randomHex(32)
	tokenHash, err := bcrypt.GenerateFromPassword([]byte(newToken), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	var agent models.Agent
	txErr := h.db.Transaction(func(tx *gorm.DB) error {
		// Lock the row to serialize concurrent rotation requests
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND organization_id = ?", agentID, orgID).
			First(&agent).Error; err != nil {
			return err
		}
		now := time.Now()
		return tx.Model(&agent).Updates(map[string]interface{}{
			"token_hash":       string(tokenHash),
			"token_rotated_at": &now,
		}).Error
	})
	if txErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	// Disconnect any active gRPC stream — old token must not keep a stream alive
	h.grpc.DisconnectAgent(agent.ID.String())

	userID, _ := uuid.Parse(c.GetString("user_id"))
	h.db.Create(&models.AuditLog{
		OrganizationID: orgID,
		UserID:         userID,
		Action:         "AGENT_TOKEN_ROTATED",
		Resource:       "agent",
		ResourceID:     agent.ID.String(),
		IPAddress:      c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{
		"agent_id":    agent.ID,
		"agent_token": newToken, // shown once, never stored
	})
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}
