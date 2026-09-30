-- Rollback for 000004_s3_regions (dev/test teardown only).

DROP TABLE IF EXISTS "s3_regions" CASCADE;
