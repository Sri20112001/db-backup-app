-- S3 region reference data for the storage-target picker. AWS region codes
-- plus an `auto` preset for S3-compatible providers with account-scoped
-- endpoints (R2) or free-form endpoints (MinIO — typed per target).
-- Seeded with ON CONFLICT DO NOTHING so re-runs and upgrades are safe.
-- StorageTarget.Region stays free text: unknown future regions work even
-- before anyone inserts them here, and admins add private regions via
-- POST /storage-regions (is_system=false) instead of code deploys.

CREATE TABLE IF NOT EXISTS "s3_regions" (
	"id" uuid PRIMARY KEY,
	"created_at" timestamptz,
	"updated_at" timestamptz,
	"deleted_at" timestamptz,
	"code" text NOT NULL,
	"name" text NOT NULL,
	"provider" text NOT NULL DEFAULT 'AWS',
	"endpoint" text,
	"is_system" boolean NOT NULL DEFAULT true,
	"active" boolean NOT NULL DEFAULT true,
	CONSTRAINT "uni_s3_regions_code" UNIQUE ("code")
);
CREATE INDEX IF NOT EXISTS "idx_s3_regions_deleted_at" ON "s3_regions" ("deleted_at");
CREATE INDEX IF NOT EXISTS "idx_s3_regions_active" ON "s3_regions" ("active");

-- Fixed IDs (not gen_random_uuid(), which needs the pgcrypto extension
-- non-superusers may not install). Deterministic across environments.
INSERT INTO "s3_regions" ("id", "created_at", "updated_at", "code", "name", "provider") VALUES
	('b96da2fe-4f19-4469-bfb1-310978a3670c', now(), now(), 'us-east-1', 'US East (N. Virginia)', 'AWS'),
	('2c67ebf5-6786-40ef-927f-45557371dda9', now(), now(), 'us-east-2', 'US East (Ohio)', 'AWS'),
	('1632e7dd-9866-45bb-ac64-7f9e1c0159a2', now(), now(), 'us-west-1', 'US West (N. California)', 'AWS'),
	('4b767c0b-940c-4319-ade5-881e4c29ab8b', now(), now(), 'us-west-2', 'US West (Oregon)', 'AWS'),
	('161602a3-28b5-4a6b-a2a6-f0f6146c3412', now(), now(), 'af-south-1', 'Africa (Cape Town)', 'AWS'),
	('2bc6f16e-a66e-4136-89a0-3f282690781b', now(), now(), 'ap-east-1', 'Asia Pacific (Hong Kong)', 'AWS'),
	('f56ecc31-e332-4bd4-ab74-afa8f0be27e0', now(), now(), 'ap-south-1', 'Asia Pacific (Mumbai)', 'AWS'),
	('ee6841cf-30ae-4437-b58f-a4ed2ec2eac2', now(), now(), 'ap-south-2', 'Asia Pacific (Hyderabad)', 'AWS'),
	('bbff5f87-02f0-42b5-9c87-91c1ca0f143e', now(), now(), 'ap-northeast-1', 'Asia Pacific (Tokyo)', 'AWS'),
	('8fa3b9a7-db00-45a7-adbb-d8bf0bd4ec86', now(), now(), 'ap-northeast-2', 'Asia Pacific (Seoul)', 'AWS'),
	('74545ff2-5310-432a-80a6-8922eb35f5c9', now(), now(), 'ap-northeast-3', 'Asia Pacific (Osaka)', 'AWS'),
	('58e5a47b-ee30-4aab-beb9-faf80d2c1951', now(), now(), 'ap-southeast-1', 'Asia Pacific (Singapore)', 'AWS'),
	('84a22aec-673f-4d79-9f76-b747caf10579', now(), now(), 'ap-southeast-2', 'Asia Pacific (Sydney)', 'AWS'),
	('1dd1d9d3-e80d-4d50-a862-33ca540662db', now(), now(), 'ap-southeast-3', 'Asia Pacific (Jakarta)', 'AWS'),
	('2f0b48a5-6360-4e16-9d60-72755f5b9ee0', now(), now(), 'ap-southeast-4', 'Asia Pacific (Melbourne)', 'AWS'),
	('65ed91f8-ab68-4bc6-9f56-6167f27eaf82', now(), now(), 'ca-central-1', 'Canada (Central)', 'AWS'),
	('f5d0ee3f-2abd-4d3e-ad63-28a89f5be57a', now(), now(), 'ca-west-1', 'Canada West (Calgary)', 'AWS'),
	('ec30b316-0b5b-4ee5-a4cf-cc521b5f9ff4', now(), now(), 'eu-central-1', 'Europe (Frankfurt)', 'AWS'),
	('3b3e302b-ef57-4902-b8e7-8ed0ce592cca', now(), now(), 'eu-central-2', 'Europe (Zurich)', 'AWS'),
	('f866357c-d132-4793-9385-c5c6497a2bf2', now(), now(), 'eu-west-1', 'Europe (Ireland)', 'AWS'),
	('15934382-438c-473e-8a9f-dc6f1e3fa32b', now(), now(), 'eu-west-2', 'Europe (London)', 'AWS'),
	('8f5d6ce9-56f3-4e22-b5c8-b24084d2f281', now(), now(), 'eu-west-3', 'Europe (Paris)', 'AWS'),
	('618dce66-2488-4b38-8b6c-19e53028919e', now(), now(), 'eu-south-1', 'Europe (Milan)', 'AWS'),
	('3e6e9e34-9ca2-4e3c-aa70-60b02f4ac103', now(), now(), 'eu-south-2', 'Europe (Spain)', 'AWS'),
	('ccf6f994-9545-49c5-852d-f25fe0f38363', now(), now(), 'eu-north-1', 'Europe (Stockholm)', 'AWS'),
	('ee622b79-9b41-4b55-81f1-3b856c1b24ae', now(), now(), 'il-central-1', 'Israel (Tel Aviv)', 'AWS'),
	('7fa073d3-a929-4186-8bbc-885632be16b5', now(), now(), 'me-south-1', 'Middle East (Bahrain)', 'AWS'),
	('da8a7698-2f0d-439a-b00e-a894ed20e22c', now(), now(), 'me-central-1', 'Middle East (UAE)', 'AWS'),
	('868b829d-f674-4fba-b058-a3bd59339fde', now(), now(), 'sa-east-1', 'South America (São Paulo)', 'AWS'),
	('4204e4f8-269b-4dcd-9f7f-00f48138179f', now(), now(), 'auto', 'Custom / S3-compatible (R2, MinIO, Wasabi…)', 'CUSTOM')
ON CONFLICT ("code") DO NOTHING;
