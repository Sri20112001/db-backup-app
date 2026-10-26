package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/backup-saas/server/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// NotificationSettingsHandler manages per-org alert routing (webhook, quiet
// hours, severity floors, monitor thresholds) and per-user email prefs.
type NotificationSettingsHandler struct {
	db     *gorm.DB
	mailer *services.Mailer
}

func NewNotificationSettingsHandler(db *gorm.DB, mailer *services.Mailer) *NotificationSettingsHandler {
	return &NotificationSettingsHandler{db: db, mailer: mailer}
}

// defaultSettings fills the zero-value defaults so a brand-new org can be
// rendered in the UI without a DB row yet (rows are created on first PUT).
func defaultSettings(orgID uuid.UUID) models.NotificationSettings {
	return models.NotificationSettings{
		OrganizationID:      orgID,
		QuietHoursStart:     -1,
		QuietHoursEnd:       -1,
		MinEmailSeverity:    models.SeverityWarning,
		OfflineThresholdMin: 3,
		MissedWindowMult:    2,
	}
}

// Get returns the org's notification settings, or defaults when none exist.
// Open to every member (read-only visibility); writes are Owner/Admin only.
func (h *NotificationSettingsHandler) Get(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	var s models.NotificationSettings
	if err := h.db.Where("organization_id = ?", orgID).First(&s).Error; err != nil {
		c.JSON(http.StatusOK, defaultSettings(orgID))
		return
	}
	c.JSON(http.StatusOK, s)
}

// UpdateRequest is the wire shape for PUT. Pointers distinguish "absent"
// (leave unchanged) from "present zero value" where it matters.
type UpdateNotificationSettingsRequest struct {
	WebhookURL          *string               `json:"webhook_url"`
	WebhookEnabled      *bool                 `json:"webhook_enabled"`
	WebhookSecret       *string               `json:"webhook_secret"` // empty string = keep existing
	QuietHoursStart     *int                  `json:"quiet_hours_start"`
	QuietHoursEnd       *int                  `json:"quiet_hours_end"`
	MinEmailSeverity    *models.AlertSeverity `json:"min_email_severity"`
	OfflineThresholdMin *int                  `json:"offline_threshold_minutes"`
	MissedWindowMult    *int                  `json:"missed_window_multiplier"`
}

// Update upserts the org's settings. Owner/Admin only (enforced in router).
func (h *NotificationSettingsHandler) Update(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)

	var req UpdateNotificationSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	current := defaultSettings(orgID)
	exists := true
	if err := h.db.Where("organization_id = ?", orgID).First(&current).Error; err != nil {
		exists = false
	}

	if req.WebhookURL != nil {
		current.WebhookURL = strings.TrimSpace(*req.WebhookURL)
	}
	if req.WebhookEnabled != nil {
		current.WebhookEnabled = *req.WebhookEnabled
	}
	if req.WebhookSecret != nil {
		// Empty means "don't change": the secret is write-only and never
		// echoed back, so a round-trip of the GET response must not wipe it.
		if trimmed := strings.TrimSpace(*req.WebhookSecret); trimmed != "" {
			current.WebhookSecret = trimmed
		}
	}
	if req.QuietHoursStart != nil {
		current.QuietHoursStart = *req.QuietHoursStart
	}
	if req.QuietHoursEnd != nil {
		current.QuietHoursEnd = *req.QuietHoursEnd
	}
	if req.MinEmailSeverity != nil {
		if !validSeverity(*req.MinEmailSeverity) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "min_email_severity must be info, warning, or critical"})
			return
		}
		current.MinEmailSeverity = *req.MinEmailSeverity
	}
	if req.OfflineThresholdMin != nil {
		if *req.OfflineThresholdMin < 1 || *req.OfflineThresholdMin > 60 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "offline_threshold_minutes must be between 1 and 60"})
			return
		}
		current.OfflineThresholdMin = *req.OfflineThresholdMin
	}
	if req.MissedWindowMult != nil {
		if *req.MissedWindowMult < 1 || *req.MissedWindowMult > 24 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missed_window_multiplier must be between 1 and 24"})
			return
		}
		current.MissedWindowMult = *req.MissedWindowMult
	}

	if exists {
		// Save (not Updates-with-struct): GORM's struct Updates skips zero
		// values, which would silently ignore WebhookEnabled=false and the
		// -1 quiet-hours sentinels.
		current.UpdatedAt = time.Now()
		if err := h.db.Save(&current).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update notification settings"})
			return
		}
	} else {
		if err := h.db.Create(&current).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save notification settings"})
			return
		}
	}

	// Re-read to return canonical values (e.g. server-side trimming).
	var saved models.NotificationSettings
	h.db.Where("organization_id = ?", orgID).First(&saved)
	c.JSON(http.StatusOK, saved)
}

// TestEmail sends a sample notification to the requesting admin's own
// address, bypassing severity floors and quiet hours (it's a delivery probe,
// not a real alert). SMTP misconfiguration surfaces here, not on first alert.
func (h *NotificationSettingsHandler) TestEmail(c *gin.Context) {
	orgID := c.MustGet("org_id").(uuid.UUID)
	userID := uuid.MustParse(c.GetString("user_id"))

	if !h.mailer.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "email is not configured on this server"})
		return
	}

	var user models.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load user"})
		return
	}
	email := strings.TrimSpace(user.Email)
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "your account has no email address"})
		return
	}

	var org models.Organization
	orgName := "your organization"
	if err := h.db.First(&org, "id = ?", orgID).Error; err == nil {
		orgName = org.Name
	}

	subject, body := services.FormatTestEmail(orgName)
	if err := h.mailer.Send([]string{email}, subject, body); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to send test email: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sent": true, "to": email})
}

// --- Per-user preferences ---

// GetMyPrefs returns the requesting user's email prefs (defaults if none).
func (h *NotificationSettingsHandler) GetMyPrefs(c *gin.Context) {
	userID := uuid.MustParse(c.GetString("user_id"))
	var prefs models.UserNotificationPrefs
	if err := h.db.Where("user_id = ?", userID).First(&prefs).Error; err != nil {
		c.JSON(http.StatusOK, models.UserNotificationPrefs{
			UserID:       userID,
			EmailEnabled: true,
			MinSeverity:  models.SeverityInfo,
		})
		return
	}
	c.JSON(http.StatusOK, prefs)
}

type UpdateUserNotificationPrefsRequest struct {
	EmailEnabled *bool                 `json:"email_enabled"`
	MinSeverity  *models.AlertSeverity `json:"min_severity"`
}

// UpdateMyPrefs upserts the requesting user's email prefs. Any authed member
// may change their own prefs — no role gate.
func (h *NotificationSettingsHandler) UpdateMyPrefs(c *gin.Context) {
	userID := uuid.MustParse(c.GetString("user_id"))

	var req UpdateUserNotificationPrefsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	var prefs models.UserNotificationPrefs
	err := h.db.Where("user_id = ?", userID).First(&prefs).Error
	if err != nil {
		prefs = models.UserNotificationPrefs{
			UserID:       userID,
			EmailEnabled: true,
			MinSeverity:  models.SeverityInfo,
		}
	}

	if req.EmailEnabled != nil {
		prefs.EmailEnabled = *req.EmailEnabled
	}
	if req.MinSeverity != nil {
		if !validSeverity(*req.MinSeverity) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "min_severity must be info, warning, or critical"})
			return
		}
		prefs.MinSeverity = *req.MinSeverity
	}
	prefs.UpdatedAt = time.Now()

	if err := h.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		UpdateAll: true,
	}).Create(&prefs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save preferences"})
		return
	}
	c.JSON(http.StatusOK, prefs)
}

func validSeverity(s models.AlertSeverity) bool {
	switch s {
	case models.SeverityInfo, models.SeverityWarning, models.SeverityCritical:
		return true
	}
	return false
}
