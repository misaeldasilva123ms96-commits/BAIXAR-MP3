@echo off
setlocal EnableExtensions
cd /d "%~dp0"
title MP3 Downloader

if exist "%~dp0MP3_Downloader.exe" (
  start "" "%~dp0MP3_Downloader.exe"
  exit /b 0
)

echo.
echo O executavel MP3_Downloader.exe nao foi encontrado.
echo Baixe o pacote Windows completo na pagina Releases.
echo Para continuar usando a versao anterior, abra Abrir_Baixador_MP3_V2.bat.
echo.
pause
exit /b 1
