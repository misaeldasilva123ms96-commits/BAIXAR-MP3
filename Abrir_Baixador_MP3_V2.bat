@echo off
chcp 65001 >nul
setlocal
cd /d "%~dp0"
title Baixador MP3 V2

powershell.exe -NoLogo -NoProfile -ExecutionPolicy Bypass -File "%~dp0Baixador_MP3_V2.ps1"
set "EXIT_CODE=%ERRORLEVEL%"

if not "%EXIT_CODE%"=="0" (
    echo.
    echo O programa terminou inesperadamente.
    echo Consulte ultimo_download.log ou a pasta logs.
    echo.
    pause
)

exit /b %EXIT_CODE%
