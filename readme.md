# VaultGuard - Your Simple Data Backup Assistant

Welcome to **VaultGuard**! This application is designed to help you easily protect your important data, files, and databases without needing a degree in computer science. Whether you want to back up a single folder on your computer or an entire company database, VaultGuard handles the heavy lifting for you.

## What can VaultGuard do?

VaultGuard automatically copies your important data on a schedule that you choose and stores it safely. 

Here are the main things you can do:
- **Back up regular files and folders** (like your Documents or Photos).
- **Back up business databases** (supports Microsoft SQL Server, PostgreSQL, MongoDB, and DBF files).
- **Set it and forget it**: Tell VaultGuard to run your backups every day, every week, or at any time you choose.
- **Save space**: VaultGuard can automatically compress your backups so they take up less storage space.
- **Keep it private**: You can encrypt your backups with a password. This means even if someone gets their hands on your backup file, they won't be able to read your data without your password.
- **Track history**: See a clear log of all your past backups, showing exactly what succeeded and when it happened.

## How to use VaultGuard

Using VaultGuard is as simple as following a few steps in the **New Backup Job** wizard:

1. **Choose what to back up (Source)**: 
   Select whether you want to back up a normal folder on your computer, or connect to a database (like PostgreSQL or SQL Server).
2. **Choose your worker (Agent)**: 
   Select which computer or server will perform the backup task.
3. **Pick a schedule**: 
   Decide how often you want the backup to happen (for example, "Every day at 11 PM"). You can also choose how long to keep old backups before they are automatically deleted to save space.
4. **Choose Processing Options**: 
   - *Compression*: Turn this on to shrink your backup size.
   - *Encryption*: Turn this on and set a secure password to lock your data.
5. **Review and Create**: 
   Check your settings and hit "Create Backup Job". 

That's it! Your backup job will now run automatically based on the schedule you set. 

## Managing Your Backups

On the main **Backup Jobs** dashboard, you will see a list of all your active backups. From here you can:
- **Run Now**: Click the "Play" button to force a backup to happen immediately, regardless of the schedule.
- **Enable/Disable**: Use the toggle switch to temporarily pause a backup job without deleting it.
- **View History**: Click on any backup job to see exactly when it ran in the past, how large the files were, and if any errors occurred.

## Need Help?
If a backup fails, VaultGuard will show a red warning on your dashboard. Simply click on the failed job to view the specific error message, which will usually tell you if a password was incorrect, or if the destination storage is full.

---

## How to Set Up VaultGuard (After Downloading)

If you have just downloaded VaultGuard from the internet (Git) and want to get it running on your computer, follow these simple steps. 

### 1. What You Need Installed First
Before you start, you'll need two basic pieces of free software installed on your computer:
- **Go** (for the background server): [Download Go here](https://go.dev/dl/)
- **Node.js** (for the visual dashboard): [Download Node.js here](https://nodejs.org/)

### 2. Start the Backend Server (The Brain)
The server handles the central database, scheduling, and tracking.
1. Open a terminal or command prompt (search for `cmd` or `terminal` on your computer).
2. Navigate to the VaultGuard folder, and then into the `server` folder:
   ```bash
   cd path/to/db-backup-app/server
   ```
3. Run this command to start the server:
   ```bash
   go run cmd/api/main.go
   ```
*(Leave this window open and running!)*

### 3. Start the Backup Agent (The Worker)
VaultGuard uses "Agents" to actually perform the backup work. You need at least one running!
1. Open a **new** terminal or command prompt window.
2. Navigate to the VaultGuard folder, and then into the `agent` folder:
   ```bash
   cd path/to/db-backup-app/agent
   ```
3. Run this command to start your local worker:
   ```bash
   go run cmd/agent/main.go
   ```
*(Leave this window open and running too!)*

### 4. Start the Visual Dashboard (The Interface)
Now, let's start the screens you actually click on.
1. Open a **third** terminal or command prompt window.
2. Navigate to the VaultGuard folder, and then into the `client` folder:
   ```bash
   cd path/to/db-backup-app/client
   ```
3. Type this to download the necessary pieces (you only need to do this once):
   ```bash
   npm install
   ```
4. Finally, type this to start the dashboard:
   ```bash
   npm run dev
   ```

### 5. Open the App!
Once both the Server and Dashboard are running, the command prompt will give you a website address (usually something like `http://localhost:7540` or `http://localhost:5173`). 
- Open your favorite web browser (like Chrome, Edge, or Safari).
- Type that address into the top bar.
- Welcome to VaultGuard! You can now start creating your backup jobs.
