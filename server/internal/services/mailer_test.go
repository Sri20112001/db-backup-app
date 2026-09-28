package services

import "testing"

func TestMailerDisabled(t *testing.T) {
	m := NewMailer("", 587, "", "", "")
	if m.Enabled() {
		t.Error("mailer with empty host must be disabled")
	}
	var nilMailer *Mailer
	if nilMailer.Enabled() {
		t.Error("nil mailer must be disabled")
	}
	// Disabled sends are silent no-ops, never errors.
	if err := m.Send([]string{"a@x.io"}, "s", "b"); err != nil {
		t.Errorf("disabled send: %v", err)
	}
	if err := nilMailer.Send([]string{"a@x.io"}, "s", "b"); err != nil {
		t.Errorf("nil send: %v", err)
	}
}

func TestMailerDefaults(t *testing.T) {
	m := NewMailer("smtp.example.com", 0, "user", "pass", "")
	if !m.Enabled() {
		t.Error("mailer with host must be enabled")
	}
	if m.port != 587 {
		t.Errorf("default port = %d, want 587", m.port)
	}
	if m.from != "user" {
		t.Errorf("from defaults to user, got %q", m.from)
	}
}
