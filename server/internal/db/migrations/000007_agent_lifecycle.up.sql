-- Agent lifecycle + secure enrollment (Phase: agent lifecycle milestone).
--
-- agents: lifecycle metadata (platform/arch/machine/installed/revoked) plus
-- registration expiry. registration_key now stores a SHA-256 hex digest,
-- never the plaintext key: pre-migration pending (never-enrolled) keys are
-- single-use and transient, so they are cleared and must be regenerated.
-- No enrolled agent is affected (token_hash path unchanged).

ALTER TABLE "agents" ADD COLUMN IF NOT EXISTS "platform" text DEFAULT '';
ALTER TABLE "agents" ADD COLUMN IF NOT EXISTS "architecture" text DEFAULT '';
ALTER TABLE "agents" ADD COLUMN IF NOT EXISTS "machine_name" text DEFAULT '';
ALTER TABLE "agents" ADD COLUMN IF NOT EXISTS "installed_at" timestamptz;
ALTER TABLE "agents" ADD COLUMN IF NOT EXISTS "revoked_at" timestamptz;
ALTER TABLE "agents" ADD COLUMN IF NOT EXISTS "registration_expires_at" timestamptz;
UPDATE "agents" SET registration_key = NULL WHERE registration_key IS NOT NULL AND token_hash IS NULL;
CREATE INDEX IF NOT EXISTS "idx_agents_revoked_at" ON "agents" ("revoked_at");

-- enrollment_tokens: short-lived single-use enrollment tokens. Only the
-- SHA-256 hex digest is stored; the plaintext token is shown once at
-- creation and never logged or returned again.
CREATE TABLE IF NOT EXISTS "enrollment_tokens" ("id" uuid,"created_at" timestamptz,"updated_at" timestamptz,"deleted_at" timestamptz,"organization_id" uuid NOT NULL,"agent_id" uuid,"token_hash" text NOT NULL,"expires_at" timestamptz NOT NULL,"used_at" timestamptz,"created_by" uuid,PRIMARY KEY ("id"),CONSTRAINT "fk_enrollment_tokens_agent" FOREIGN KEY ("agent_id") REFERENCES "agents"("id") ON DELETE CASCADE);
CREATE UNIQUE INDEX IF NOT EXISTS "idx_enrollment_tokens_token_hash" ON "enrollment_tokens" ("token_hash");
CREATE INDEX IF NOT EXISTS "idx_enrollment_tokens_organization_id" ON "enrollment_tokens" ("organization_id");
CREATE INDEX IF NOT EXISTS "idx_enrollment_tokens_agent_id" ON "enrollment_tokens" ("agent_id");
CREATE INDEX IF NOT EXISTS "idx_enrollment_tokens_expires_at" ON "enrollment_tokens" ("expires_at");
CREATE INDEX IF NOT EXISTS "idx_enrollment_tokens_deleted_at" ON "enrollment_tokens" ("deleted_at");
