package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/backup-saas/server/internal/models"
	"github.com/backup-saas/server/internal/realtime"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// AlertInput describes an alert to raise. DedupKey, when non-empty, makes
// repeat occurrences coalesce into the existing FIRING row (occurrence
// counting) instead of creating noise. Email + webhook fire only on new
// rows so a flapping condition cannot spam channels.
type AlertInput struct {
	OrgID        uuid.UUID
	Type         models.AlertType
	Title        string
	Message      string
	BackupJobID  *uuid.UUID
	AgentID      *uuid.UUID
	DedupKey     string // empty = always create a new row
	Severity     models.AlertSeverity // empty = derived from Type
	EmailSubject string               // empty = no email
	EmailBody    string
}

// RaiseAlert creates a FIRING alert, or coalesces into an existing firing
// alert with the same (org, dedup_key). A nil mailer is fine (email off).
// Returns the alert and whether a new row was created.
func RaiseAlert(db *gorm.DB, m *Mailer, in AlertInput) (*models.Alert, bool) {
	severity := in.Severity
	if severity == "" {
		severity = models.DefaultSeverity(in.Type)
	}
	now := time.Now()

	if in.DedupKey != "" {
		if existing, ok := coalesceFiring(db, in.OrgID, in.DedupKey, in.Message, now); ok {
			return existing, false
		}
	}

	key := in.DedupKey
	alert := models.Alert{
		OrganizationID:  in.OrgID,
		Type:            in.Type,
		Severity:        severity,
		Status:          models.AlertFiring,
		Title:           in.Title,
		Message:         in.Message,
		BackupJobID:     in.BackupJobID,
		AgentID:         in.AgentID,
		OccurrenceCount: 1,
		LastOccurredAt:  &now,
	}
	if key != "" {
		alert.DedupKey = &key
	}
	if err := db.Create(&alert).Error; err != nil {
		// Unique violation on the firing-dedup index: a concurrent raise won
		// the insert — coalesce into it instead of failing the caller.
		if existing, ok := coalesceFiring(db, in.OrgID, key, in.Message, now); ok && key != "" {
			return existing, false
		}
		log.Warn().Err(err).Str("org_id", in.OrgID.String()).
			Str("type", string(in.Type)).Msg("alerts: create failed")
		return nil, false
	}
	publishAlert(alert)
	if in.EmailSubject != "" {
		NotifyAlert(db, m, in.OrgID, severity, in.EmailSubject, in.EmailBody)
	}
	dispatchWebhook(db, in.OrgID, &alert)
	return &alert, true
}

// coalesceFiring bumps occurrence_count/last_occurred_at on the org's
// currently-firing alert with the given dedup key, refreshing the message.
func coalesceFiring(db *gorm.DB, orgID uuid.UUID, key, message string, now time.Time) (*models.Alert, bool) {
	if key == "" {
		return nil, false
	}
	var existing models.Alert
	err := db.Where("organization_id = ? AND dedup_key = ? AND status = ?",
		orgID, key, models.AlertFiring).First(&existing).Error
	if err != nil {
		return nil, false
	}
	updates := map[string]interface{}{
		"occurrence_count": gorm.Expr("occurrence_count + 1"),
		"last_occurred_at": now,
	}
	if message != "" {
		updates["message"] = message
	}
	if err := db.Model(&existing).Updates(updates).Error; err != nil {
		log.Warn().Err(err).Str("dedup_key", key).Msg("alerts: coalesce update failed")
		return nil, false
	}
	db.Model(&existing).Select("occurrence_count", "last_occurred_at", "message").Find(&existing)
	publishAlertUpdate(existing)
	return &existing, true
}

// ResolveAlerts resolves every firing alert matching (org, type, dedup key)
// and broadcasts the change. Returns how many were resolved.
func ResolveAlerts(db *gorm.DB, orgID uuid.UUID, typ models.AlertType, dedupKey string) int {
	if dedupKey == "" {
		return 0
	}
	var firing []models.Alert
	db.Where("organization_id = ? AND type = ? AND dedup_key = ? AND status = ?",
		orgID, typ, dedupKey, models.AlertFiring).Find(&firing)
	if len(firing) == 0 {
		return 0
	}
	now := time.Now()
	ids := make([]uuid.UUID, len(firing))
	for i, a := range firing {
		ids[i] = a.ID
	}
	if err := db.Model(&models.Alert{}).Where("id IN ?", ids).
		Updates(map[string]interface{}{"status": models.AlertResolved, "resolved_at": &now}).Error; err != nil {
		log.Warn().Err(err).Str("dedup_key", dedupKey).Msg("alerts: resolve failed")
		return 0
	}
	for _, a := range firing {
		a.Status = models.AlertResolved
		a.ResolvedAt = &now
		publishAlertUpdate(a)
	}
	return len(firing)
}

// NotifyAlert emails the org's OWNER/ADMIN members, honoring the org's
// notification settings (severity floor, quiet hours) and each user's prefs.
// Quiet hours suppress non-critical email only. Failures are logged, never
// fatal — the alert row itself is the durable record.
func NotifyAlert(db *gorm.DB, m *Mailer, orgID uuid.UUID, severity models.AlertSeverity, subject, body string) {
	if !m.Enabled() {
		return
	}
	var s models.NotificationSettings
	if err := db.First(&s, "organization_id = ?", orgID).Error; err != nil {
		s = models.NotificationSettings{
			OrganizationID:   orgID,
			QuietHoursStart:  -1,
			QuietHoursEnd:    -1,
			MinEmailSeverity: models.SeverityWarning,
		}
	}
	if models.SeverityRank(severity) < models.SeverityRank(s.MinEmailSeverity) {
		return
	}
	if inQuietHours(s, time.Now().UTC()) && severity != models.SeverityCritical {
		return
	}

	var members []models.OrganizationMember
	db.Preload("User").
		Where("organization_id = ? AND role IN ?", orgID,
			[]models.MemberRole{models.RoleOwner, models.RoleAdmin}).
		Find(&members)
	var to []string
	seen := map[string]bool{}
	for _, mem := range members {
		email := strings.TrimSpace(mem.User.Email)
		if email == "" || seen[email] {
			continue
		}
		var prefs models.UserNotificationPrefs
		if err := db.First(&prefs, "user_id = ?", mem.UserID).Error; err == nil {
			if !prefs.EmailEnabled {
				continue
			}
			if models.SeverityRank(severity) < models.SeverityRank(prefs.MinSeverity) {
				continue
			}
		}
		seen[email] = true
		to = append(to, email)
	}
	if len(to) == 0 {
		return
	}
	if err := m.Send(to, subject, body); err != nil {
		log.Warn().Err(err).Str("org_id", orgID.String()).Msg("mailer: send failed")
	}
}

// inQuietHours reports whether `at` (UTC) falls in the org's quiet-hours
// window. -1 on either end disables the feature; equal ends disable it too.
// A start after end wraps midnight (e.g. 22 → 6).
func inQuietHours(s models.NotificationSettings, at time.Time) bool {
	if s.QuietHoursStart < 0 || s.QuietHoursEnd < 0 || s.QuietHoursStart == s.QuietHoursEnd {
		return false
	}
	h := at.Hour()
	if s.QuietHoursStart < s.QuietHoursEnd {
		return h >= s.QuietHoursStart && h < s.QuietHoursEnd
	}
	return h >= s.QuietHoursStart || h < s.QuietHoursEnd
}

// dispatchWebhook POSTs the alert as JSON to the org's webhook URL (when
// enabled), signed with HMAC-SHA256 when a secret is set. Fire-and-forget:
// delivery failure must never block the request that raised the alert.
func dispatchWebhook(db *gorm.DB, orgID uuid.UUID, alert *models.Alert) {
	var s models.NotificationSettings
	if err := db.First(&s, "organization_id = ?", orgID).Error; err != nil {
		return
	}
	if !s.WebhookEnabled || s.WebhookURL == "" {
		return
	}
	if u, err := url.Parse(s.WebhookURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		log.Warn().Str("org_id", orgID.String()).Msg("webhook: invalid URL, skipping")
		return
	}
	payload, err := json.Marshal(map[string]interface{}{
		"event": "alert.raised",
		"alert": alert,
	})
	if err != nil {
		return
	}
	endpoint, secret := s.WebhookURL, s.WebhookSecret
	go func() {
		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			log.Warn().Err(err).Str("org_id", orgID.String()).Msg("webhook: build request failed")
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "VaultGuard-Alerting")
		if secret != "" {
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write(payload)
			req.Header.Set("X-VaultGuard-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
		}
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Warn().Err(err).Str("org_id", orgID.String()).Msg("webhook: delivery failed")
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			log.Warn().Str("org_id", orgID.String()).Int("status", resp.StatusCode).
				Msg("webhook: non-2xx response")
		}
	}()
}

// publishAlert broadcasts a freshly created alert row (nil-hub safe).
func publishAlert(alert models.Alert) {
	realtime.Publish(realtime.DefaultHub, alert.OrganizationID.String(), realtime.Event{
		Type:    realtime.TypeAlert,
		Payload: alert,
	})
}

// publishAlertUpdate broadcasts a lifecycle change (coalesce/resolve/ack).
func publishAlertUpdate(alert models.Alert) {
	realtime.Publish(realtime.DefaultHub, alert.OrganizationID.String(), realtime.Event{
		Type:    realtime.TypeAlert,
		Payload: alert,
	})
}

// FormatTestEmail renders the body of the settings-page test email.
func FormatTestEmail(orgName string) (string, string) {
	subject := "[VaultGuard] Test notification"
	body := fmt.Sprintf("This is a test notification for %s.\n\n"+
		"If you received this, SMTP delivery is working.", orgName)
	return subject, body
}
