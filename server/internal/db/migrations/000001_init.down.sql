-- Rollback for 000001_init. Drops all VaultGuard tables (data loss is
-- intentional here — down migrations exist for dev/test teardown only).

DROP TABLE IF EXISTS "refresh_tokens" CASCADE;
DROP TABLE IF EXISTS "audit_logs" CASCADE;
DROP TABLE IF EXISTS "alerts" CASCADE;
DROP TABLE IF EXISTS "backup_size_baselines" CASCADE;
DROP TABLE IF EXISTS "restore_jobs" CASCADE;
DROP TABLE IF EXISTS "backup_chunks" CASCADE;
DROP TABLE IF EXISTS "backup_artifacts" CASCADE;
DROP TABLE IF EXISTS "backup_runs" CASCADE;
DROP TABLE IF EXISTS "backup_schedules" CASCADE;
DROP TABLE IF EXISTS "backup_jobs" CASCADE;
DROP TABLE IF EXISTS "storage_targets" CASCADE;
DROP TABLE IF EXISTS "machines" CASCADE;
DROP TABLE IF EXISTS "agents" CASCADE;
DROP TABLE IF EXISTS "organization_members" CASCADE;
DROP TABLE IF EXISTS "users" CASCADE;
DROP TABLE IF EXISTS "organizations" CASCADE;
