@echo off
REM Build script for twitch-multi-tool Windows executable
REM Требуется: Go 1.24.2 или выше

setlocal enabledelayedexpansion

echo Building twitch-multi-tool...

REM Проверка наличия Go
go version >nul 2>&1
if errorlevel 1 (
    echo Error: Go is not installed or not in PATH
    exit /b 1
)

REM Переменные сборки
set OUTPUT=twitch-multi-tool.exe
set LDFLAGS=-ldflags="-s -w"

REM Очистка старой сборки
if exist %OUTPUT% (
    echo Removing old build...
    del %OUTPUT%
)

REM Сборка
echo Compiling...
go build %LDFLAGS% -o %OUTPUT%

if errorlevel 1 (
    echo Build failed!
    exit /b 1
)

echo.
echo Build successful!
echo Output: %OUTPUT%
echo.
echo Usage:
echo   %OUTPUT% accounts
echo   %OUTPUT% web --channel ^<channel^>
echo   %OUTPUT% live --account ^<name^> --channel ^<channel^>
echo   %OUTPUT% send --account ^<name^> --channel ^<channel^> --message "text"

endlocal
