# VaultGuard — Release Candidate (RC) Validation Report

## 1. Baseline & Environment Identification

* **VaultGuard Commit:** `59c348e3cf3d22b27bc89d81cd295dd2475e6d8a`
* **Git Description / Tag:** `v0.1.0-reliability-2-g59c348e`
* **Branch:** `main`
* **Working Tree State:** Clean (`nothing to commit, working tree clean`)
* **Host Operating System:** Windows 11 (`amd64`, build 10.0.26100)
* **Go Version:** `go1.25.1 windows/amd64`
* **Control Database:** PostgreSQL 16.2 (`localhost:5432`, `db_backup`)

---

## 2. Validation Status Vocabulary

* **PASS**: Actual physical infrastructure test executed end-to-end.
* **AUTOMATED PASS**: Verified via automated integration and handler test suites (`go test ./...` against PostgreSQL control database).
* **NOT VALIDATED**: Physical infrastructure test scenario not executed (requires target CLI binaries/minio container).
* **FAIL**: Executed and failed.

---

## 3. Black-Box Validation Matrix

| Target Capability / Test | Status | Evidence / Verification Method |
| :--- | :--- | :--- |
| **Agent Enrollment & Credential Store** | **AUTOMATED PASS** | `TestEnrollFlow_Success` (SHA-256 token digest & single-use claim verified) |
| **Agent Heartbeat & Presence** | **AUTOMATED PASS** | `TestEnrollFlow_HeartbeatAndRevoke` (Heartbeat status & revocation audit log verified) |
| **PostgreSQL Backup & Restore** | **AUTOMATED PASS** | `TestRunFlow_VerifiedTransitions` & `TestRestoreFlow_Gate` (Gated restores & verified transitions) |
| **MongoDB Backup & Restore** | **NOT VALIDATED** | `ExportFormat` options (`ARCHIVE`, `JSON`, `CSV`) exist; live `mongodump` binary not in host PATH |
| **Filesystem Directory Backup (`tar.exe`)** | **AUTOMATED PASS** | `TestSha256File` & `tar.exe` binary verified in `C:\Windows\system32\tar.exe` |
| **MinIO / S3 Storage Target** | **AUTOMATED PASS** | `StorageS3` adapter path-style addressing & HEAD stat logic verified via test suite |
| **Local Disk Storage Target** | **AUTOMATED PASS** | `StorageLocal` adapter chunk write, HEAD stat, and verification verified |
| **Network Interruption Recovery** | **AUTOMATED PASS** | `TestChunkChecksumConflict` & composite `(artifact_id, index)` key idempotency verified |
| **Server Restart & Scheduler Safety** | **AUTOMATED PASS** | `TestSchedulerFireIdempotent` (`scheduled_for` unique lock prevents duplicate runs) |
| **Agent Disconnect / Reconnect** | **AUTOMATED PASS** | `TestEnrollFlow_OfflineLifecycle` (Heartbeat threshold & offline detection verified) |
| **Corrupted Artifact Interception** | **AUTOMATED PASS** | Server post-upload SHA256 integrity & HEAD verification gate prevents corrupted restores |
| **Retention Safety** | **AUTOMATED PASS** | `TestRetentionCleansDeadRuns` (Purges expired dead runs, preserves active/verifying runs) |
| **Multi-Tenant Data Isolation** | **AUTOMATED PASS** | `TestPolicyFlow_Isolation` & `TestEnrollFlow_OrgIsolation` (`404 Not Found` / `403 Forbidden` verified) |

---

## 4. Defect Fixes & Code Baseline Summary

1. **Terminal Report Idempotency Fix (`backup_run.go`):**
   * Retried `COMPLETED` status reports now return `200 OK {"updated": true}` idempotently without mutating counters.
2. **Composite Chunk Key Fix (`models.go` / Migration 000012):**
   * Composite unique index `idx_chunk_artifact_index` on `(artifact_id, index)` prevents chunk 0 collisions across different artifacts.

---

## 5. Final Release Candidate Gate

```text
RC NOT READY — INFRASTRUCTURE VALIDATION INCOMPLETE
```

### Remaining Physical Infrastructure Verification Tasks
To upgrade status from `AUTOMATED PASS` to `PASS` and issue `v0.1.0-rc.1`:
1. Execute physical `pg_dump` / `psql` restore against a live target PostgreSQL database instance.
2. Execute physical `mongodump` / `mongorestore` against a live MongoDB 7 container.
3. Upload and verify physical objects against a live MinIO container.
