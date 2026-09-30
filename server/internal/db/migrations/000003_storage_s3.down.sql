-- Rollback for 000003_storage_s3 (dev/test teardown only).

DROP INDEX IF EXISTS idx_backup_artifacts_storage_target_id;
ALTER TABLE backup_artifacts DROP COLUMN IF EXISTS verification_status;
ALTER TABLE backup_artifacts DROP COLUMN IF EXISTS verified_at;
ALTER TABLE backup_artifacts DROP COLUMN IF EXISTS storage_target_id;
ALTER TABLE storage_targets DROP COLUMN IF EXISTS use_path_style;
