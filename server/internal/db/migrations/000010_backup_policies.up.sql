-- Backup policies + retry linkage. Additive and nullable-only: existing
-- jobs keep running on inline settings (policy_id NULL), existing runs are
-- untouched (retry columns default to "original, never retried").

CREATE TABLE IF NOT EXISTS "backup_policies" ("id" uuid,"created_at" timestamptz,"updated_at" timestamptz,"deleted_at" timestamptz,"organization_id" uuid NOT NULL,"name" text NOT NULL,"strategy" text NOT NULL DEFAULT 'FULL',"cron_expr" text DEFAULT '',"timezone" text DEFAULT 'UTC',"enabled" boolean DEFAULT true,"mode" text NOT NULL DEFAULT 'NORMAL',"encrypted" boolean DEFAULT false,"retention_days" bigint DEFAULT 30,"verification_enabled" boolean DEFAULT true,"max_retries" bigint DEFAULT 0,"retry_delay_seconds" bigint DEFAULT 300,"sla_target_minutes" bigint DEFAULT 0,"rpo_target_minutes" bigint DEFAULT 0,"rto_target_minutes" bigint DEFAULT 0,PRIMARY KEY ("id"));
CREATE INDEX IF NOT EXISTS "idx_backup_policies_organization_id" ON "backup_policies" ("organization_id");
CREATE INDEX IF NOT EXISTS "idx_backup_policies_deleted_at" ON "backup_policies" ("deleted_at");

ALTER TABLE "backup_jobs" ADD COLUMN IF NOT EXISTS "policy_id" uuid;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_backup_jobs_policy') THEN
    ALTER TABLE "backup_jobs" ADD CONSTRAINT "fk_backup_jobs_policy" FOREIGN KEY ("policy_id") REFERENCES "backup_policies"("id") ON DELETE SET NULL;
  END IF;
END $$;
CREATE INDEX IF NOT EXISTS "idx_backup_jobs_policy_id" ON "backup_jobs" ("policy_id");

ALTER TABLE "backup_runs" ADD COLUMN IF NOT EXISTS "retry_of_run_id" uuid;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_backup_runs_retry_of') THEN
    ALTER TABLE "backup_runs" ADD CONSTRAINT "fk_backup_runs_retry_of" FOREIGN KEY ("retry_of_run_id") REFERENCES "backup_runs"("id") ON DELETE SET NULL;
  END IF;
END $$;
CREATE INDEX IF NOT EXISTS "idx_backup_runs_retry_of_run_id" ON "backup_runs" ("retry_of_run_id");
ALTER TABLE "backup_runs" ADD COLUMN IF NOT EXISTS "retry_attempt" bigint DEFAULT 0;
