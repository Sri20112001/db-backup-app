-- Database details cache: JSON metadata (sizes, table counts) collected at
-- Test/Refresh time. database_names stays as the lightweight fallback.

ALTER TABLE "database_connections" ADD COLUMN IF NOT EXISTS "database_metadata" text DEFAULT '';
