package handlers

import (
	"net/http"

	"github.com/backup-saas/server/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserHandler struct {
	db *gorm.DB
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{db: db}
}

// ListMembers returns all members of the org with their roles.
func (h *UserHandler) ListMembers(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	var members []models.OrganizationMember
	h.db.Preload("User").Where("organization_id = ?", orgID).Find(&members)
	c.JSON(http.StatusOK, members)
}

// InviteMember creates a user (if not exists) and adds them to the org.
type inviteRequest struct {
	Email string             `json:"email" binding:"required,email"`
	Name  string             `json:"name"`
	Role  models.MemberRole  `json:"role" binding:"required"`
}

func (h *UserHandler) InviteMember(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	var req inviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		// Create user with a temporary password they must reset
		tempHash, _ := bcrypt.GenerateFromPassword([]byte(randomHex(16)), bcrypt.DefaultCost)
		user = models.User{
			Base:         models.Base{ID: uuid.New()},
			Email:        req.Email,
			Name:         req.Name,
			PasswordHash: string(tempHash),
		}
		h.db.Create(&user)
	}

	// Check not already a member
	var existing int64
	h.db.Model(&models.OrganizationMember{}).
		Where("organization_id = ? AND user_id = ?", orgID, user.ID).Count(&existing)
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "user already a member"})
		return
	}

	member := models.OrganizationMember{
		OrganizationID: orgID,
		UserID:         user.ID,
		Role:           req.Role,
	}
	h.db.Create(&member)
	c.JSON(http.StatusCreated, gin.H{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    member.Role,
	})
}

// UpdateRole changes a member's role.
func (h *UserHandler) UpdateRole(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	memberID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var req struct {
		Role models.MemberRole `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := h.db.Model(&models.OrganizationMember{}).
		Where("organization_id = ? AND user_id = ?", orgID, memberID).
		Update("role", req.Role)
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "member not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"role": req.Role})
}

// RemoveMember removes a user from the org.
func (h *UserHandler) RemoveMember(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	memberID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	h.db.Where("organization_id = ? AND user_id = ?", orgID, memberID).
		Delete(&models.OrganizationMember{})
	c.JSON(http.StatusNoContent, nil)
}

// Me returns the current authenticated user's profile.
func (h *UserHandler) Me(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
	})
}
