@echo off
setlocal
echo [VaultGuard] Compiling Go Wasm module...
cd /d "%~dp0..\wasm"
set GOOS=js
set GOARCH=wasm
go build -trimpath -ldflags="-s -w" -o "..\client\public\vaultguard.wasm" .
if %ERRORLEVEL% equ 0 (
    echo [VaultGuard] Successfully built client\public\vaultguard.wasm
) else (
    echo [VaultGuard] Failed to build wasm binary.
    exit /b %ERRORLEVEL%
)
endlocal
