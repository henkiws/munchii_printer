@echo off
echo ============================================
echo  Munchii Printer - Build Script
echo ============================================
echo.

where go >nul 2>&1
if errorlevel 1 (
    echo ERROR: Go belum terinstall atau tidak ada di PATH
    echo Download dari: https://go.dev/dl/
    pause
    exit /b 1
)

echo Go version:
go version
echo.

if exist resource.syso del /f resource.syso
if exist *.syso del /f *.syso
if exist icon.ico del /f icon.ico

echo Downloading dependencies...
go mod tidy
if errorlevel 1 (
    echo ERROR: Gagal download dependencies
    pause
    exit /b 1
)

echo Building munchii-printer.exe...
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags="-H windowsgui -s -w" -o munchii-printer.exe .
if errorlevel 1 (
    echo.
    echo ERROR: Build gagal — lihat error di atas
    pause
    exit /b 1
)

echo.
echo ============================================
echo  BUILD SUKSES!
echo ============================================
echo.
echo Output: munchii-printer.exe
echo.
echo Yang perlu dikopi ke PC target:
echo   munchii-printer.exe   ^<-- satu file ini saja
echo.
echo PC target TIDAK perlu:
echo   - wkhtmltoimage.exe  (generate gambar di server)
echo   - Go, PHP, Laravel, atau apapun
echo.