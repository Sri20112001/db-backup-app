-- Restore via saved connections: nullable restore_jobs.connection_id copied
-- from the source backup job at restore creation (NULL = filesystem/legacy).

ALTER TABLE "restore_jobs" ADD COLUMN IF NOT EXISTS "connection_id" uuid;
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_restore_jobs_connection') THEN
    ALTER TABLE "restore_jobs" ADD CONSTRAINT "fk_restore_jobs_connection" FOREIGN KEY ("connection_id") REFERENCES "database_connections"("id") ON DELETE SET NULL;
  END IF;
END $$;
CREATE INDEX IF NOT EXISTS "idx_restore_jobs_connection_id" ON "restore_jobs" ("connection_id");
