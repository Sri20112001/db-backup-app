package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/backup-saas/server/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AlertHandler struct {
	db *gorm.DB
}

func NewAlertHandler(db *gorm.DB) *AlertHandler {
	return &AlertHandler{db: db}
}

func (h *AlertHandler) List(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	query := h.db.Where("organization_id = ?", orgID)
	if c.Query("unread") == "true" {
		query = query.Where("read = false")
	}
	// Free-text search across title and message.
	if q := strings.TrimSpace(c.Query("search")); q != "" {
		like := "%" + escapeLike(q) + "%"
		query = query.Where("title ILIKE ? OR message ILIKE ?", like, like)
	}

	var total int64
	query.Model(&models.Alert{}).Count(&total)

	// Only created_at is sortable today; direction is user-controlled.
	order := "DESC"
	if strings.EqualFold(c.Query("order"), "asc") {
		order = "ASC"
	}
	query = query.Order("created_at " + order).Order("id " + order)

	var alerts []models.Alert
	query.Limit(limit).Offset(offset).Find(&alerts)

	c.JSON(http.StatusOK, gin.H{
		"data":  asArray(alerts),
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *AlertHandler) MarkRead(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	h.db.Model(&models.Alert{}).
		Where("id = ? AND organization_id = ?", id, orgID).
		Update("read", true)
	c.JSON(http.StatusOK, gin.H{"read": true})
}

// MarkAllRead flags every unread alert in the organization as read.
// Returns the number of rows changed so clients can confirm the sweep.
func (h *AlertHandler) MarkAllRead(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	res := h.db.Model(&models.Alert{}).
		Where("organization_id = ? AND read = ?", orgID, false).
		Update("read", true)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark alerts as read"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"read": true, "count": res.RowsAffected})
}
