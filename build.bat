@echo off
REM Build script for twitch-multi-tool Windows executable
REM Requires: Go 1.24.2 or higher

setlocal enabledelayedexpansion

echo Building twitch-multi-tool...
echo.

REM Check if Go is installed
go version >nul 2>&1
if errorlevel 1 (
    echo Error: Go is not installed or not in PATH
    echo Please install Go from https://golang.org/dl/
    exit /b 1
)

REM Check if we're in the right directory
if not exist "go.mod" (
    echo Error: go.mod not found
    echo Please run this script from the project root directory
    exit /b 1
)

REM Build variables
set OUTPUT=twitch-multi-tool.exe

REM Clean old build
if exist %OUTPUT% (
    echo Removing old build...
    del %OUTPUT%
)

REM Download dependencies
echo Downloading dependencies...
call go mod download
if errorlevel 1 (
    echo Failed to download dependencies
    exit /b 1
)

REM Build
echo Compiling...
call go build -o %OUTPUT%

if errorlevel 1 (
    echo.
    echo Build failed!
    exit /b 1
)

echo.
echo ========================================
echo Build successful!
echo Output: %OUTPUT%
echo ========================================
echo.
echo Starting twitch-multi-tool web server...
echo Browser will open automatically...
echo.
timeout /t 2 /nobreak
start http://127.0.0.1:8080

REM Launch the executable with web flag and default channel
%OUTPUT% web --channel general --listen 127.0.0.1:8080

endlocal
pause
