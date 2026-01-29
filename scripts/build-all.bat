@echo off
REM Build script for Windows

setlocal enabledelayedexpansion

set VERSION=%1
if "%VERSION%"=="" set VERSION=v1.0.0

set OUTPUT_DIR=dist

echo Building Xray Panel %VERSION%
echo ================================

REM Clean output directory
if exist "%OUTPUT_DIR%" rmdir /s /q "%OUTPUT_DIR%"
mkdir "%OUTPUT_DIR%"

REM Build for different platforms
call :build linux amd64
call :build linux arm64
call :build windows amd64
call :build windows arm64
call :build darwin amd64
call :build darwin arm64

echo.
echo ================================
echo Build Summary:
dir /b "%OUTPUT_DIR%"
echo.
echo Build completed!
goto :eof

:build
set GOOS=%1
set GOARCH=%2
set OUTPUT_NAME=panel-%GOOS%-%GOARCH%
if "%GOOS%"=="windows" set OUTPUT_NAME=%OUTPUT_NAME%.exe

echo.
echo Building %GOOS%/%GOARCH%...

set CGO_ENABLED=0
go build -v -trimpath -ldflags="-s -w -X main.Version=%VERSION%" -o "%OUTPUT_DIR%\%OUTPUT_NAME%" ./cmd/panel

if %errorlevel% equ 0 (
    echo [OK] Built %OUTPUT_NAME%
) else (
    echo [FAIL] Failed to build %OUTPUT_NAME%
    exit /b 1
)
goto :eof
