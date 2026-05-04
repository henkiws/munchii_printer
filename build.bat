@echo off
echo ============================================
echo  Munchii Printer - Build Script
echo ============================================
echo.

REM Check Go installation
where go >nul 2>&1
if errorlevel 1 (
    echo ERROR: Go is not installed or not in PATH
    echo Download Go from: https://go.dev/dl/
    pause
    exit /b 1
)

echo Go version:
go version
echo.

REM Clean any leftover resource files that could corrupt the build
if exist resource.syso del /f resource.syso
if exist *.syso del /f *.syso
if exist icon.ico del /f icon.ico

REM Download dependencies
echo Downloading dependencies...
go mod tidy
if errorlevel 1 (
    echo ERROR: Failed to download dependencies
    pause
    exit /b 1
)

REM Build - explicitly target Windows amd64, no CGO
echo Building munchii-printer.exe...
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags="-H windowsgui -s -w" -o munchii-printer.exe .
if errorlevel 1 (
    echo.
    echo ERROR: Build failed - see errors above
    pause
    exit /b 1
)

echo.
echo ============================================
echo  BUILD SUCCESS!
echo ============================================
echo.
echo Output: munchii-printer.exe
echo.
echo Next steps:
echo  1. Copy munchii-printer.exe to your desired folder
echo  2. Also copy wkhtmltoimage.exe to the SAME folder
echo     Download from: https://wkhtmltopdf.org/downloads.html
echo  3. Run munchii-printer.exe
echo  4. Right-click the tray icon to manage printers
echo.
pause