@echo off
setlocal

set PROJECT_DIR=%~dp0
set BACKUP_DIR=%USERPROFILE%\Desktop\backup-saas-backups
set TIMESTAMP=%DATE:~10,4%%DATE:~4,2%%DATE:~7,2%_%TIME:~0,2%%TIME:~3,2%%TIME:~6,2%
set TIMESTAMP=%TIMESTAMP: =0%
set BACKUP_FILE=%BACKUP_DIR%\backup-saas_%TIMESTAMP%.zip

if not exist "%BACKUP_DIR%" mkdir "%BACKUP_DIR%"

echo Creating backup: %BACKUP_FILE%

powershell -Command "
  $source = '%PROJECT_DIR%';
  $dest = '%BACKUP_FILE%';
  $files = Get-ChildItem -Path $source -Recurse -File | Where-Object { $_.FullName -notlike '*\node_modules\*' };
  Compress-Archive -Path $files.FullName -DestinationPath $dest -Force
"

if %ERRORLEVEL% == 0 (
    echo Backup completed successfully: %BACKUP_FILE%
) else (
    echo Backup failed.
    exit /b 1
)

endlocal
