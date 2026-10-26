DROP INDEX IF EXISTS "idx_alerts_severity";
DROP INDEX IF EXISTS "idx_alerts_status_created";
DROP INDEX IF EXISTS "idx_alerts_dedup_firing";

ALTER TABLE "alerts"
  DROP COLUMN IF EXISTS "last_occurred_at",
  DROP COLUMN IF EXISTS "resolved_at",
  DROP COLUMN IF EXISTS "acknowledged_at",
  DROP COLUMN IF EXISTS "acknowledged_by",
  DROP COLUMN IF EXISTS "occurrence_count",
  DROP COLUMN IF EXISTS "dedup_key",
  DROP COLUMN IF EXISTS "status",
  DROP COLUMN IF EXISTS "severity";
