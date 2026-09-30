-- Scheduler idempotency: a UNIQUE(backup_job_id, scheduled_for) guard so
-- two racing scheduler ticks (multi-instance deploys, restarts) can never
-- insert the same scheduled run twice. Ad-hoc runs keep scheduled_for NULL,
-- and NULLs never collide in a unique index.

ALTER TABLE backup_runs ADD COLUMN IF NOT EXISTS scheduled_for timestamptz;
CREATE UNIQUE INDEX IF NOT EXISTS idx_backup_runs_job_scheduled
	ON backup_runs (backup_job_id, scheduled_for);
