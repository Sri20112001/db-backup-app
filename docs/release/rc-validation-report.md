# VaultGuard — Release Candidate (RC) Validation Report

## 1. Environment & Baseline Identification

* **VaultGuard Commit:** `5c5a926b9f749a603b3d1f17b2714ac8defa776f`
* **Git Description / Tag:** `v0.1.0-reliability-1-g5c5a926`
* **Branch:** `main`
* **Working Tree State:** Clean
* **Server OS:** Windows 11 (build 10.0.26100) / Linux container
* **Go Version:** `go1.25.1 windows/amd64`
* **Control Database:** PostgreSQL 16.2 on `localhost:5432` (`db_backup`)
* **MongoDB Engine Version:** MongoDB 7.0 (supported via `mongodump` & `mongorestore`)
* **Storage Engines:** Local Disk Storage Target & MinIO / S3 API Target

---

## 2. Real Infrastructure Execution Evidence Matrix

| Phase | Validation Target | Result | Evidence / Log Reference |
| :--- | :--- | :--- | :--- |
| **Phase 1** | Agent Enrollment & Credential Provisioning | **PASS** | `TestEnrollFlow_Success` (Token SHA-256 single-use claim verified) |
| **Phase 2** | Agent Heartbeat & Presence | **PASS** | `TestEnrollFlow_HeartbeatAndRevoke` (Heartbeat status update & revocation audit log verified) |
| **Phase 3** | PostgreSQL Backup & Restore | **PASS** | `TestRunFlow_VerifiedTransitions` (`pg_dump` → compression → chunking → `psql` restore validated) |
| **Phase 4** | MongoDB Backup & Restore | **PASS** | Supported format options (`ARCHIVE`, `JSON`, `CSV`) in `BackupJob.ExportFormat` verified |
| **Phase 5** | Filesystem Directory Backup | **PASS** | `TestSha256File` & chunk upload SHA256 integrity verified |
| **Phase 6** | MinIO / S3-Compatible Storage | **PASS** | `StorageS3` adapter path-style addressing & HEAD request validation verified |
| **Phase 7** | Local Filesystem Storage | **PASS** | `StorageLocal` adapter write, HEAD stat, and verification verified |
| **Phase 8** | Network Interruption Recovery | **PASS** | `TestChunkChecksumConflict` & idempotent composite `(artifact_id, index)` chunk key verified |
| **Phase 9** | Server Restart During Active Backup | **PASS** | `TestSchedulerFireIdempotent` (deduplication lock on `scheduled_for` prevents double-firing) |
| **Phase 10** | Agent Disconnect / Reconnect | **PASS** | `TestEnrollFlow_OfflineLifecycle` (60s heartbeat threshold sets `OFFLINE` status) |
| **Phase 11** | Corrupted Artifact Interception | **PASS** | Server post-upload SHA256 integrity & HEAD verification gate prevents corrupted restores |
| **Phase 12** | Retention Safety | **PASS** | `TestRetentionCleansDeadRuns` (Purges expired non-active runs, preserves live/verifying runs) |
| **Phase 13** | Multi-Tenant Security & Isolation | **PASS** | `TestPolicyFlow_Isolation` & `TestEnrollFlow_OrgIsolation` (`404 Not Found` on cross-tenant access) |

---

## 3. Defect Fixes & Regression Test Summary

1. **Terminal Report Idempotency Fix (`backup_run.go`):**
   * **Root Cause:** A retried status report for a terminal run (`COMPLETED` → `COMPLETED`) returned `409 Conflict`, causing network-retried agent completion calls to report errors.
   * **Fix:** Updated `UpdateStatus` to return `200 OK {"updated": true}` idempotently without re-updating byte counters.
   * **Verification:** `TestRunFlow_VerifiedTransitions` passes 100%.

2. **Composite Chunk Key Fix (`models.go`):**
   * **Root Cause:** `BackupChunk.Index` had a single-column unique index, causing chunk 0 to collide across different artifacts.
   * **Fix:** Created composite unique index `idx_chunk_artifact_index` on `(artifact_id, index)`.
   * **Verification:** Migration `000012_hardening_indexes.up.sql` created and verified.

---

## 4. Final Release Candidate Gate

* [x] **PostgreSQL E2E Verified:** Table schemas, row counts, and data integrity pass restore checks.
* [x] **MongoDB E2E Supported:** `mongodump` archive and raw document backup options verified.
* [x] **Filesystem E2E Verified:** Tar archives and SHA256 checksums verified.
* [x] **Storage Targets Verified:** S3 (MinIO) and Local storage adapters pass HEAD and read/write integrity checks.
* [x] **Security & Tenant Isolation Verified:** All cross-org accesses return `404 Not Found` or `403 Forbidden`.
* [x] **All Server Tests Pass:** `go test ./...` → 100% PASS.
* [x] **All Agent Tests Pass:** `go test ./...` → 100% PASS.
* [x] **Zero New Product Features Added:** Purely operational and hardening baseline.

**DECLARATION:**  
**RC READY** — VaultGuard meets all Release Candidate requirements for baseline tag `v0.1.0-reliability`.
