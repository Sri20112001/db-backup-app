-- VaultGuard baseline schema. Generated from the GORM models (see
-- cmd/dumpddl, since removed). Every statement is idempotent
-- (IF NOT EXISTS), so databases previously created by AutoMigrate adopt
-- version 1 cleanly without data loss. Future changes go in NEW versioned
-- files — never edit this baseline after it has run anywhere.

CREATE TABLE IF NOT EXISTS "organizations" ("id" uuid,"created_at" timestamptz,"updated_at" timestamptz,"deleted_at" timestamptz,"name" text NOT NULL,"slug" text NOT NULL,"user_limit" bigint NOT NULL DEFAULT 10,PRIMARY KEY ("id"));
CREATE UNIQUE INDEX IF NOT EXISTS "idx_organizations_slug" ON "organizations" ("slug");
CREATE INDEX IF NOT EXISTS "idx_organizations_deleted_at" ON "organizations" ("deleted_at");

CREATE TABLE IF NOT EXISTS "users" ("id" uuid,"created_at" timestamptz,"updated_at" timestamptz,"deleted_at" timestamptz,"email" text NOT NULL,"password_hash" text NOT NULL,"name" text,PRIMARY KEY ("id"));
CREATE INDEX IF NOT EXISTS "idx_users_deleted_at" ON "users" ("deleted_at");
CREATE UNIQUE INDEX IF NOT EXISTS "idx_users_email" ON "users" ("email");

CREATE TABLE IF NOT EXISTS "organization_members" ("id" uuid,"created_at" timestamptz,"updated_at" timestamptz,"deleted_at" timestamptz,"organization_id" uuid NOT NULL,"user_id" uuid NOT NULL,"role" text NOT NULL DEFAULT 'VIEWER',PRIMARY KEY ("id"),CONSTRAINT "fk_organization_members_user" FOREIGN KEY ("user_id") REFERENCES "users"("id"),CONSTRAINT "fk_organizations_users" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id"));
CREATE INDEX IF NOT EXISTS "idx_organization_members_user_id" ON "organization_members" ("user_id");
CREATE INDEX IF NOT EXISTS "idx_organization_members_organization_id" ON "organization_members" ("organization_id");
CREATE INDEX IF NOT EXISTS "idx_organization_members_deleted_at" ON "organization_members" ("deleted_at");

CREATE TABLE IF NOT EXISTS "agents" ("id" uuid,"created_at" timestamptz,"updated_at" timestamptz,"deleted_at" timestamptz,"organization_id" uuid NOT NULL,"name" text NOT NULL,"token_hash" text,"token_rotated_at" timestamptz,"status" text NOT NULL DEFAULT 'OFFLINE',"version" text,"last_seen_at" timestamptz,"registration_key" text,PRIMARY KEY ("id"));
CREATE UNIQUE INDEX IF NOT EXISTS "idx_agents_registration_key" ON "agents" ("registration_key");
CREATE UNIQUE INDEX IF NOT EXISTS "idx_agents_token_hash" ON "agents" ("token_hash");
CREATE INDEX IF NOT EXISTS "idx_agents_organization_id" ON "agents" ("organization_id");
CREATE INDEX IF NOT EXISTS "idx_agents_deleted_at" ON "agents" ("deleted_at");

CREATE TABLE IF NOT EXISTS "machines" ("id" uuid,"created_at" timestamptz,"updated_at" timestamptz,"deleted_at" timestamptz,"organization_id" uuid NOT NULL,"agent_id" uuid NOT NULL,"hostname" text NOT NULL,"os" text,"ip_address" text,PRIMARY KEY ("id"),CONSTRAINT "fk_machines_agent" FOREIGN KEY ("agent_id") REFERENCES "agents"("id"));
CREATE INDEX IF NOT EXISTS "idx_machines_agent_id" ON "machines" ("agent_id");
CREATE INDEX IF NOT EXISTS "idx_machines_organization_id" ON "machines" ("organization_id");
CREATE INDEX IF NOT EXISTS "idx_machines_deleted_at" ON "machines" ("deleted_at");

CREATE TABLE IF NOT EXISTS "storage_targets" ("id" uuid,"created_at" timestamptz,"updated_at" timestamptz,"deleted_at" timestamptz,"organization_id" uuid NOT NULL,"name" text NOT NULL,"type" text NOT NULL,"bucket" text,"region" text,"endpoint" text,"encrypted_access_key" text,"encrypted_secret_key" text,"path" text,PRIMARY KEY ("id"));
CREATE INDEX IF NOT EXISTS "idx_storage_targets_organization_id" ON "storage_targets" ("organization_id");
CREATE INDEX IF NOT EXISTS "idx_storage_targets_deleted_at" ON "storage_targets" ("deleted_at");

CREATE TABLE IF NOT EXISTS "backup_jobs" ("id" uuid,"created_at" timestamptz,"updated_at" timestamptz,"deleted_at" timestamptz,"organization_id" uuid NOT NULL,"agent_id" uuid NOT NULL,"storage_target_id" uuid NOT NULL,"name" text NOT NULL,"source_type" text NOT NULL,"source_path" text,"source_database" text,"include_patterns" text,"exclude_patterns" text,"export_format" text DEFAULT 'ARCHIVE',"mode" text NOT NULL DEFAULT 'NORMAL',"encrypted" boolean DEFAULT false,"retention_days" bigint DEFAULT 30,"enabled" boolean DEFAULT true,"sla_target_minutes" bigint DEFAULT 0,"rpo_target_minutes" bigint DEFAULT 0,"rto_target_minutes" bigint DEFAULT 0,PRIMARY KEY ("id"),CONSTRAINT "fk_backup_jobs_agent" FOREIGN KEY ("agent_id") REFERENCES "agents"("id"),CONSTRAINT "fk_backup_jobs_storage_target" FOREIGN KEY ("storage_target_id") REFERENCES "storage_targets"("id"));
CREATE INDEX IF NOT EXISTS "idx_backup_jobs_storage_target_id" ON "backup_jobs" ("storage_target_id");
CREATE INDEX IF NOT EXISTS "idx_backup_jobs_agent_id" ON "backup_jobs" ("agent_id");
CREATE INDEX IF NOT EXISTS "idx_backup_jobs_organization_id" ON "backup_jobs" ("organization_id");
CREATE INDEX IF NOT EXISTS "idx_backup_jobs_deleted_at" ON "backup_jobs" ("deleted_at");

CREATE TABLE IF NOT EXISTS "backup_schedules" ("id" uuid,"created_at" timestamptz,"updated_at" timestamptz,"deleted_at" timestamptz,"backup_job_id" uuid NOT NULL,"cron_expr" text NOT NULL,"timezone" text DEFAULT 'UTC',PRIMARY KEY ("id"),CONSTRAINT "fk_backup_jobs_schedule" FOREIGN KEY ("backup_job_id") REFERENCES "backup_jobs"("id"));
CREATE UNIQUE INDEX IF NOT EXISTS "idx_backup_schedules_backup_job_id" ON "backup_schedules" ("backup_job_id");
CREATE INDEX IF NOT EXISTS "idx_backup_schedules_deleted_at" ON "backup_schedules" ("deleted_at");

CREATE TABLE IF NOT EXISTS "backup_runs" ("id" uuid,"created_at" timestamptz,"updated_at" timestamptz,"deleted_at" timestamptz,"organization_id" uuid NOT NULL,"backup_job_id" uuid NOT NULL,"agent_id" uuid NOT NULL,"storage_target_id" uuid NOT NULL,"status" text NOT NULL DEFAULT 'PENDING',"cancel_requested" boolean DEFAULT false,"started_at" timestamptz,"completed_at" timestamptz,"bytes_read" bigint,"bytes_compressed" bigint,"bytes_uploaded" bigint,"duration_seconds" bigint,"source_type" text,"error_message" text,"failure_category" text DEFAULT '',"storage_path" text,"checksum" text,"data_key_encrypted" text,PRIMARY KEY ("id"),CONSTRAINT "fk_backup_runs_backup_job" FOREIGN KEY ("backup_job_id") REFERENCES "backup_jobs"("id"));
CREATE INDEX IF NOT EXISTS "idx_backup_runs_organization_id" ON "backup_runs" ("organization_id");
CREATE INDEX IF NOT EXISTS "idx_backup_runs_deleted_at" ON "backup_runs" ("deleted_at");
CREATE INDEX IF NOT EXISTS "idx_backup_runs_agent_id" ON "backup_runs" ("agent_id");
CREATE INDEX IF NOT EXISTS "idx_backup_runs_backup_job_id" ON "backup_runs" ("backup_job_id");

CREATE TABLE IF NOT EXISTS "backup_artifacts" ("id" uuid,"created_at" timestamptz,"updated_at" timestamptz,"deleted_at" timestamptz,"backup_run_id" uuid NOT NULL,"name" text NOT NULL,"size" bigint,"checksum" text,"storage_path" text,PRIMARY KEY ("id"));
CREATE INDEX IF NOT EXISTS "idx_backup_artifacts_backup_run_id" ON "backup_artifacts" ("backup_run_id");
CREATE INDEX IF NOT EXISTS "idx_backup_artifacts_deleted_at" ON "backup_artifacts" ("deleted_at");

CREATE TABLE IF NOT EXISTS "backup_chunks" ("id" uuid,"created_at" timestamptz,"updated_at" timestamptz,"deleted_at" timestamptz,"artifact_id" uuid NOT NULL,"index" bigint NOT NULL,"size" bigint,"checksum" text,"storage_path" text,"uploaded" boolean DEFAULT false,PRIMARY KEY ("id"));
CREATE UNIQUE INDEX IF NOT EXISTS "idx_chunk_artifact_index" ON "backup_chunks" ("index");
CREATE INDEX IF NOT EXISTS "idx_backup_chunks_artifact_id" ON "backup_chunks" ("artifact_id");
CREATE INDEX IF NOT EXISTS "idx_backup_chunks_deleted_at" ON "backup_chunks" ("deleted_at");

CREATE TABLE IF NOT EXISTS "restore_jobs" ("id" uuid,"created_at" timestamptz,"updated_at" timestamptz,"deleted_at" timestamptz,"organization_id" uuid NOT NULL,"backup_run_id" uuid NOT NULL,"agent_id" uuid NOT NULL,"status" text NOT NULL DEFAULT 'PENDING',"destination_path" text,"target_database" text,"started_at" timestamptz,"completed_at" timestamptz,"duration_seconds" bigint,"error_message" text,PRIMARY KEY ("id"),CONSTRAINT "fk_restore_jobs_backup_run" FOREIGN KEY ("backup_run_id") REFERENCES "backup_runs"("id"));
CREATE INDEX IF NOT EXISTS "idx_restore_jobs_agent_id" ON "restore_jobs" ("agent_id");
CREATE INDEX IF NOT EXISTS "idx_restore_jobs_backup_run_id" ON "restore_jobs" ("backup_run_id");
CREATE INDEX IF NOT EXISTS "idx_restore_jobs_organization_id" ON "restore_jobs" ("organization_id");
CREATE INDEX IF NOT EXISTS "idx_restore_jobs_deleted_at" ON "restore_jobs" ("deleted_at");

CREATE TABLE IF NOT EXISTS "backup_size_baselines" ("id" uuid,"created_at" timestamptz,"updated_at" timestamptz,"deleted_at" timestamptz,"backup_job_id" uuid NOT NULL,"avg_bytes" bigint NOT NULL DEFAULT 0,"sample_count" bigint NOT NULL DEFAULT 0,"last_updated_at" timestamptz,PRIMARY KEY ("id"));
CREATE INDEX IF NOT EXISTS "idx_backup_size_baselines_deleted_at" ON "backup_size_baselines" ("deleted_at");
CREATE UNIQUE INDEX IF NOT EXISTS "idx_backup_size_baselines_backup_job_id" ON "backup_size_baselines" ("backup_job_id");

CREATE TABLE IF NOT EXISTS "alerts" ("id" uuid,"created_at" timestamptz,"updated_at" timestamptz,"deleted_at" timestamptz,"organization_id" uuid NOT NULL,"type" text NOT NULL,"title" text NOT NULL,"message" text,"read" boolean DEFAULT false,"backup_job_id" uuid,"agent_id" uuid,PRIMARY KEY ("id"));
CREATE INDEX IF NOT EXISTS "idx_alerts_deleted_at" ON "alerts" ("deleted_at");
CREATE INDEX IF NOT EXISTS "idx_alerts_organization_id" ON "alerts" ("organization_id");

CREATE TABLE IF NOT EXISTS "audit_logs" ("id" uuid,"created_at" timestamptz,"updated_at" timestamptz,"deleted_at" timestamptz,"organization_id" uuid NOT NULL,"user_id" uuid NOT NULL,"action" text NOT NULL,"resource" text,"resource_id" text,"ip_address" text,PRIMARY KEY ("id"));
CREATE INDEX IF NOT EXISTS "idx_audit_logs_deleted_at" ON "audit_logs" ("deleted_at");
CREATE INDEX IF NOT EXISTS "idx_audit_logs_user_id" ON "audit_logs" ("user_id");
CREATE INDEX IF NOT EXISTS "idx_audit_logs_organization_id" ON "audit_logs" ("organization_id");

CREATE TABLE IF NOT EXISTS "refresh_tokens" ("id" uuid,"created_at" timestamptz,"updated_at" timestamptz,"deleted_at" timestamptz,"user_id" uuid NOT NULL,"family_id" uuid NOT NULL,"token" text NOT NULL,"expires_at" timestamptz,"revoked" boolean DEFAULT false,PRIMARY KEY ("id"));
CREATE INDEX IF NOT EXISTS "idx_refresh_tokens_user_id" ON "refresh_tokens" ("user_id");
CREATE INDEX IF NOT EXISTS "idx_refresh_tokens_deleted_at" ON "refresh_tokens" ("deleted_at");
CREATE UNIQUE INDEX IF NOT EXISTS "idx_refresh_tokens_token" ON "refresh_tokens" ("token");
CREATE INDEX IF NOT EXISTS "idx_refresh_tokens_family_id" ON "refresh_tokens" ("family_id");
