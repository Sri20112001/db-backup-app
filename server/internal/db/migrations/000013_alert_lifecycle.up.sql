-- Alert lifecycle: severity, firing/acknowledged/resolved state, and
-- dedup-key coalescing (replaces the ad-hoc window COUNTs in the health
-- monitor and closes the duplicate-alert hole on the gRPC failure path).
--
-- Backfill: existing alerts get a severity derived from their type and are
-- marked as last-occurring at creation time. Their status stays 'firing' —
-- the honest default for alerts that were never resolved by anyone.

ALTER TABLE "alerts"
  ADD COLUMN "severity"       varchar(16)  NOT NULL DEFAULT 'warning',
  ADD COLUMN "status"         varchar(16)  NOT NULL DEFAULT 'firing',
  ADD COLUMN "dedup_key"      text,
  ADD COLUMN "occurrence_count" integer    NOT NULL DEFAULT 1,
  ADD COLUMN "acknowledged_by"  uuid,
  ADD COLUMN "acknowledged_at"  timestamptz,
  ADD COLUMN "resolved_at"      timestamptz,
  ADD COLUMN "last_occurred_at" timestamptz;

UPDATE "alerts" SET "severity" = CASE
  WHEN "type" IN ('BACKUP_FAILED', 'BACKUP_MISSED', 'AGENT_OFFLINE', 'RESTORE_FAILED')
    THEN 'critical'
  ELSE 'warning'
END;
UPDATE "alerts" SET "last_occurred_at" = "created_at" WHERE "last_occurred_at" IS NULL;

-- At most one FIRING alert per (org, dedup_key): re-fires after a resolve
-- create a fresh row; repeats while still firing coalesce via occurrence_count.
CREATE UNIQUE INDEX "idx_alerts_dedup_firing"
  ON "alerts" ("organization_id", "dedup_key")
  WHERE "dedup_key" IS NOT NULL AND "status" = 'firing' AND "deleted_at" IS NULL;

-- Status filter on the alerts page + retention sweep (resolved older than N days).
CREATE INDEX "idx_alerts_status_created" ON "alerts" ("status", "created_at");
CREATE INDEX "idx_alerts_severity"       ON "alerts" ("severity");
