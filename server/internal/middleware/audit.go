package middleware

import (
	"github.com/backup-saas/server/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Audit writes an AuditLog entry for mutating requests (POST, PUT, DELETE, PATCH).
func Audit(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		method := c.Request.Method
		if method != "POST" && method != "PUT" && method != "DELETE" && method != "PATCH" {
			return
		}

		userIDStr := c.GetString("user_id")
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return
		}

		orgID, _ := c.Get("org_id")
		orgUUID, _ := orgID.(uuid.UUID)

		db.Create(&models.AuditLog{
			OrganizationID: orgUUID,
			UserID:         userID,
			Action:         method,
			Resource:       c.FullPath(),
			ResourceID:     c.Param("id"),
			IPAddress:      c.ClientIP(),
		})
	}
}
