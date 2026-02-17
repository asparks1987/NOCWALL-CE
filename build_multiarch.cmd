@echo off
setlocal
powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\publish-local-registry.ps1" %*
exit /b %ERRORLEVEL%
