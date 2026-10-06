ALTER TABLE "backup_runs" DROP CONSTRAINT IF EXISTS "fk_backup_runs_retry_of";
DROP INDEX IF EXISTS "idx_backup_runs_retry_of_run_id";
ALTER TABLE "backup_runs" DROP COLUMN IF EXISTS "retry_of_run_id";
ALTER TABLE "backup_runs" DROP COLUMN IF EXISTS "retry_attempt";
ALTER TABLE "backup_jobs" DROP CONSTRAINT IF EXISTS "fk_backup_jobs_policy";
DROP INDEX IF EXISTS "idx_backup_jobs_policy_id";
ALTER TABLE "backup_jobs" DROP COLUMN IF EXISTS "policy_id";
DROP TABLE IF EXISTS "backup_policies";
