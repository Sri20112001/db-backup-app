-- Artifact registration idempotency: (backup_run_id, name) is unique so a
-- retried RegisterArtifact call (network timeout + client retry after the
-- server already committed) returns the existing row instead of duplicating
-- recovery-point records. Chunks already had this protection.
--
-- Dedupe first: keep the earliest row per (run, name) so the index builds
-- even on databases that already contain timeout duplicates.

DELETE FROM "backup_artifacts" a USING "backup_artifacts" b
WHERE a."id" > b."id"
  AND a."backup_run_id" = b."backup_run_id"
  AND a."name" = b."name"
  AND a."deleted_at" IS NULL
  AND b."deleted_at" IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS "idx_artifacts_run_name" ON "backup_artifacts" ("backup_run_id", "name") WHERE deleted_at IS NULL;
