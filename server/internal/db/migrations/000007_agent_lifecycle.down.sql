DROP TABLE IF EXISTS "enrollment_tokens";
ALTER TABLE "agents" DROP COLUMN IF EXISTS "registration_expires_at";
ALTER TABLE "agents" DROP COLUMN IF EXISTS "revoked_at";
ALTER TABLE "agents" DROP COLUMN IF EXISTS "installed_at";
ALTER TABLE "agents" DROP COLUMN IF EXISTS "machine_name";
ALTER TABLE "agents" DROP COLUMN IF EXISTS "architecture";
ALTER TABLE "agents" DROP COLUMN IF EXISTS "platform";
DROP INDEX IF EXISTS "idx_agents_revoked_at";
