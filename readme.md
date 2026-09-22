# Backup SaaS

## Product & Technical Specification

**Document Version:** 1.0
**Project Type:** Multi-tenant B2B SaaS + On-Premise Backup Agent
**Primary Goal:** Automated backup and recovery for SMB/legacy Windows environments
**Backend:** Go
**Frontend:** React + TypeScript
**Agent:** Go Windows Service
**Database:** PostgreSQL

---

# 1. Product Overview

Build a SaaS-based backup and recovery platform designed primarily for small and medium-sized businesses that operate on-premise Windows servers and legacy business applications.

The platform must allow administrators to configure, schedule, monitor, and restore backups of:

* SQL Server databases
* DBF-based legacy databases
* Excel files (`.xlsx`, `.xls`)
* Documents
* Images
* Arbitrary files
* Entire folders
* Network/SMB folders
* Other day-to-day business data

The product consists of two major parts:

1. **Web-based SaaS control plane**
2. **Lightweight Go backup agent installed on customer machines**

The SaaS application should manage backup configuration and monitoring, while the local Go agent performs the actual backup operations.

---

# 2. Product Positioning

The product should focus on:

> Simple, reliable backup and recovery for SMBs running legacy databases, SQL Server, Windows applications, and business files.

The product should avoid the complexity of enterprise backup platforms while providing significantly more functionality than simple file-sync applications.

Primary target customers:

* Small businesses
* Local IT administrators
* Manufacturing companies
* Accounting firms
* Retail businesses
* Auto/service businesses
* Businesses using legacy ERP/accounting applications
* Companies running SQL Server Express
* Companies using FoxPro/dBase/DBF applications
* Small MSPs managing multiple customer environments

---

# 3. Core Architecture

The architecture must separate the **control plane** from the **data plane**.

```text
                    SaaS Control Plane
                           |
                +----------+----------+
                |                     |
                v                     v
          React Dashboard        Go API Server
                                      |
                                PostgreSQL
                                      |
                              Agent Communication
                                      |
                              gRPC over TLS
                                      |
                                      v
                         +-------------------------+
                         |     Go Backup Agent     |
                         |                         |
                         | Scheduler               |
                         | VSS                     |
                         | SQL Server              |
                         | DBF                     |
                         | File System             |
                         | Compression             |
                         | Encryption              |
                         | Chunking                |
                         | Upload                  |
                         | Restore                 |
                         +-----------+-------------+
                                     |
                    +----------------+----------------+
                    |                |                |
                    v                v                v
                  S3               R2              MinIO
                    |
                 Wasabi
                    |
               Local / NAS
```

The SaaS backend must not act as a bandwidth proxy for customer backup data.

The backup agent should upload backup data directly to the configured storage target whenever possible.

---

# 4. Applications

The repository should contain three primary applications.

```text
backup-saas/
│
├── frontend/
├── server/
├── agent/
├── proto/
├── deployment/
└── .github/
```

## 4.1 Frontend

React-based SaaS dashboard.

Responsibilities:

* Authentication
* Organization management
* Agent management
* Machine management
* Backup job configuration
* Schedule configuration
* Storage configuration
* Backup history
* Backup health
* Restore management
* Alerts
* Settings

---

## 4.2 Server

Go-based control-plane API.

Responsibilities:

* Authentication
* Authorization
* Multi-tenancy
* Organization management
* Agent registration
* Machine management
* Backup job configuration
* Schedule metadata
* Storage configuration
* Agent communication
* Backup run tracking
* Restore requests
* Alerts
* Audit logs

The server should not process or store customer backup payloads in the normal backup flow.

---

## 4.3 Agent

Go-based application installed on customer infrastructure.

The agent should run as a Windows Service.

Responsibilities:

* Connect to SaaS
* Receive backup jobs
* Execute scheduled jobs
* Perform filesystem backups
* Perform SQL Server backups
* Handle DBF datasets
* Use Windows VSS where required
* Compress data
* Encrypt data
* Generate checksums
* Split data into chunks
* Upload directly to storage
* Resume interrupted uploads
* Restore backups
* Report progress
* Report failures
* Report health
* Self-update in future versions

---

# 5. Technology Stack

## Frontend

Use:

* React
* TypeScript
* Vite
* Tailwind CSS
* Zustand
* TanStack Query
* Recharts

Do not use Next.js for the frontend.

---

## Backend

Use:

* Go
* Gin
* GORM
* PostgreSQL
* Zerolog

Do not introduce Node.js or another backend language.

---

## Agent

Use:

* Go
* Windows Service integration
* Windows VSS
* SQL Server native backup functionality
* S3-compatible SDK
* Zstandard
* AES-256-GCM
* SHA-256

---

## Communication

Use:

* REST/HTTPS for browser → API
* gRPC over TLS for agent → API
* WebSocket for browser real-time updates if required

The agent must initiate outbound communication.

Do not require inbound ports to be opened on customer networks.

---

## Infrastructure

Use:

* Docker
* Docker Compose
* Apache
* PostgreSQL
* GitHub Actions

Apache will act as the reverse proxy for the web application/API deployment.

---

## Observability

Use:

* Zerolog for application logs
* Prometheus for metrics
* Grafana for infrastructure/agent metrics

Observability should initially remain simple and should not introduce unnecessary distributed infrastructure.

---

# 6. Backup Sources

The system must support different source types.

## 6.1 Filesystem

Example:

```text
D:\CompanyData\
```

The user should be able to select an entire folder.

The backup should preserve:

* Directory structure
* File names
* File metadata where practical
* File contents

---

## 6.2 File patterns

Allow optional include/exclude patterns.

Examples:

```text
*.xlsx
*.xls
*.pdf
*.docx
*.csv
```

Exclude:

```text
*.tmp
*.log
node_modules/
.cache/
```

Patterns should be configurable per backup job.

---

## 6.3 SQL Server

SQL Server must be treated as an application-aware backup source.

Do not simply copy `.mdf` files while SQL Server is running.

The agent should use SQL Server's native backup mechanism.

Initial support:

```text
Full Database Backup
```

Future support:

```text
Differential Backup
Transaction Log Backup
```

Example flow:

```text
SQL Server
    |
    v
BACKUP DATABASE
    |
    v
.bak
    |
    v
Compression
    |
    v
Encryption
    |
    v
Upload
```

The `.mdf` and `.ldf` files should not be treated as ordinary files for the primary SQL Server backup workflow.

---

# 7. DBF Backup

Support legacy DBF-based applications.

A DBF backup should normally operate on the entire relevant dataset directory rather than a single `.dbf` file.

Example:

```text
D:\LegacyERP\Data\
```

Possible related files:

```text
CUSTOMER.DBF
CUSTOMER.CDX
CUSTOMER.FPT
CUSTOMER.DBT
```

The system should preserve the complete dataset.

For active Windows files, use VSS where required to obtain a consistent snapshot.

---

# 8. Backup Modes

The user must be able to select:

```text
Normal
Compressed
```

Future:

```text
Compressed + Encrypted
```

Encryption should be enabled by default for cloud backups once implemented.

---

# 9. Backup Pipeline

The backup engine should use a streaming pipeline.

Conceptually:

```text
Source
  |
  v
Snapshot / Native Backup
  |
  v
Reader
  |
  v
Chunker
  |
  v
Compression
  |
  v
Encryption
  |
  v
Checksum
  |
  v
Upload Manager
  |
  v
Storage
```

Do not load entire backups into RAM.

The system must support large files and large backup sets.

---

# 10. Compression

Use Zstandard.

For compressed mode:

```text
Source
  ↓
Zstandard
  ↓
Encryption
  ↓
Upload
```

Compression level should be configurable internally but should initially expose a simple user-facing setting.

Example:

```text
Compression:
Enabled
```

---

# 11. Encryption

Use AES-256-GCM.

Encryption must happen on the customer machine before data leaves the customer environment.

Conceptually:

```text
Customer Machine

Original Data
     |
     v
Compression
     |
     v
AES-256-GCM
     |
     v
Encrypted Backup
     |
     v
Cloud Storage
```

The SaaS server should not require access to plaintext backup data.

Key management must be designed carefully.

Do not store raw encryption keys in plaintext in PostgreSQL.

---

# 12. Integrity

Use SHA-256 checksums.

Each backup artifact/chunk should have integrity information.

Example:

```text
chunk-000001
size: 64 MB
sha256: ...
```

The system should be able to verify:

* Local backup generation
* Upload integrity
* Download integrity
* Restore integrity

A backup should not be marked fully healthy until required verification succeeds.

---

# 13. Storage Targets

The system should support an abstraction around storage providers.

Initial target:

```text
S3-compatible object storage
```

This should allow:

* AWS S3
* Cloudflare R2
* MinIO
* Wasabi
* Other S3-compatible services

Also support:

```text
Local filesystem
SMB/NAS
```

Create a storage abstraction so additional providers can be added without changing the backup engine.

Conceptually:

```text
StorageProvider

Upload()
Download()
Delete()
List()
Stat()
```

---

# 14. S3 Multipart Upload

Large backups must use multipart uploads.

Example:

```text
20 GB Backup

Chunk 1
Chunk 2
Chunk 3
...
Chunk N
```

Upload chunks independently.

If an upload fails:

```text
Uploaded:
1-14

Failed:
15

Resume:
15
```

Do not restart the entire 20 GB upload.

---

# 15. Backup Jobs

A backup job is the primary operational object.

Example:

```text
Name:
Daily ERP Backup

Agent:
OFFICE-SERVER-01

Source:
SQL Server

Database:
ERP

Mode:
Compressed

Encryption:
Enabled

Destination:
AWS S3

Schedule:
Daily at 11:00 PM

Retention:
30 days
```

Another example:

```text
Name:
Company Documents

Source:
D:\CompanyDocuments

Mode:
Compressed

Destination:
S3

Schedule:
Every 6 hours
```

---

# 16. Scheduling

Support:

```text
Every X minutes
Hourly
Daily
Weekly
Monthly
```

Examples:

```text
Every day at 23:00

Every 6 hours

Every Monday at 01:00

Every 30 minutes
```

Eventually support cron expressions.

The agent should be able to execute schedules locally so a temporary SaaS connectivity issue does not prevent scheduled backups.

The SaaS remains the source of truth for configuration.

---

# 17. Agent Communication

The agent should establish an outbound gRPC connection.

```text
Agent
  |
  | outbound TLS
  v
SaaS API
```

No inbound connection should be required to the customer's network.

The server should be able to send commands such as:

```text
RUN_BACKUP
CANCEL_BACKUP
START_RESTORE
GET_STATUS
UPDATE_AGENT
```

The agent should report:

```text
AGENT_CONNECTED
BACKUP_STARTED
BACKUP_PROGRESS
BACKUP_COMPLETED
BACKUP_FAILED
RESTORE_STARTED
RESTORE_COMPLETED
RESTORE_FAILED
AGENT_HEALTH
```

---

# 18. Agent Registration

A customer should install the agent and receive a registration code/token from the web application.

Flow:

```text
Admin
  |
  v
Web Dashboard
  |
  v
Generate Agent Registration Token
  |
  v
Install Agent
  |
  v
Enter Token
  |
  v
Agent connects to SaaS
  |
  v
Machine registered
```

The agent should receive a unique identity.

Never use a shared static credential for all agents.

---

# 19. Backup Run Lifecycle

Every execution of a backup job should create a backup run.

Possible statuses:

```text
PENDING
RUNNING
UPLOADING
VERIFYING
COMPLETED
FAILED
CANCELLED
```

Example:

```text
Backup Job
    |
    +-- Run #001 → COMPLETED
    +-- Run #002 → COMPLETED
    +-- Run #003 → FAILED
    +-- Run #004 → COMPLETED
```

Store:

* Start time
* End time
* Duration
* Bytes read
* Bytes compressed
* Bytes uploaded
* Compression ratio
* Status
* Error information
* Storage target
* Agent
* Source

---

# 20. Backup History

The dashboard should show historical recovery points.

Example:

```text
ERP Database

22 Sep 2026 23:00
21 Sep 2026 23:00
20 Sep 2026 23:00
19 Sep 2026 23:00
18 Sep 2026 23:00
```

Each recovery point should provide:

```text
View Details
Restore
Delete
Verify
```

---

# 21. Restore

Restore must be a first-class feature.

For filesystem backups:

```text
Restore

Source:
Company Documents

Recovery Point:
21 Sep 2026 23:00

Destination:
○ Original location
○ Custom location

[Restore]
```

Support:

* Full restore
* Folder restore
* File restore
* Restore to original location
* Restore to alternate location

For SQL Server:

```text
SQL Server Restore

Database:
ERP

Recovery Point:
21 Sep 2026 23:00

Target:
OFFICE-SERVER-01

Database:
ERP_RESTORED
```

The restore operation should execute on the agent, not through the browser.

---

# 22. Retention

Implement retention policies.

Initial configuration:

```text
Keep backups for:
30 days
```

Future:

```text
Daily:
30 days

Weekly:
12 weeks

Monthly:
12 months
```

The retention system must never delete a backup that is still required by the retention policy.

---

# 23. Backup Health

Create a dedicated Backup Health system.

Health should consider:

* Last successful backup
* Expected schedule
* Last failure
* Agent connectivity
* Backup verification
* Recovery point availability
* Storage availability

Example:

```text
ERP Database

● HEALTHY

Last successful backup:
2 hours ago

Expected interval:
24 hours

Recovery points:
30

Last verification:
2 hours ago
```

Example warning:

```text
⚠ WARNING

Last successful backup:
31 hours ago

Expected interval:
24 hours
```

The system must detect **silent failures**, including situations where a backup job never ran or the agent stopped reporting.

---

# 24. Alerts

Initial alert channels:

```text
Email
```

Future:

```text
Slack
Telegram
Webhook
```

Trigger alerts for:

* Backup failure
* Backup missed
* Agent offline
* Storage unavailable
* Restore failure
* Low storage
* Verification failure

Example:

```text
Backup Failed

Job:
Daily ERP Backup

Machine:
OFFICE-SERVER-01

Reason:
SQL Server unavailable

Last successful backup:
22 hours ago
```

---

# 25. Multi-Tenancy

The SaaS must support multiple organizations.

Core hierarchy:

```text
Organization
   |
   ├── Users
   ├── Agents
   ├── Machines
   ├── Backup Jobs
   ├── Storage Targets
   ├── Backup Runs
   ├── Restore Jobs
   └── Alerts
```

Every organization-owned resource must contain an organization relationship.

Never allow users to access another organization's resources.

---

# 26. User Roles

Initial roles:

```text
OWNER
ADMIN
OPERATOR
VIEWER
```

Example permissions:

```text
OWNER
- Everything

ADMIN
- Manage users
- Manage agents
- Manage jobs
- Manage storage
- Restore

OPERATOR
- Run backups
- View backups
- Restore

VIEWER
- Read-only
```

RBAC can be expanded later.

---

# 27. Database Model

Use PostgreSQL with GORM.

Initial entities:

```text
users

organizations

organization_members

agents

machines

backup_jobs

backup_sources

backup_schedules

storage_targets

backup_runs

backup_artifacts

backup_chunks

restore_jobs

alerts

audit_logs
```

Relationships:

```text
Organization
   |
   +-- Members
   |
   +-- Agents
   |     |
   |     +-- Machine
   |
   +-- Backup Jobs
   |     |
   |     +-- Sources
   |     +-- Schedule
   |     +-- Storage Target
   |
   +-- Backup Runs
   |     |
   |     +-- Artifacts
   |
   +-- Restore Jobs
   |
   +-- Alerts
   |
   +-- Audit Logs
```

---

# 28. Suggested Backend API

Use REST APIs for browser communication.

Example:

```text
POST   /api/auth/login
POST   /api/auth/refresh
POST   /api/auth/logout

GET    /api/organizations
POST   /api/organizations

GET    /api/agents
POST   /api/agents/register
DELETE /api/agents/:id

GET    /api/machines

GET    /api/backup-jobs
POST   /api/backup-jobs
GET    /api/backup-jobs/:id
PUT    /api/backup-jobs/:id
DELETE /api/backup-jobs/:id

POST   /api/backup-jobs/:id/run
POST   /api/backup-jobs/:id/enable
POST   /api/backup-jobs/:id/disable

GET    /api/backup-runs
GET    /api/backup-runs/:id

GET    /api/storage-targets
POST   /api/storage-targets

GET    /api/restores
POST   /api/restores

GET    /api/alerts
PUT    /api/alerts/:id/read
```

API design should remain RESTful and predictable.

---

# 29. Frontend Pages

Initial routes:

```text
/login

/dashboard

/agents
/agents/:id

/machines
/machines/:id

/backup-jobs
/backup-jobs/new
/backup-jobs/:id

/backups
/backups/:id

/restores
/restores/:id

/storage

/alerts

/settings
/settings/users
/settings/security
```

---

# 30. Dashboard

The main dashboard should show:

```text
Backup Health
```

```text
Total Jobs
Successful
Failed
Warning
Agents Online
Agents Offline
Storage Used
Last Backup
```

Example:

```text
+---------------------------------------------+
| Backup Overview                             |
+---------------------------------------------+
|                                             |
|  12 Jobs       11 Healthy       1 Warning  |
|                                             |
|  8 Agents      7 Online         1 Offline   |
|                                             |
|  428 GB Stored                              |
|                                             |
+---------------------------------------------+
```

Recent backup activity:

```text
✓ ERP Database          2 min ago
✓ Company Documents     1 hour ago
✓ Accounting DB         4 hours ago
✗ HR Documents          Yesterday
```

---

# 31. Backup Job UI

The job creation wizard should be divided into steps.

```text
1. Source
2. Schedule
3. Processing
4. Destination
5. Retention
6. Review
```

Source:

```text
○ Folder
○ DBF
○ SQL Server
```

Processing:

```text
Compression:
○ Disabled
● Zstandard

Encryption:
○ Disabled
● AES-256-GCM
```

Destination:

```text
○ Local
○ SMB
○ S3
```

---

# 32. Security Requirements

Security is critical because this system manages access to customer backup infrastructure.

Implement:

* Password hashing
* JWT/refresh token security
* TLS
* Agent authentication
* Organization-level authorization
* Role-based permissions
* Encrypted storage credentials
* Audit logs
* Secure secret handling
* No plaintext backup payloads on SaaS servers
* Encryption before cloud upload
* Secure agent registration
* Token rotation where appropriate

Never log:

* Passwords
* Access tokens
* Secret keys
* Encryption keys
* Storage credentials

---

# 33. Storage Credentials

S3 credentials must not be returned to the frontend unnecessarily.

Sensitive credentials should be encrypted at rest.

The agent should receive only the credentials required for its assigned storage target.

Future improvement:

Use short-lived credentials or workload-specific credentials where supported.

---

# 34. Windows VSS

VSS support is required for reliable backup of active Windows files.

VSS functionality must be isolated behind an abstraction.

Conceptually:

```go
type SnapshotProvider interface {
    CreateSnapshot(...)
    MountSnapshot(...)
    ReleaseSnapshot(...)
}
```

Windows implementation:

```text
VSSSnapshotProvider
```

The generic backup pipeline should not directly depend on Windows-specific APIs.

---

# 35. Windows Service

The agent must run as a Windows Service.

Responsibilities:

* Start automatically
* Restart after crash
* Maintain SaaS connection
* Run scheduled jobs
* Perform backups
* Perform restores
* Report health
* Handle shutdown gracefully

A future desktop tray application can be added separately.

---

# 36. Agent Reliability

The agent must handle:

* Network interruption
* SaaS disconnection
* S3 failure
* Machine restart
* Backup cancellation
* Partial uploads
* Process crashes
* Storage unavailable
* SQL Server unavailable
* Insufficient disk space

The agent should retry transient failures with exponential backoff.

Do not retry permanent failures indefinitely.

---

# 37. Offline Behavior

If the SaaS is temporarily unreachable:

The agent should:

* Continue executing locally configured schedules where possible
* Queue status updates
* Retry SaaS connection
* Preserve backup results locally
* Synchronize state after reconnecting

A temporary control-plane outage should not automatically destroy the data-plane backup schedule.

---

# 38. Local Agent State

The agent should maintain a small local state database or state directory for:

* Agent identity
* Job configuration cache
* Upload state
* Resume information
* Last execution
* Pending status events

Use a lightweight local database if required.

SQLite is acceptable for **agent-local state**.

PostgreSQL remains the SaaS database.

---

# 39. Backup Metadata

For each recovery point maintain metadata such as:

```text
Backup ID
Job ID
Agent ID
Machine ID
Source
Created At
Completed At
Original Size
Compressed Size
Encrypted
Storage Provider
Object Path
Checksum
Status
```

Example:

```json
{
  "backupId": "...",
  "jobId": "...",
  "source": "ERP Database",
  "originalSize": 2147483648,
  "compressedSize": 734003200,
  "encrypted": true,
  "status": "COMPLETED"
}
```

---

# 40. Object Storage Layout

Use deterministic object paths.

Example:

```text
organization/
  agent/
    backup-job/
      backup-run/
        manifest.json
        chunks/
          000001
          000002
          000003
```

Do not expose internal filesystem paths unnecessarily in object names.

---

# 41. Manifest

Each backup should contain a manifest describing the backup.

Example:

```text
manifest.json

Backup ID
Job ID
Agent ID
Created At
Source Type
Files
Sizes
Checksums
Compression
Encryption
Chunk information
```

The manifest is required for reliable restore operations.

---

# 42. Restore Architecture

Restore must be agent-driven.

```text
User
 |
 v
React
 |
 v
Go API
 |
 v
Restore Job
 |
 v
Agent
 |
 v
Download
 |
 v
Verify
 |
 v
Decrypt
 |
 v
Decompress
 |
 v
Restore
```

The browser must never download large backup payloads through the SaaS API unless explicitly required for a small metadata operation.

---

# 43. MVP Scope

The first version should focus on reliability instead of feature quantity.

## MVP must include

### SaaS

* Authentication
* Organization
* User
* Agent registration
* Machine list
* Backup jobs
* Scheduling
* Storage configuration
* Backup history
* Backup health
* Manual "Run Now"
* Basic restore
* Email alerts

### Agent

* Windows Service
* Agent registration
* Filesystem backup
* SQL Server full backup
* DBF directory backup
* VSS
* Normal mode
* Zstandard compression
* AES-256-GCM encryption
* SHA-256
* S3-compatible storage
* Local storage
* Multipart upload
* Retry/resume
* Restore
* Agent health

---

# 44. Do Not Implement Initially

Avoid unnecessary complexity during MVP.

Do not initially implement:

* Kubernetes
* Kafka
* RabbitMQ
* Redis unless a real requirement appears
* Microservices
* Mobile application
* AI features
* Billing
* White-labeling
* Advanced deduplication
* Complex global infrastructure
* Multiple regional deployments
* Custom distributed filesystem
* Dozens of cloud providers

The first priority is:

> Reliable backup + reliable restore.

---

# 45. Development Phases

## Phase 1 — Project Foundation

Build:

* Repository structure
* React application
* Go API
* PostgreSQL
* GORM
* Authentication
* Docker Compose
* Apache configuration
* GitHub Actions

---

## Phase 2 — Agent

Build:

* Go agent
* Windows Service
* Agent registration
* gRPC communication
* Agent heartbeat
* Agent health
* Basic job execution

---

## Phase 3 — Filesystem Backup

Implement:

* Folder source
* File scanner
* Include/exclude patterns
* Normal backup
* Local storage
* Backup metadata
* Backup history

---

## Phase 4 — Compression & Encryption

Implement:

* Streaming architecture
* Zstandard
* AES-256-GCM
* SHA-256
* Manifest
* Chunking

---

## Phase 5 — Cloud Storage

Implement:

* S3
* Multipart uploads
* Retry
* Resume
* Upload progress
* Storage verification

Ensure S3-compatible providers work.

---

## Phase 6 — SQL Server

Implement:

* SQL Server discovery/configuration
* Full backup
* `.bak` creation
* Compression
* Encryption
* Upload
* Restore

Do not implement raw `.mdf` copying as the primary SQL Server backup mechanism.

---

## Phase 7 — DBF + VSS

Implement:

* DBF directory backup
* VSS snapshots
* Snapshot-based filesystem reading
* Related DBF files
* Restore

---

## Phase 8 — Restore

Implement:

* Browse recovery points
* Full restore
* File restore
* Folder restore
* Alternate destination
* SQL Server restore
* Restore progress
* Restore history

---

## Phase 9 — Monitoring

Implement:

* Backup health
* Agent health
* Failed backup detection
* Missed backup detection
* Email notifications
* Dashboard analytics

---

## Phase 10 — Production Hardening

Implement:

* RBAC
* Audit logs
* Security hardening
* Agent update mechanism
* Improved retry logic
* Resource limits
* Metrics
* Prometheus
* Grafana
* Backup verification
* Operational logging

---

# 46. Future Features

After MVP:

```text
Incremental backups
Differential SQL backups
SQL transaction log backups
Deduplication
Global deduplication
GFS retention
Backup verification
Automated test restores
Ransomware protection
Immutable backups
Object Lock
MSP multi-client management
White labeling
Billing
Team management
Slack
Telegram
Webhooks
Additional cloud providers
Linux agent
macOS agent
```

---

# 47. MSP Mode

Future architecture should allow:

```text
MSP Organization
       |
       ├── Customer A
       │     ├── Servers
       │     └── Backup Jobs
       │
       ├── Customer B
       │     ├── Servers
       │     └── Backup Jobs
       │
       └── Customer C
             ├── Servers
             └── Backup Jobs
```

MSPs should eventually be able to monitor multiple customers from one dashboard.

---

# 48. Design Principles

Follow these principles throughout development.

### Reliability over features

A backup system that occasionally fails silently is worse than a smaller system that is dependable.

### Restore is equally important as backup

Every backup feature must consider how the data will be restored.

### Data plane and control plane separation

The SaaS should manage configuration and metadata.

The agent should handle customer data.

### Security by default

Encrypt before cloud upload.

### Streaming over memory-heavy operations

Never assume backups are small.

### Provider abstraction

Use interfaces so storage providers can be replaced or extended.

### Platform-specific code isolation

Windows/VSS code should not leak into generic backup logic.

### Observable operations

Every backup and restore should produce clear status information.

### Idempotency

Retrying an operation should not corrupt or duplicate the backup.

### Explicit failure states

Never silently ignore an error.

---

# 49. UI Design Direction

The UI should feel like a modern infrastructure/operations dashboard rather than a generic SaaS admin panel.

Preferred characteristics:

* Dark/light theme support
* Dense but readable information
* Clear status indicators
* Tables for backup history
* Charts for storage and backup trends
* Job execution timeline
* Agent health indicators
* Minimal unnecessary decoration
* Strong visual distinction between healthy/warning/failed states

Primary dashboard emphasis:

```text
HEALTH
BACKUPS
AGENTS
STORAGE
RESTORES
```

---

# 50. Success Criteria

The MVP is considered successful when the following workflow works reliably:

```text
1. Install Go agent on Windows machine.

2. Register agent from SaaS.

3. Agent appears as ONLINE.

4. Create filesystem backup job.

5. Select:
   D:\CompanyData

6. Configure:
   Daily at 11 PM

7. Enable:
   Zstandard
   AES-256-GCM

8. Configure:
   S3-compatible storage

9. Agent executes backup.

10. Backup is compressed.

11. Backup is encrypted locally.

12. Backup is uploaded directly to storage.

13. SaaS records the recovery point.

14. Dashboard reports HEALTHY.

15. User selects the recovery point.

16. User requests RESTORE.

17. Agent downloads the backup.

18. Agent verifies checksum.

19. Agent decrypts.

20. Agent decompresses.

21. Files are restored.

22. Restore is marked COMPLETED.
```

The same architecture should subsequently work for:

```text
Filesystem
DBF
SQL Server
```

---

# 51. Primary Engineering Goal

The most important engineering objective is:

> Build a reliable end-to-end backup and restore pipeline before expanding the product with billing, advanced analytics, MSP features, or additional infrastructure.

The project should demonstrate production-oriented engineering in:

* Go
* Distributed systems
* Windows services
* VSS
* SQL Server integration
* Filesystem operations
* Streaming I/O
* Compression
* Encryption
* Checksums
* Chunked uploads
* Object storage
* gRPC
* REST APIs
* PostgreSQL
* Multi-tenancy
* Monitoring
* Fault tolerance

---

# 52. Final Architecture Summary

```text
                         INTERNET
                            |
                 +----------+----------+
                 |                     |
                 v                     v
          React Dashboard          REST API
                 |                     |
                 |              Go + Gin + GORM
                 |                     |
                 |                PostgreSQL
                 |                     |
                 |               gRPC / TLS
                 |                     |
                 |                     v
                 |             Go Backup Agent
                 |                     |
                 |        +------------+------------+
                 |        |            |            |
                 |        v            v            v
                 |     SQL Server     DBF       Files/Folders
                 |        |            |            |
                 |        +------------+------------+
                 |                     |
                 |                    VSS
                 |                     |
                 |                 Compression
                 |                     |
                 |                 Encryption
                 |                     |
                 |                  SHA-256
                 |                     |
                 |                 Chunking
                 |                     |
                 |              Multipart Upload
                 |                     |
                 |        +------------+------------+
                 |        |            |            |
                 |        v            v            v
                 |       S3           R2          MinIO
                 |                                 |
                 |                              Wasabi
                 |
                 v
          Backup Monitoring
          Restore Management
          Alerts
          Audit Logs
```

# 53. Project Definition

**Product:** Backup SaaS

**Category:** B2B Backup & Recovery

**Primary Users:** SMB IT administrators, legacy software operators, MSPs

**Frontend:** React + TypeScript + Vite + Tailwind

**Backend:** Go + Gin + GORM

**Database:** PostgreSQL

**Agent:** Go Windows Service

**Communication:** REST + gRPC + WebSocket

**Backup Sources:** SQL Server, DBF, Filesystem, SMB

**Processing:** VSS + Zstandard + AES-256-GCM + SHA-256

**Storage:** S3-compatible + Local + SMB

**Deployment:** Docker + Docker Compose + Apache

**CI/CD:** GitHub Actions

**Monitoring:** Prometheus + Grafana

**Logging:** Zerolog

**Architecture:** Multi-tenant SaaS control plane + customer-side data-plane agent

**Primary product promise:**

> **Automatically back up your business databases and files, keep encrypted recovery points in your own storage, and restore them when you need them.**
