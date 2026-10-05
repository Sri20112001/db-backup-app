# VaultGuard Agent Setup (Fyne GUI + console fallback)

Two frontends, one shared core. **All logic lives in the agent module**
(`../agent/setup`: controller, enrollment client, installer, URL validation)
and is built + tested there with stdlib only.

- **Fyne GUI** (`main.go`, `ui/`): the customer-facing installer. Requires
  network (module downloads) + a C toolchain — see below.
- **Console setup** (`../agent/cmd/agent-setup`): same controller, stdin/stdout
  prompts. Builds anywhere with no extra dependencies — use it wherever the
  GUI can't be built yet. Token entry echoes (no masked input without extra
  deps); the token is single-use and never stored.

## Layout

```text
setup/
  main.go        # Fyne app bootstrap (headless agent is NOT here)
  ui/wizard.go   # 5 screens: Welcome → Connect → Registering → Installing → Complete
  go.mod         # requires fyne + agent module (replace ../agent)
```

## Build requirements

- Network access (`go mod download` for Fyne; Fyne is NOT vendored).
- A C toolchain (Fyne uses cgo + platform graphics backends).

```sh
cd setup
go mod tidy        # first time / after dep changes (needs network)
go build -o VaultGuard-Agent-Setup.exe .
```

## Windows release layout (shipped together)

```text
dist/windows/
  VaultGuard-Agent.exe        # headless agent (built from ../agent)
  VaultGuard-Agent-Setup.exe  # this GUI
```

Build the agent for Windows (pure Go, works from any OS):

```sh
cd ../agent
GOOS=windows GOARCH=amd64 go build -o ../dist/windows/VaultGuard-Agent.exe ./cmd/agent
```

The GUI looks for the agent binary next to itself and installs it to
`C:\Program Files\VaultGuard\Agent\`; config + credential go to
`C:\ProgramData\VaultGuard\Agent\`.

## Windows GUI builds

Fyne cross-compilation (Linux → Windows GUI) needs a Windows-capable C
toolchain and is **not** supported on the current Linux Jenkins node — GUI
packaging requires a **Windows build agent** (future infra). The Linux
Jenkins pipeline covers: agent logic tests, agent Windows cross-compile
(pure Go, no C), backend, and frontend.

## Running locally (dev, no install)

```sh
# 1. Dashboard → Agents → Add Agent → copy the enrollment token.
# 2. From the repo root with the stack running:
cd setup
go run .    # needs network + C toolchain (see above)
# 3. Enter server URL (https://… or http://localhost:…) + token → Register.
```

Without network/C toolchain, exercise the same code path headlessly:

```sh
cd ../agent
AGENT_SERVER=http://localhost:7541/vaultguard/api \
AGENT_ENROLLMENT_TOKEN=<token> \
AGENT_CREDENTIAL_DIR=$PWD/.tmp-creds \
go run ./cmd/agent   # enrolls once, then polls (Ctrl-C to stop)
```

## End-to-end enrollment test

1. `POST /organizations/:org_id/agents/enrollment-token` (Owner/Admin)
   → `{agent_id, enrollment_token, expires_at}` (token shown once).
2. Run setup GUI or the headless command above with that token.
3. Dashboard → Agents shows the machine **Online** (heartbeat ≤ 3 min).
4. Reusing the token → `401 invalid enrollment token` (single-use).
5. `POST /organizations/:org_id/agents/:id/revoke` → agent shows
   **Revoked**; further heartbeat/claims → `401 agent revoked`.

## Security notes

- The enrollment token is single-use, short-lived (default 30 min), stored
  as SHA-256, never logged, never in error strings.
- The permanent credential is stored via `CredentialStore` (0600 file;
  DPAPI opt-in on Windows via `AGENT_CREDENTIAL_STORE=dpapi`) — never in
  `config.json`, which holds only server URL + hostname.
- The GUI never displays any secret; completion screen shows machine name
  and running status only.
