@echo off
rem VaultGuard root launcher wrapper -> scripts\run.bat
call "%~dp0scripts\run.bat" %*
exit /b %ERRORLEVEL%
