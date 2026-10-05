@echo off
rem VaultGuard Windows release build.
rem   Builds: headless agent + console setup (pure Go, anywhere)
rem           Fyne setup GUI (needs network for modules + MinGW-w64 gcc for cgo)
rem   Output: dist\windows\VaultGuard-Agent.exe
rem           dist\windows\VaultGuard-Agent-Setup-Console.exe
rem           dist\windows\VaultGuard-Agent-Setup.exe  (Fyne GUI + UAC manifest)
rem Usage: build-windows.bat  (run from the repository root)
setlocal EnableDelayedExpansion

set "ROOT=%~dp0"
rem %~dp0 ends with a backslash, which would escape the closing quote below.
if "%ROOT:~-1%"=="\" set "ROOT=%ROOT:~0,-1%"
cd /d "%ROOT%" || exit /b 1

where go >nul 2>&1
if errorlevel 1 (
  echo ERROR: Go toolchain not found on PATH.
  exit /b 1
)

rem --- C toolchain for the Fyne GUI (MinGW-w64; MSVC cl.exe is NOT cgo-compatible) ---
rem NOTE: flat single-line ifs + goto on purpose. Nested parenthesized blocks
rem are fragile: PATH values contain parentheses like (x86), and stray ( ) in
rem echo text, both of which corrupt block parsing. Do NOT nest ( ) here.
where gcc >nul 2>&1
if not errorlevel 1 goto have_gcc
if not exist "C:\winlibs\mingw64\bin\gcc.exe" goto no_gcc
set "PATH=C:\winlibs\mingw64\bin;%PATH%"
goto have_gcc
:no_gcc
echo ERROR: gcc not found. Install MinGW-w64, e.g.:
echo   winget install BrechtSanders.WinLibs
echo NOTE: MSVC cl.exe cannot serve as Go's cgo compiler.
exit /b 1
:have_gcc

set CGO_ENABLED=1
set GOOS=windows
set GOARCH=amd64
if not exist "dist\windows" mkdir "dist\windows"

echo === [1/4] headless agent ===
cd agent
go build -trimpath -o ..\dist\windows\VaultGuard-Agent.exe ./cmd/agent
if errorlevel 1 exit /b 1
go build -trimpath -o ..\dist\windows\VaultGuard-Agent-Setup-Console.exe ./cmd/agent-setup
if errorlevel 1 exit /b 1
go run ./cmd/verify-artifact --arch amd64 --min-bytes 5000000 ..\dist\windows\VaultGuard-Agent.exe
if errorlevel 1 exit /b 1
cd ..

echo === [2/4] Fyne setup GUI ===
cd setup
rem -H=windowsgui: GUI subsystem, so no console window opens beside the app.
rem (Console setup intentionally keeps the default console subsystem.)
go build -trimpath -ldflags "-H=windowsgui" -o ..\dist\windows\VaultGuard-Agent-Setup.exe .
if not errorlevel 1 goto gui_built
echo HINT: first build needs network for go mod tidy - Fyne modules.
exit /b 1
:gui_built
cd ..

echo === [3/4] UAC manifest (mt.exe replaces MinGW default asInvoker) ===
rem A running GUI locks the exe and mt.exe fails cryptically - check first.
tasklist /FI "IMAGENAME eq VaultGuard-Agent-Setup.exe" 2>nul | find /I "VaultGuard-Agent-Setup.exe" >nul
if not errorlevel 1 goto gui_running
goto mt_start
:gui_running
echo ERROR: VaultGuard-Agent-Setup.exe is currently running - close all Setup windows first, then re-run.
exit /b 1
:mt_start
set MT="%ProgramFiles(x86)%\Windows Kits\10\bin\10.0.26100.0\x64\mt.exe"
if not exist %MT% (
  for /f "delims=" %%M in ('where mt.exe 2^>nul') do set MT="%%M"
)
if not exist %MT% (
  echo WARNING: mt.exe not found - GUI will lack requireAdministrator elevation.
  echo Install Windows SDK w/ MSVC Build Tools, then re-run.
) else (
  set MT_TRIES=0
  goto mt_retry
)
goto mt_done
:mt_retry
%MT% -manifest setup\VaultGuard-Agent-Setup.exe.manifest "-outputresource:dist\windows\VaultGuard-Agent-Setup.exe;#1"
if not errorlevel 1 goto mt_done
set /a MT_TRIES+=1
if %MT_TRIES% GEQ 3 goto mt_giveup
echo mt.exe locked, transient AV scan likely - waiting 15s, retry %MT_TRIES% of 3...
timeout /t 15 /nobreak >nul
goto mt_retry
:mt_giveup
echo ERROR: mt.exe failed 3 times - close any running Setup and re-run.
exit /b 1
:mt_done

echo === [4/4] gate release artifacts ===
cd agent
go run ./cmd/verify-artifact --arch amd64 --min-bytes 100000 ..\dist\windows\VaultGuard-Agent-Setup-Console.exe
if errorlevel 1 exit /b 1
cd ..
for %%F in (dist\windows\VaultGuard-Agent.exe dist\windows\VaultGuard-Agent-Setup-Console.exe dist\windows\VaultGuard-Agent-Setup.exe) do (
  if not exist "%%F" (
    echo ERROR: expected artifact missing: %%F
    exit /b 1
  )
)

echo.
echo BUILD OK:
dir /b dist\windows\*.exe
endlocal
