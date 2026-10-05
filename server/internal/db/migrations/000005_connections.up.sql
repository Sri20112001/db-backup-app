-- Connections feature: reusable database connection metadata + nullable
-- backup_jobs.connection_id (nullable so legacy/filesystem jobs keep working).

CREATE TABLE IF NOT EXISTS "database_connections" ("id" uuid,"created_at" timestamptz,"updated_at" timestamptz,"deleted_at" timestamptz,"organization_id" uuid NOT NULL,"agent_id" uuid NOT NULL,"name" text NOT NULL,"type" text NOT NULL,"host" text NOT NULL,"port" bigint NOT NULL,"username" text NOT NULL,"encrypted_password" text,"status" text NOT NULL DEFAULT 'UNKNOWN',"last_checked_at" timestamptz,"database_names" text DEFAULT '',PRIMARY KEY ("id"),CONSTRAINT "fk_database_connections_agent" FOREIGN KEY ("agent_id") REFERENCES "agents"("id"));
CREATE INDEX IF NOT EXISTS "idx_database_connections_organization_id" ON "database_connections" ("organization_id");
CREATE INDEX IF NOT EXISTS "idx_database_connections_agent_id" ON "database_connections" ("agent_id");
CREATE INDEX IF NOT EXISTS "idx_database_connections_deleted_at" ON "database_connections" ("deleted_at");
CREATE UNIQUE INDEX IF NOT EXISTS "idx_database_connections_org_name" ON "database_connections" ("organization_id", "name") WHERE deleted_at IS NULL;

ALTER TABLE "backup_jobs" ADD COLUMN IF NOT EXISTS "connection_id" uuid;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_backup_jobs_connection') THEN
    ALTER TABLE "backup_jobs" ADD CONSTRAINT "fk_backup_jobs_connection" FOREIGN KEY ("connection_id") REFERENCES "database_connections"("id") ON DELETE SET NULL;
  END IF;
END $$;
CREATE INDEX IF NOT EXISTS "idx_backup_jobs_connection_id" ON "backup_jobs" ("connection_id");
