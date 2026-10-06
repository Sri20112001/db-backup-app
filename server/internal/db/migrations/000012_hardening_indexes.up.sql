-- Production hardening: correctness constraints + indexes backed by actual
-- hot query patterns (scheduler tick, agent polling, health monitor,
-- dashboard, retention). Additive only; no data changes, no cascades.
--
-- Correctness fixes:
-- 1. Chunk idempotency key is (artifact_id, index), but 000001 created a
--    single-column UNIQUE on index alone, so chunk N of artifact A collides
--    with chunk N of artifact B. Replace with the composite key. (Safe: the
--    old global unique implies no (artifact,index) duplicates can exist.)
-- 2. Organization membership must be unique per (org, user); the app checks
--    first, but only the database closes the race.
-- 3. The restore duplicate guard is app-level (First-then-Create); a partial
--    unique on live PENDING restores closes the double-submit race. Scoped
--    to PENDING so completed restores can always be repeated.
--
-- Performance indexes (each matches a WHERE + ORDER pattern in code):
-- - runs (job, status, completed_at): scheduler in-flight guard, per-job
--   dashboard/health lookups, retention scans.
-- - runs (agent, status, created_at): agent poll/claim oldest-first.
-- - restores (agent, status, created_at): agent restore poll/claim.
-- - runs (status, updated_at): stale-run sweep every 5 minutes.
-- - agents (status, last_seen_at): offline-agent sweep every 5 minutes.
-- - alerts (type, created_at) + alerts (agent_id) + alerts (backup_job_id):
--   alert dedup checks (type + owner + window).

-- 1. Chunk composite idempotency key.
DROP INDEX IF EXISTS "idx_chunk_artifact_index";
CREATE UNIQUE INDEX IF NOT EXISTS "idx_chunk_artifact_index"
  ON "backup_chunks" ("artifact_id", "index") WHERE deleted_at IS NULL;

-- 2. Membership uniqueness (soft-delete aware like other partial uniques).
CREATE UNIQUE INDEX IF NOT EXISTS "idx_members_org_user"
  ON "organization_members" ("organization_id", "user_id") WHERE deleted_at IS NULL;

-- 3. Pending-restore duplicate guard (soft-delete aware).
CREATE UNIQUE INDEX IF NOT EXISTS "idx_restores_pending_dedupe"
  ON "restore_jobs" ("backup_run_id", "agent_id", "destination_path", "target_database")
  WHERE status = 'PENDING' AND deleted_at IS NULL;

-- Scheduler in-flight guard + per-job dashboard/retention lookups.
CREATE INDEX IF NOT EXISTS "idx_runs_job_status_completed"
  ON "backup_runs" ("backup_job_id", "status", "completed_at");

-- Agent run poll/claim (oldest PENDING first).
CREATE INDEX IF NOT EXISTS "idx_runs_agent_status_created"
  ON "backup_runs" ("agent_id", "status", "created_at");

-- Agent restore poll/claim.
CREATE INDEX IF NOT EXISTS "idx_restores_agent_status_created"
  ON "restore_jobs" ("agent_id", "status", "created_at");

-- Stale-run sweep: status IN (RUNNING, UPLOADING, VERIFYING) AND updated_at < ?.
CREATE INDEX IF NOT EXISTS "idx_runs_status_updated"
  ON "backup_runs" ("status", "updated_at");

-- Offline-agent sweep: status = ONLINE AND last_seen_at < ? (+ revoked_at IS NULL).
CREATE INDEX IF NOT EXISTS "idx_agents_status_last_seen"
  ON "agents" ("status", "last_seen_at");

-- Alert dedup windows: type + owner + recent created_at.
CREATE INDEX IF NOT EXISTS "idx_alerts_type_created"
  ON "alerts" ("type", "created_at");
CREATE INDEX IF NOT EXISTS "idx_alerts_agent_id"
  ON "alerts" ("agent_id");
CREATE INDEX IF NOT EXISTS "idx_alerts_backup_job_id"
  ON "alerts" ("backup_job_id");
