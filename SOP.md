# Standard Operating Procedure (SOP): VaultGuard Data Backup Assistant

## 1. Purpose
The purpose of this Standard Operating Procedure (SOP) is to provide comprehensive guidelines for the installation, configuration, daily operation, and troubleshooting of **VaultGuard**. This document ensures that all users and system administrators can consistently and reliably manage data backups across the organization.

## 2. Scope
This SOP covers the end-to-end usage of VaultGuard, including:
- Initial system setup and environmental prerequisites.
- Creation and scheduling of automated backup jobs.
- Application of security and optimization processing (Compression and Encryption).
- Routine monitoring and manual interventions (Run Now, Pause, Delete).
- Basic troubleshooting and log review.

## 3. Definitions and Glossary
- **VaultGuard:** The centralized data backup application consisting of a backend scheduling server and a visual frontend dashboard.
- **Source:** The origin data to be backed up. Supported types include Filesystems (directories), PostgreSQL, Microsoft SQL Server, MongoDB, and legacy DBF Datasets.
- **Agent:** The specific worker machine, server, or container responsible for executing the backup instruction against the source.
- **Storage Target:** The final destination where the securely packaged backup archive is stored (e.g., Cloud Storage, Local Network Drive).
- **Cron Expression:** A standard string format used to specify recurring schedules (e.g., `0 23 * * *` for daily at 11 PM).
- **Zstandard (Zstd):** A fast, lossless compression algorithm used to reduce backup storage footprint.
- **AES-256-GCM:** The military-grade encryption standard used to secure backup payloads before they leave the agent.

---

## 4. Prerequisites & Initial Setup

### 4.1 Required Software Dependencies
Before operating VaultGuard, ensure the host machine environment meets the following requirements:
- **Go (Golang)**: Required to compile and run the backend scheduling server and agent workers.
- **Node.js & npm**: Required to install dependencies and run the visual frontend client.

### 4.2 System Initialization Procedure
1. **Start the Backend Server:**
   - Open a terminal and navigate to the `/server` directory.
   - Execute `go run cmd/api/main.go`.
   - Ensure the terminal displays a successful startup message without fatal errors. This terminal must remain open.
2. **Start the Agent Worker:**
   - Open a secondary terminal and navigate to the `/agent` directory.
   - Execute `go run cmd/agent/main.go`.
   - The agent should successfully connect to the local server. This terminal must also remain open.
3. **Start the Frontend Dashboard:**
   - Open a third terminal and navigate to the `/client` directory.
   - Execute `npm install` (required only on first setup or after updates).
   - Execute `npm run dev` to start the local web server.
4. **Access the Application:**
   - Navigate to the provided local address (e.g., `http://localhost:7540` or `http://localhost:5173`) in a standard web browser (Chrome, Edge, Firefox).

---

## 5. Operating Procedures

### 5.1 Creating a New Backup Job
*To ensure data continuity, backups must be scheduled according to the data's criticality.*

1. Navigate to the **Backup Jobs** dashboard and click **+ New Backup Job**.
2. **Step 1: Source Configuration**
   - Provide a clear, identifiable name for the backup job (e.g., "Daily HR Database Backup").
   - Select the target Source Type (Filesystem, SQL Server, PostgreSQL, DBF Dataset).
   - Enter the exact path or connection string to the target data.
3. **Step 2: Agent Selection**
   - Select an available, `ONLINE` agent from the list. The chosen agent must have network access to the Source Data.
4. **Step 3: Schedule & Retention**
   - Select a predefined schedule (e.g., Daily at 11 PM) or input a custom Cron Expression.
   - Define the **Retention Policy** (e.g., 30 days). VaultGuard will automatically purge backups older than this threshold to conserve storage space.
5. **Step 4: Processing & Security**
   - **Compression:** Enable this toggle to apply Zstandard compression. *Recommended for all text-based databases; may be disabled for already compressed media files.*
   - **Encryption:** Enable this toggle and provide a secure passphrase to apply AES-256-GCM encryption. *Mandatory for jobs containing Personally Identifiable Information (PII).*
6. **Step 5: Review & Create**
   - Verify all configurations. Click **Create Backup Job** to finalize.

### 5.2 Routine Monitoring and Maintenance
Administrators should review the dashboard daily to verify backup integrity.

- **Dashboard Review:** The main screen displays the status of all jobs (`ONLINE`, `PAUSED`, `WARNING`, `FAILED`).
- **Manual Execution (Run Now):** For immediate data protection (e.g., prior to a major system update), click the **Run Now (Play icon)** button on the specific job row to trigger an out-of-schedule backup.
- **Pausing Jobs:** If a source system is undergoing maintenance, use the toggle switch next to the job to place it in `STANDBY`/`PAUSED` mode. This prevents scheduled runs without deleting the configuration.

### 5.3 Reviewing Job History
1. Click on the name of any backup job in the list to open the **Job Detail Page**.
2. Review the **Run History** table.
3. Verify that recent runs show a `COMPLETED` or `HEALTHY` status.
4. Monitor the **Size** column over time to predict future storage requirements and identify anomalies (e.g., sudden massive drops in backup size).

---

## 6. Troubleshooting and Incident Response

### 6.1 Failed Backup Jobs
If a job status displays as `FAILED`:
1. Navigate to the Job Detail Page.
2. Locate the failed instance in the Run History table.
3. Review the provided error logs.
   - *Common Issue:* `Connection Refused` - Verify the Source database is online and the Agent has network access.
   - *Common Issue:* `Disk Full` - Verify the Storage Target has adequate capacity; adjust the retention policy if necessary.
   - *Common Issue:* `Authentication Failed` - Update the credentials in the Source configuration.

### 6.2 Agent Offline Status
If an agent displays as `OFFLINE`:
1. The scheduled backups assigned to this agent will fail to execute.
2. Log into the host machine of the offline agent.
3. Restart the agent service and verify network connectivity back to the central VaultGuard server.

---
*Document Version: 1.0*
*Last Updated: [Current Date]*
