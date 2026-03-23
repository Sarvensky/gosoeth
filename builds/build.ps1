# Скрипт сборки проекта gosoeth для Windows (PowerShell)

Write-Host "--- Запуск сборки и упаковки gosoeth (Windows PS) ---" -ForegroundColor Cyan

$version = "1.0.0"
$distDir = "./dist"
$rootDir = "../"

# 1. Очистка и создание папок
if (Test-Path "windows") { Remove-Item -Recurse -Force "windows" }
if (Test-Path "linux") { Remove-Item -Recurse -Force "linux" }
if (Test-Path $distDir) { Remove-Item -Recurse -Force $distDir }

$null = New-Item -ItemType Directory -Path "windows" -Force
$null = New-Item -ItemType Directory -Path "linux" -Force
$null = New-Item -ItemType Directory -Path $distDir -Force

# 2. Сборка под Windows (amd64)
Write-Host "Сборка: Windows (amd64)..."
$env:GOOS="windows"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"
go build -ldflags "-s -w" -o windows/gosoeth.exe "${rootDir}main.go"

# Упаковка Windows .zip
Write-Host "Упаковка: gosoeth-v$version-windows-amd64.zip..."
Copy-Item "${rootDir}config.ini.example" "windows/config.ini"
Compress-Archive -Path "windows/*" -DestinationPath "$distDir/gosoeth-v$version-windows-amd64.zip" -Force

# 3. Сборка под Linux (amd64, статическая)
Write-Host "Сборка: Linux (amd64)..."
$env:GOOS="linux"; $env:GOARCH="amd64"; $env:CGO_ENABLED="0"
go build -ldflags "-s -w" -o linux/gosoeth "${rootDir}main.go"

# Упаковка Linux .tar.gz
Write-Host "Упаковка: gosoeth-v$version-linux-amd64.tar.gz..."
Copy-Item "${rootDir}config.ini.example" "linux/config.ini"
Push-Location "linux"
tar -czf "../$distDir/gosoeth-v$version-linux-amd64.tar.gz" gosoeth config.ini
Pop-Location

# Сброс переменных
$env:GOOS=""; $env:GOARCH=""; $env:CGO_ENABLED=""

Write-Host "`n--- Все архивы готовы в папке $distDir ---" -ForegroundColor Cyan
