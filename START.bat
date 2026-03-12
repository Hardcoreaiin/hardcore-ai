@echo off
title HardcoreAI Master Launcher
echo ==================================================
echo   HARDCORE AI - PROFESSIONAL LEARNING PLATFORM
echo ==================================================
echo.

set "HC_ROOT=%~dp0"

:: [1/2] Starting Backend Server
echo [1/2] Starting Backend Server on Port 8003...
start "HardcoreAI Backend" /min cmd /k "cd /d "%HC_ROOT%Hardcore.ai\orchestrator" && python server.py"

:: Wait for backend to initialize
echo Waiting for backend...
timeout /t 3 /nobreak > nul

:: [2/2] Starting Frontend (Electron + Vite)
echo [2/2] Starting Frontend Interface...
start "HardcoreAI Frontend" /min cmd /k "cd /d "%HC_ROOT%hardcore-desktop" && npm run dev"

echo.
echo ==================================================
echo   SYSTEM READY
echo ==================================================
echo   - Backend API: http://localhost:8003/docs
echo   - Frontend: Running in Electron / http://localhost:5173
echo.
echo Please allow a few seconds for the UI to load.
echo Close this window to keep servers running in background.
echo.
pause
