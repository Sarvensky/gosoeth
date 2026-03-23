# Скрипт сборки проекта gosoeth для Windows (PowerShell)
# Можно запускать из любой директории: .\builds\build.ps1

Write-Host "--- Запуск сборки и упаковки gosoeth (Windows PS) ---" -ForegroundColor Cyan

$version = "1.1.0"
$scriptDir = $PSScriptRoot
$rootDir = (Resolve-Path "$scriptDir/..").Path
$distDir = "$scriptDir/dist"
$winDir = "$scriptDir/windows"
$linDir = "$scriptDir/linux"

# 1. Очистка и создание папок
if (Test-Path $winDir) { Remove-Item -Recurse -Force $winDir }
if (Test-Path $linDir) { Remove-Item -Recurse -Force $linDir }
if (Test-Path $distDir) { Remove-Item -Recurse -Force $distDir }

$null = New-Item -ItemType Directory -Path $winDir -Force
$null = New-Item -ItemType Directory -Path $linDir -Force
$null = New-Item -ItemType Directory -Path $distDir -Force

# 2. Сборка под Windows (amd64)
Write-Host "Сборка: Windows (amd64)..."
$env:GOOS = "windows"; $env:GOARCH = "amd64"; $env:CGO_ENABLED = "0"
Push-Location $rootDir
go build -ldflags "-s -w" -o "$winDir/gosoeth.exe" .
Pop-Location

# Упаковка Windows .zip
Write-Host "Упаковка: gosoeth-v$version-windows-amd64.zip..."
Copy-Item "$rootDir/config.ini.example" "$winDir/config.ini"
Compress-Archive -Path "$winDir/*" -DestinationPath "$distDir/gosoeth-v$version-windows-amd64.zip" -Force

# 3. Сборка под Linux (amd64, статическая)
Write-Host "Сборка: Linux (amd64)..."
$env:GOOS = "linux"; $env:GOARCH = "amd64"; $env:CGO_ENABLED = "0"
Push-Location $rootDir
go build -ldflags "-s -w" -o "$linDir/gosoeth" .
Pop-Location

# Упаковка Linux .tar.gz
Write-Host "Упаковка: gosoeth-v$version-linux-amd64.tar.gz..."
Copy-Item "$rootDir/config.ini.example" "$linDir/config.ini"
Push-Location $linDir
tar -czf "$distDir/gosoeth-v$version-linux-amd64.tar.gz" gosoeth config.ini
Pop-Location

# Сброс переменных
$env:GOOS = ""; $env:GOARCH = ""; $env:CGO_ENABLED = ""

Write-Host "`n--- Все архивы готовы в папке $distDir ---" -ForegroundColor Cyan
