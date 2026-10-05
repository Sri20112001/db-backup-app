ALTER TABLE "backup_jobs" DROP CONSTRAINT IF EXISTS "fk_backup_jobs_connection";
DROP INDEX IF EXISTS "idx_backup_jobs_connection_id";
ALTER TABLE "backup_jobs" DROP COLUMN IF EXISTS "connection_id";
DROP TABLE IF EXISTS "database_connections";
