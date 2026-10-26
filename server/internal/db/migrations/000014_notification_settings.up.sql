-- Notification routing: per-org webhook + quiet hours + email severity floor
-- + monitor thresholds, and per-user email preferences. SMTP transport stays
-- process-env (one relay per deployment); these tables only shape who gets
-- what, when, and through which extra channels.

CREATE TABLE "notification_settings" (
  "organization_id"          uuid PRIMARY KEY REFERENCES "organizations"("id") ON DELETE CASCADE,
  "webhook_url"              text        NOT NULL DEFAULT '',
  "webhook_enabled"          boolean     NOT NULL DEFAULT false,
  "webhook_secret"           text        NOT NULL DEFAULT '',
  "quiet_hours_start"        integer     NOT NULL DEFAULT -1,
  "quiet_hours_end"          integer     NOT NULL DEFAULT -1,
  "min_email_severity"       varchar(16) NOT NULL DEFAULT 'warning',
  "offline_threshold_minutes" integer    NOT NULL DEFAULT 3,
  "missed_window_multiplier" integer     NOT NULL DEFAULT 2,
  "updated_at"               timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE "user_notification_prefs" (
  "user_id"       uuid PRIMARY KEY REFERENCES "users"("id") ON DELETE CASCADE,
  "email_enabled" boolean     NOT NULL DEFAULT true,
  "min_severity"  varchar(16) NOT NULL DEFAULT 'info',
  "updated_at"    timestamptz NOT NULL DEFAULT now()
);
