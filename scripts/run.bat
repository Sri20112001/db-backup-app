@echo off
rem VaultGuard dev launcher: opens VS Code, then starts the API server and
rem the web client, each in its own console window.
rem   API    - go run ./cmd/api             http://localhost:7541
rem            (loads server\.env via godotenv; needs Postgres, e.g.:
rem             docker compose up -d postgres)
rem   Client - npm run dev  (Vite, proxy)   http://localhost:7540/vaultguard/
rem   The verification suite (build/vet/test/lint) lives in verify.bat.
rem Usage: run.bat [--no-vscode] [--no-pause]   (works from any directory)
rem   --no-vscode  skip launching VS Code
rem   --no-pause   exit instead of waiting for a keypress (for automation)
setlocal

rem Locate the repo root - the nearest ancestor holding server\go.mod - so
rem the script works whether it sits in the repo root or in scripts\.
set "ROOT=%~dp0"
if exist "%ROOT%server\go.mod" goto root_found
if exist "%ROOT%..\server\go.mod" goto root_parent
echo ERROR: repo root not found - no server\go.mod above %ROOT%
exit /b 1
:root_parent
set "ROOT=%ROOT%.."
:root_found
for %%I in ("%ROOT%") do set "ROOT=%%~fI"
if not "%ROOT:~-1%"=="\" goto root_ok
set "ROOT=%ROOT:~0,-1%"
:root_ok
cd /d "%ROOT%" || exit /b 1

set "NO_PAUSE="
set "NO_VSCODE="
if /i "%~1"=="--no-pause" set "NO_PAUSE=1"
if /i "%~2"=="--no-pause" set "NO_PAUSE=1"
if /i "%~1"=="--no-vscode" set "NO_VSCODE=1"
if /i "%~2"=="--no-vscode" set "NO_VSCODE=1"

rem --- Prerequisites: fail fast, flat ifs only (no parenthesized blocks) ---
where go >nul 2>&1
if not errorlevel 1 goto have_go
echo ERROR: Go toolchain not found on PATH.
exit /b 1
:have_go
where npm >nul 2>&1
if not errorlevel 1 goto have_npm
echo ERROR: npm not found on PATH.
exit /b 1
:have_npm

rem --- Open VS Code first so it is ready while the apps start ---
rem call code.cmd --new-window: explicit .cmd (an extensionless 'code' also
rem sits on the same PATH entry), 'call' so the CLI cannot abandon this
rem batch frame, --new-window so the project gets its own window.
if defined NO_VSCODE goto skip_vscode
where code.cmd >nul 2>&1
if errorlevel 1 goto no_vscode
call code.cmd --new-window "%ROOT%"
echo [INFO] Opened VS Code at: %ROOT%
goto skip_vscode
:no_vscode
echo [WARN] 'code.cmd' CLI not found on PATH - skipping VS Code.
:skip_vscode

rem --- API server in its own window (cmd /k keeps it open for the logs) ---
rem Inner quotes survive cmd /k's strip-first-and-last-quote rule.
start "VaultGuard API" cmd /k "cd /d "%ROOT%\server" && go run ./cmd/api"
echo [INFO] API server started - http://localhost:7541  (window: VaultGuard API)

rem --- Web client (Vite) in its own window ---
start "VaultGuard Client" cmd /k "cd /d "%ROOT%\client" && npm run dev"
echo [INFO] Web client started - http://localhost:7540/vaultguard/  (window: VaultGuard Client)

echo.
echo Both apps run in their own windows - Ctrl+C there stops that process.
echo The first API start compiles the server, so give it a few seconds.
echo.
if /i "%~1"=="--no-pause" exit /b 0
if /i "%~2"=="--no-pause" exit /b 0
pause
exit /b 0
