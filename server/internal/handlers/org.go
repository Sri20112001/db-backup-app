package handlers

import (
	"net/http"
	"strings"

	"github.com/backup-saas/server/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrgHandler struct {
	db *gorm.DB
}

func NewOrgHandler(db *gorm.DB) *OrgHandler {
	return &OrgHandler{db: db}
}

func (h *OrgHandler) List(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	var members []models.OrganizationMember
	h.db.Preload("Organization").Where("user_id = ?", userID).Find(&members)

	orgs := make([]gin.H, 0, len(members))
	for _, m := range members {
		orgs = append(orgs, gin.H{
			"id":   m.Organization.ID,
			"name": m.Organization.Name,
			"slug": m.Organization.Slug,
			"role": m.Role,
		})
	}
	c.JSON(http.StatusOK, orgs)
}

type createOrgRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *OrgHandler) Create(c *gin.Context) {
	var req createOrgRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	org := models.Organization{
		Base: models.Base{ID: uuid.New()},
		Name: req.Name,
		Slug: slugify(req.Name),
	}
	if err := h.db.Create(&org).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "organization name already taken"})
		return
	}

	member := models.OrganizationMember{
		OrganizationID: org.ID,
		UserID:         userID,
		Role:           models.RoleOwner,
	}
	h.db.Create(&member)

	c.JSON(http.StatusCreated, org)
}

func (h *OrgHandler) Get(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	var org models.Organization
	if err := h.db.First(&org, orgID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, org)
}

func slugify(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), " ", "-"))
}
