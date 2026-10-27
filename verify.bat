@echo off
rem VaultGuard root verification wrapper -> scripts\verify.bat
call "%~dp0scripts\verify.bat" %*
exit /b %ERRORLEVEL%
