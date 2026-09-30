-- Rollback for 000002_run_schedule_idempotency (dev/test teardown only).

DROP INDEX IF EXISTS idx_backup_runs_job_scheduled;
ALTER TABLE backup_runs DROP COLUMN IF EXISTS scheduled_for;
