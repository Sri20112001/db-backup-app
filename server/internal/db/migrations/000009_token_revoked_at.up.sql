-- Token revocation auditability: revoked_at distinguishes admin-revoked
-- tokens from consumed ones (revoke path sets it; enroll rejects it).

ALTER TABLE "enrollment_tokens" ADD COLUMN IF NOT EXISTS "revoked_at" timestamptz;
