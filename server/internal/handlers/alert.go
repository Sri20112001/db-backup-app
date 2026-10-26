package handlers

import (
	"net/http"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/backup-saas/server/internal/realtime"
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
	p := parseListParams(c)

	query := h.db.Model(&models.Alert{}).Where("alerts.organization_id = ?", orgID)
	if c.Query("unread") == "true" {
		query = query.Where("read = false")
	}
	if alertType := c.Query("type"); alertType != "" {
		query = query.Where("alerts.type = ?", alertType)
	}
	// Lifecycle + severity filters. Empty "all" means no constraint.
	if status := c.Query("status"); status != "" && status != "all" {
		query = query.Where("alerts.status = ?", models.AlertStatus(status))
	}
	if sev := c.Query("severity"); sev != "" && sev != "all" {
		query = query.Where("alerts.severity = ?", models.AlertSeverity(sev))
	}
	// Free-text search across title and message.
	if p.Search != "" {
		like := "%" + escapeLike(p.Search) + "%"
		query = query.Where("title ILIKE ? OR message ILIKE ?", like, like)
	}

	var total int64
	query.Count(&total)

	sortCol := p.sortColumn(c, map[string]string{
		"created":   "alerts.created_at",
		"type":      "alerts.type",
		"title":     "alerts.title",
		"severity":  "alerts.severity",
		"status":    "alerts.status",
		"occurred":  "alerts.last_occurred_at",
	}, "alerts.created_at")
	query = query.Order(sortCol + " " + p.Order).Order("id " + p.Order)

	var alerts []models.Alert
	query.Limit(p.Limit).Offset(p.Offset).Find(&alerts)

	respondPaginated(c, asArray(alerts), total, p)
}

func (h *AlertHandler) MarkRead(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	res := h.db.Model(&models.Alert{}).
		Where("id = ? AND organization_id = ?", id, orgID).
		Update("read", true)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark alert as read"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"read": true})
}

// Ack records that a human owns the alert. Only FIRING alerts transition;
// already-acknowledged/resolved rows 409 so double-clicks are visible, not silent.
func (h *AlertHandler) Ack(c *gin.Context) {
	h.transition(c, func(db *gorm.DB, id, orgID uuid.UUID, userID uuid.UUID) *gorm.DB {
		return db.Model(&models.Alert{}).
			Where("id = ? AND organization_id = ? AND status = ?", id, orgID, models.AlertFiring).
			Updates(map[string]interface{}{
				"status":          models.AlertAcked,
				"acknowledged_by": userID,
				"acknowledged_at": time.Now(),
			})
	})
}

// Resolve closes the alert regardless of prior state (acknowledged or firing).
func (h *AlertHandler) Resolve(c *gin.Context) {
	h.transition(c, func(db *gorm.DB, id, orgID uuid.UUID, _ uuid.UUID) *gorm.DB {
		return db.Model(&models.Alert{}).
			Where("id = ? AND organization_id = ? AND status IN ?", id, orgID,
				[]models.AlertStatus{models.AlertFiring, models.AlertAcked}).
			Updates(map[string]interface{}{
				"status":      models.AlertResolved,
				"resolved_at": time.Now(),
			})
	})
}

// transition is shared plumbing for lifecycle POSTs: parse ids, run the
// scoped update, then publish the flip so live dashboards react immediately.
func (h *AlertHandler) transition(
	c *gin.Context,
	update func(db *gorm.DB, id, orgID, userID uuid.UUID) *gorm.DB,
) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	userID := uuid.MustParse(c.GetString("user_id"))
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	res := update(h.db, id, orgID, userID)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update alert"})
		return
	}
	if res.RowsAffected == 0 {
		// Either not found or already past the allowed source state.
		c.JSON(http.StatusConflict, gin.H{"error": "alert not found or not in a valid state"})
		return
	}
	var alert models.Alert
	h.db.Where("id = ? AND organization_id = ?", id, orgID).First(&alert)
	realtime.Publish(realtime.DefaultHub, orgID.String(), realtime.Event{
		Type:    realtime.TypeAlert,
		Payload: alert,
	})
	c.JSON(http.StatusOK, alert)
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
