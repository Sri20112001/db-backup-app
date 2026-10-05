# Windows Agent — Validation Guide

Validates the complete customer installation flow on a real Windows machine.
Run top to bottom; every step has an expected result. Anything marked
**[WINDOWS]** was NOT executable on headless/Linux CI and needs this machine.

Prerequisites: VaultGuard server reachable over HTTPS (or loopback HTTP for
lab), Owner/Admin dashboard access, local administrator on the test machine.

## Build artifacts (exact commands)

```powershell
# Headless agent — pure Go, builds anywhere (Linux Jenkins included):
#   (from agent/) GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o dist/windows/VaultGuard-Agent.exe ./cmd/agent

# Setup GUI — needs network (module downloads) + a C toolchain + Fyne deps.
# Native Windows build (on the Windows build agent):
cd setup
go mod tidy
go build -o VaultGuard-Agent-Setup.exe .
```

Embed the UAC manifest so elevation is automatic (needs `rsrc`,
`go install github.com/akavel/rsrc@latest` — network required):

```powershell
rsrc -manifest VaultGuard-Agent-Setup.exe.manifest -o setup.syso
go build -o VaultGuard-Agent-Setup.exe .
```

Without the embedded manifest, right-click → **Run as administrator**
(the installer's `RequireAdmin` preflight fails fast with that instruction).

Ship both files side by side — the GUI installs the agent EXE found next
to itself. The setup binary is generic: no customer/token data inside.

## Checklist

### A. Fresh install
1. Dashboard → Agents → **Add Agent** (name optional) → copy token + note expiry.
2. Run `VaultGuard-Agent-Setup.exe` → expect the **Windows UAC prompt**. If no
   prompt and no manifest embedded, right-click → Run as administrator.
3. Welcome → Continue → enter Server URL + token → Register.
4. Expect: Connecting ✓ → token validated → machine registered →
   Installing (dirs → binary → config → credential → service → start) →
   Complete shows Computer + Agent ID + Running. **Finish exits; no GUI stays.**

### B. Enrollment edge cases (each: fresh token per attempt where noted)
- **M. Invalid token** (random string) → `enrollment token invalid or expired`, back to Connect, token entry wiped.
- **N. Expired token** (generate, wait past expiry or craft short TTL) → same 401 path.
- **O. Reused token** (enroll once, try again) → `401 invalid enrollment token`.
- Cross-org: token is bound to its org at lookup — another org's dashboard never lists the agent.

### C. Service verification (elevated PowerShell)
```powershell
Get-Service VaultGuardAgent            # Status: Running, StartType: Automatic
Get-CimInstance Win32_Service -Filter "Name='VaultGuardAgent'" | Select PathName
# PathName must point at C:\Program Files\VaultGuard\Agent\VaultGuard-Agent.exe -config "C:\ProgramData\VaultGuard\Agent\config.json"
& 'C:\Program Files\VaultGuard\Agent\VaultGuard-Agent.exe' -status
& 'C:\Program Files\VaultGuard\Agent\VaultGuard-Agent.exe' -stop   # → Stopped
& 'C:\Program Files\VaultGuard\Agent\VaultGuard-Agent.exe' -start  # → Running
& 'C:\Program Files\VaultGuard\Agent\VaultGuard-Agent.exe' -restart
```
- `C:\ProgramData\VaultGuard\Agent\` contains `config.json` (server_url, hostname, agent_id — **no token/password**), credential file (opaque DPAPI blob), `agent.log` (no secrets in log lines).

### D. Reboot + E. Reconnect
Reboot → service starts automatically → within ~1 min dashboard shows **ONLINE**. No Setup re-run, no token re-entry.

### F–H. Connection → Backup → Restore
Per existing Connections workflow: create Connection on the new agent →
Backup Job with `connection_id` → run → success; restore reuses the same
connection (server copies `connection_id` onto the restore; agent resolves
credentials per-claim; no env credentials needed).

### I. Revocation
Dashboard → agent → **Revoke** → agent shows **REVOKED**. Agent log shows the
revoked stop message; service shows **Stopped** (safe state, no restart loop).
Heartbeat/claims/restores/gRPC all reject (`401 agent revoked`). Record kept
for audit. Re-enroll path: delete local credential file → fresh token → setup.

### J. Uninstall
`VaultGuard-Agent.exe -uninstall` (elevated) or installer Uninstall: service
removed, binary + local credential removed, `config.json` + logs **preserved**,
server record untouched (dashboard → OFFLINE unless revoked). Document which
files remain.

### K. Reinstall / upgrade
Run newer Setup.exe **without** a token (credential present): binary updated,
config refreshed, **same Agent ID**, service restarted. With a token: brand-new
agent (old record goes stale → OFFLINE; revoke it if retired).

### L. Server unavailable
Stop the server: agent logs heartbeat failures at poll cadence (no crash, no
spam burst), keeps running, reconnects on server return. Never self-marks
REVOKED (only an explicit server rejection does that).

## Validation results (acceptance run 2026-10-04, Windows box, non-elevated, offline)

Result vocabulary: **PASS** = actually executed here · **FAIL** = executed, broken ·
**BLOCKED** = cannot execute in this environment (reason given) ·
**NOT TESTED** = needs a different machine/state (server, second VM, cert).

| Test | Result | Evidence / Notes |
|------|--------|------------------|
| Agent binary builds (amd64) | **PASS** | `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath ./cmd/agent` → 19 MB exe; PE Machine field `0x8664` read from header |
| Release artifact gate | **PASS** | `go run ./cmd/verify-artifact --arch amd64 --min-bytes 5000000 dist/windows/VaultGuard-Agent.exe` → OK; rejects wrong-arch and truncated files (both negative cases executed). `agent/dist/windows/` is the only distributable source (`*.exe`, `/dist/`, `agent/tmp/` all git-ignored; `tmp/` dev artifacts never ship) |
| Exe launches, `-status` | **PASS** | Prints service state / credential presence / config path, exit 0, no secrets in output (agent ID only) |
| `-install` failure mode | **PASS** | Non-elevated → `install: connect to service manager: Access is denied.`, exit 1, `sc query` 1060 (no residue) |
| Binary secret/path scan | **PASS** | 47,936 strings extracted: no dev paths (`C:\Users\…`, `/home/`, repo path), no AKIA/private-key material; only expected identifiers (API routes, env names, service strings) |
| DPAPI roundtrip (both scopes) | **PASS** | `TestDPAPIRoundtrip` + `TestDPAPIStoreRoundtrip` executed on real Windows (no admin needed) |
| DPAPI at-rest opacity + adoption | **PASS** | `TestDPAPIStoreAdoptsPlaintext` incl. self-healing re-protect |
| `RequireAdmin` preflight | **PASS** | Detects non-elevated, actionable message (`TestRequireAdminMessage`) |
| Installer rollback | **PASS** | `TestFreshInstallRollback`: failed service step → binary+config removed, credential preserved, error reports rollback without token |
| Upgrade preserves ID/credential | **PASS** | `TestUpgradeNeedsCredential`, `TestUpgradePreservesCredential` |
| Revoked latch (no retry loop) | **PASS** | `TestIsRevokedError` incl. non-revocation negatives |
| Server unit tests | **PASS** | `go test -count=1 ./...` all pkgs ok (DB-gated integration tests skip: no Postgres/Docker here) |
| Agent unit tests | **PASS** | `go test -count=1 ./...` all pkgs ok |
| `go vet` both modules × both GOOS | **PASS** | Clean (native + `GOOS=windows GOARCH=amd64`) |
| Frontend build | **PASS** | `npm run build` (tsc + vite) succeeds |
| Fyne setup build | **BLOCKED** | No network (module downloads) + no C toolchain: `missing go.sum entry for fyne.io/fyne/v2` — needs Windows build agent |
| Service install/start/stop | **BLOCKED** | Not elevated on this box (proven by the clean `-install` denial above) |
| UAC prompt UX | **BLOCKED** | Needs manifest-embedded GUI + elevation (manifest file provided, embedding needs `rsrc` = network) |
| Reboot survival | **BLOCKED** | Needs installed service + reboot on a test VM |
| SYSTEM-context DPAPI decrypt | **BLOCKED** | Crypto proven (above); service-account context needs Phase 5 on a VM |
| Enrollment E2E (token lifecycle A–F) | **BLOCKED** | Needs Postgres-backed server (no Docker here); concurrency race fixed by inspection, needs DB test |
| Connection/Backup/Restore E2E | **BLOCKED** | Needs server + agent + database on a VM |
| Revocation E2E + restart | **BLOCKED** | Needs server + installed service |
| Server-down behavior | **BLOCKED** | Needs server to stop/start (runner code: fixed 30 s poll cadence, no crash path, revocation-only stop) |
| Fresh-machine VM test | **NOT TESTED** | Needs second VM (procedure in checklist A–O above) |
| Code signing | **BLOCKED** | No certificate (documented `signtool` step, not executed, binary NOT signed) |

### Bugs discovered during validation
1. **Plaintext credential at `C:\ProgramData\VaultGuard\Agent\agent-credential.json`** (132 B, created mid-session window; origin unattributed — not writable by any test path in the current code). **Fixed class:** `DPAPIStore.Load` now self-heals — adopts the plaintext credential and immediately re-protects it at rest (tested). File left untouched on this box (not mine to delete; may be someone's live credential).
2. `svc.State` has no `String()` in x/sys v0.23.0 — fixed with explicit state mapper (found via cross-compile).
3. Test asserted Unix perms on Windows (informative failure) — made platform-aware with documented rationale.

### Security findings (this run)
- At-rest scan of release binary: no secrets, no dev paths (trimpath verified).
- New-code log audit (grep over agent/setup/server handlers): no token/password/secret/Authorization logging.
- `-status` output leaks nothing (agent ID is non-secret by design).
- Pre-existing plaintext credential file (item 1 above) remediated by code; box hygiene is the owner's action.

## Not validated (needs the checklist above)
Service SCM install/start/stop, UAC prompt UX, Fyne GUI compile + screens,
reboot survival, real HTTPS enrollment E2E, Postgres-backed concurrency.

## Build environment matrix (this machine, verified)

```text
MSVC / cl.exe       ✅ present (not cgo-compatible — Go needs gcc-style flags)
Go 1.26.x           ✅
Fyne modules        ❌ not cached
MinGW/clang         ❌
Network             ❌
```

Fyne source is NOT the blocker; the environment is. Both network AND a
gcc-compatible toolchain are required. MSVC alone changes nothing for cgo.

## Artifact classification

```text
Development / acceptance (buildable here)
├── VaultGuard-Agent-Setup-Console.exe   ← proves installer/controller/service logic
└── VaultGuard-Agent.exe                 ← release-build ready, gated

Release (needs networked Windows build agent)
├── VaultGuard-Agent-Setup.exe           ← Fyne + UAC manifest + signed
└── VaultGuard-Agent.exe                 ← gated + signed
```

The console installer is NOT a customer replacement for the Fyne GUI —
it validates the engine the GUI will drive. Full Windows RC stays blocked
pending: native Fyne build → manifest embed → sign → service/UAC/E2E (A–O).

## CI/CD split
- **Linux Jenkins (today):** backend/agent/frontend tests, headless Windows
  agent cross-compile (`Build (agent windows)` stage). Fyne GUI is NOT built
  here (needs C toolchain + graphics backends + network).
- **Windows build agent (future infra):** checkout → `go mod tidy` →
  `rsrc` manifest embed → build GUI → Authenticode sign + timestamp →
  `dist/windows/` package → release. Sketch only — no pipeline changes made
  that could break the current Linux node.

## Code signing prep (future, no cert available)
Signing happens on the Windows agent AFTER build, BEFORE release:
`signtool sign /fd SHA256 /tr http://timestamp.digicert.com /td SHA256 /a
VaultGuard-Agent-Setup.exe` (same for the agent EXE). Optional now:
stamp `versioninfo.json` via `goversioninfo` for Add/Remove-Programs metadata.
