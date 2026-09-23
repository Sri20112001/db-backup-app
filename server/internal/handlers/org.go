package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/backup-saas/server/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
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

	base := slugify(req.Name)
	var org models.Organization

	// Retry loop handles the race between pre-check and insert.
	// The unique DB constraint is the true guard; we retry on conflict.
	for i := 1; i <= 10; i++ {
		slug := base
		if i > 1 {
			slug = base + "-" + strconv.Itoa(i)
		}
		org = models.Organization{
			Base:      models.Base{ID: uuid.New()},
			Name:      req.Name,
			Slug:      slug,
			UserLimit: 10,
		}
		if err := h.db.Create(&org).Error; err == nil {
			break // success
		} else if !isSlugUniqueViolation(err) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create organization"})
			return
		}
		// unique violation — try next suffix
		if i == 10 {
			c.JSON(http.StatusConflict, gin.H{"error": "could not generate a unique slug for this organization name"})
			return
		}
	}

	h.db.Create(&models.OrganizationMember{
		OrganizationID: org.ID,
		UserID:         userID,
		Role:           models.RoleOwner,
	})

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

// isSlugUniqueViolation returns true only when the PostgreSQL error is a unique
// constraint violation (23505) on the organizations slug column.
// Uses the typed pgconn.PgError for precision — no string matching.
func isSlugUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" &&
			(strings.Contains(pgErr.ConstraintName, "slug") ||
				strings.Contains(pgErr.TableName, "organization"))
	}
	return false
}
