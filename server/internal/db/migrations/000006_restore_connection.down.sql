ALTER TABLE "restore_jobs" DROP CONSTRAINT IF EXISTS "fk_restore_jobs_connection";
DROP INDEX IF EXISTS "idx_restore_jobs_connection_id";
ALTER TABLE "restore_jobs" DROP COLUMN IF EXISTS "connection_id";
