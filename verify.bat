@echo off
rem VaultGuard developer checks: opens VS Code, then runs every verification
rem suite from the status report:
rem   server\  go build ./...    go vet ./...    go test ./...
rem   client\  npm run lint      npm run test    npm run build
rem Usage: verify.bat [--no-vscode] [--no-pause]   (works from any directory)
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
rem %~dp0 always ends in a backslash - strip it so "%ROOT%\server" stays
rem single-separator and a quoted argument like "%ROOT%" cannot end in \"
rem (which would escape its own closing quote in argv parsing).
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

set /a STEPS=0
set /a FAILS=0

rem --- Open VS Code first so it is ready while the checks run ---
rem call code.cmd --new-window is deliberate: explicit .cmd (an extensionless
rem 'code' also sits on the same PATH entry), 'call' so the CLI cannot abandon
rem this batch frame the way npm.cmd would, and --new-window so the project
rem always gets its own window rather than folding into an existing one.
if defined NO_VSCODE goto skip_vscode
where code.cmd >nul 2>&1
if errorlevel 1 goto no_vscode
call code.cmd --new-window "%ROOT%"
echo [INFO] Opened VS Code at: %ROOT%
goto skip_vscode
:no_vscode
echo [WARN] 'code.cmd' CLI not found on PATH - skipping VS Code.
:skip_vscode

echo.
echo === [1/6] server: go build ./...
echo     go build ./...
call :step "%ROOT%\server" go build ./...

echo.
echo === [2/6] server: go vet ./...
echo     go vet ./...
call :step "%ROOT%\server" go vet ./...

echo.
echo === [3/6] server: go test ./...
echo     go test ./...
call :step "%ROOT%\server" go test ./...

echo.
echo === [4/6] client: lint
echo     npm run lint
call :step "%ROOT%\client" npm run lint

echo.
echo === [5/6] client: tests
echo     npm run test
call :step "%ROOT%\client" npm run test

echo.
echo === [6/6] client: build (tsc + vite)
echo     npm run build
call :step "%ROOT%\client" npm run build

echo.
if not "%FAILS%"=="0" goto summary_fail
echo === SUMMARY: all %STEPS% steps passed ===
set "RC=0"
goto finish
:summary_fail
echo === SUMMARY: %FAILS% of %STEPS% step(s) FAILED - see output above ===
set "RC=1"
:finish
if /i "%~1"=="--no-pause" exit /b %RC%
if /i "%~2"=="--no-pause" exit /b %RC%
pause
exit /b %RC%

rem ---------------------------------------------------------------------------
rem :step <workdir> <command...> - run one command, report PASS/FAIL, keep going.
rem Flat code on purpose: parenthesized blocks break on PATH values like (x86).
:step
pushd "%~1"
if errorlevel 1 goto step_failed
rem call is required: npm is npm.cmd and would otherwise take over the frame.
call %2 %3 %4 %5 %6 %7 %8 %9
set "RC=%ERRORLEVEL%"
popd
set /a STEPS=%STEPS%+1
if not "%RC%"=="0" goto step_failed
echo        PASS
exit /b 0
:step_failed
set /a FAILS=%FAILS%+1
echo        FAIL
exit /b 1
