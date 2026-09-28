package services

import (
	"fmt"
	"net/smtp"
	"strings"

	"github.com/backup-saas/server/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// Mailer sends alert emails over SMTP (stdlib only). A nil *Mailer, or one
// with an empty Host, is disabled: all sends become no-ops. Configure via
// SMTP_HOST/PORT/USER/PASSWORD/FROM; empty HOST disables email entirely.
type Mailer struct {
	host string
	port int
	user string
	pass string
	from string
}

func NewMailer(host string, port int, user, pass, from string) *Mailer {
	if host == "" {
		return nil
	}
	if port <= 0 {
		port = 587
	}
	if from == "" {
		from = user
	}
	return &Mailer{host: host, port: port, user: user, pass: pass, from: from}
}

func (m *Mailer) Enabled() bool { return m != nil && m.host != "" }

// Send delivers a plain-text email. smtp.SendMail negotiates STARTTLS when
// the server advertises it.
func (m *Mailer) Send(to []string, subject, body string) error {
	if !m.Enabled() || len(to) == 0 {
		return nil
	}
	var auth smtp.Auth
	if m.user != "" {
		auth = smtp.PlainAuth("", m.user, m.pass, m.host)
	}
	msg := "From: " + m.from + "\r\n" +
		"To: " + strings.Join(to, ", ") + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" + body
	addr := fmt.Sprintf("%s:%d", m.host, m.port)
	return smtp.SendMail(addr, auth, m.from, to, []byte(msg))
}

// NotifyOrg emails OWNER and ADMIN members of an org. Failures are logged,
// never fatal — the alert row itself is the durable record.
func NotifyOrg(db *gorm.DB, m *Mailer, orgID uuid.UUID, subject, body string) {
	if !m.Enabled() {
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
