#!/bin/bash

# Скрипт сборки проекта gosoeth для Linux
echo "--- Запуск сборки и упаковки gosoeth (Linux Bash) ---"

VERSION="1.0.0"
DIST_DIR="./dist"
ROOT_DIR="../"

# 1. Очистка и создание папок
rm -rf ./windows ./linux "$DIST_DIR"
mkdir -p ./windows ./linux "$DIST_DIR"

# 2. Сборка под Windows (amd64)
echo "Сборка: Windows (amd64)..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o ./windows/gosoeth.exe "${ROOT_DIR}main.go"

# Упаковка Windows .zip
echo "Упаковка: gosoeth-v$VERSION-windows-amd64.zip..."
cp "${ROOT_DIR}config.ini.example" ./windows/config.ini
zip -j "$DIST_DIR/gosoeth-v$VERSION-windows-amd64.zip" ./windows/gosoeth.exe ./windows/config.ini

# 3. Сборка под Linux (amd64, статическая)
echo "Сборка: Linux (amd64)..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o ./linux/gosoeth "${ROOT_DIR}main.go"

# Упаковка Linux .tar.gz
echo "Упаковка: gosoeth-v$VERSION-linux-amd64.tar.gz..."
cp "${ROOT_DIR}config.ini.example" ./linux/config.ini
tar -czf "$DIST_DIR/gosoeth-v$VERSION-linux-amd64.tar.gz" -C ./linux gosoeth config.ini

echo -e "\n--- Все архивы готовы в папке $DIST_DIR ---"
