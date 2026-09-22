package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/backup-saas/server/internal/models"
	"gorm.io/gorm"
)

// OrgContext resolves the org from the URL param and verifies membership.
func OrgContext(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, err := uuid.Parse(c.Param("org_id"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid org_id"})
			return
		}
		userID, _ := uuid.Parse(c.GetString("user_id"))

		var member models.OrganizationMember
		if err := db.Where("organization_id = ? AND user_id = ?", orgID, userID).First(&member).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		c.Set("org_id", orgID)
		c.Set("member_role", string(member.Role))
		c.Next()
	}
}

// RequireRole aborts if the member's role is not in the allowed list.
func RequireRole(roles ...models.MemberRole) gin.HandlerFunc {
	allowed := make(map[models.MemberRole]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		role := models.MemberRole(c.GetString("member_role"))
		if !allowed[role] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}
		c.Next()
	}
}
