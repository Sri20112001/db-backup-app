package middleware

import (
	"net/http"

	"github.com/backup-saas/server/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AgentAuth validates the X-Agent-ID header + Bearer token for agent-facing routes.
// On success it sets "agent" (*models.Agent) in the context.
func AgentAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := ""
		if h := c.GetHeader("Authorization"); len(h) > 7 && h[:7] == "Bearer " {
			raw = h[7:]
		}
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing agent token"})
			return
		}
		agentID, err := uuid.Parse(c.GetHeader("X-Agent-ID"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid X-Agent-ID header"})
			return
		}
		var agent models.Agent
		if err := db.Where("id = ?", agentID).First(&agent).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid agent token"})
			return
		}
		// Unregistered (pending) agents have no token hash yet.
		if agent.TokenHash == nil || bcrypt.CompareHashAndPassword([]byte(*agent.TokenHash), []byte(raw)) != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid agent token"})
			return
		}
		c.Set("agent", &agent)
		c.Next()
	}
}
