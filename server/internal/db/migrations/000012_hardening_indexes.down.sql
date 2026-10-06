-- Roll back 000012: drop hardening indexes and restore the 000001 chunk
-- index shape. Data is untouched in both directions.
DROP INDEX IF EXISTS "idx_alerts_backup_job_id";
DROP INDEX IF EXISTS "idx_alerts_agent_id";
DROP INDEX IF EXISTS "idx_alerts_type_created";
DROP INDEX IF EXISTS "idx_agents_status_last_seen";
DROP INDEX IF EXISTS "idx_runs_status_updated";
DROP INDEX IF EXISTS "idx_restores_agent_status_created";
DROP INDEX IF EXISTS "idx_runs_agent_status_created";
DROP INDEX IF EXISTS "idx_runs_job_status_completed";
DROP INDEX IF EXISTS "idx_restores_pending_dedupe";
DROP INDEX IF EXISTS "idx_members_org_user";
DROP INDEX IF EXISTS "idx_chunk_artifact_index";
CREATE UNIQUE INDEX IF NOT EXISTS "idx_chunk_artifact_index" ON "backup_chunks" ("index");
