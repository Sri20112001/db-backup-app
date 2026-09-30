-- S3-compatible storage + artifact provenance.
--
-- storage_targets.use_path_style: path-style addressing
-- (http://host/bucket/key, MinIO-friendly) vs virtual-hosted style
-- (AWS/R2/Wasabi). Defaults to virtual-hosted (existing behavior).
--
-- backup_artifacts.storage_target_id: pins the exact target an artifact
-- was written to, surviving later target changes/deletes. Backfilled from
-- the parent run; left NULLABLE on purpose so orphaned historical rows
-- (artifacts whose run is gone — no FK enforces this) can never block the
-- migration. The API always fills it for new artifacts.
--
-- backup_artifacts.verified_at / verification_status: HEAD/checksum
-- confirmation lifecycle PENDING → VERIFIED | FAILED (recovery testing
-- will drive these later).

ALTER TABLE storage_targets
	ADD COLUMN IF NOT EXISTS use_path_style boolean NOT NULL DEFAULT false;

ALTER TABLE backup_artifacts
	ADD COLUMN IF NOT EXISTS storage_target_id uuid;
UPDATE backup_artifacts a
	SET storage_target_id = r.storage_target_id
	FROM backup_runs r
	WHERE r.id = a.backup_run_id
	  AND a.storage_target_id IS NULL;

ALTER TABLE backup_artifacts
	ADD COLUMN IF NOT EXISTS verified_at timestamptz;
ALTER TABLE backup_artifacts
	ADD COLUMN IF NOT EXISTS verification_status varchar(32) NOT NULL DEFAULT 'PENDING';

CREATE INDEX IF NOT EXISTS idx_backup_artifacts_storage_target_id
	ON backup_artifacts (storage_target_id);
