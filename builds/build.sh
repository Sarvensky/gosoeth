#!/bin/bash

# Скрипт сборки проекта gosoeth для Linux
echo "--- Запуск сборки и упаковки gosoeth (Linux Bash) ---"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
VERSION=$(cat "$ROOT_DIR/version.txt" | xargs)
DIST_DIR="$SCRIPT_DIR/dist"

# 1. Очистка и создание папок
rm -rf "$SCRIPT_DIR/windows" "$SCRIPT_DIR/linux" "$DIST_DIR"
mkdir -p "$SCRIPT_DIR/windows" "$SCRIPT_DIR/linux" "$DIST_DIR"

# 2. Сборка под Windows (amd64)
echo "Сборка: Windows (amd64)..."
cd "$ROOT_DIR"
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o "$SCRIPT_DIR/windows/gosoeth.exe" .
cd "$SCRIPT_DIR"

# Упаковка Windows .zip
echo "Упаковка: gosoeth-v$VERSION-windows-amd64.zip..."
cp "$ROOT_DIR/config.ini.example" "$SCRIPT_DIR/windows/config.ini"
zip -j "$DIST_DIR/gosoeth-v$VERSION-windows-amd64.zip" "$SCRIPT_DIR/windows/gosoeth.exe" "$SCRIPT_DIR/windows/config.ini"

# 3. Сборка под Linux (amd64, статическая)
echo "Сборка: Linux (amd64)..."
cd "$ROOT_DIR"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o "$SCRIPT_DIR/linux/gosoeth" .
cd "$SCRIPT_DIR"

# Упаковка Linux .tar.gz
echo "Упаковка: gosoeth-v$VERSION-linux-amd64.tar.gz..."
cp "$ROOT_DIR/config.ini.example" "$SCRIPT_DIR/linux/config.ini"
tar -czf "$DIST_DIR/gosoeth-v$VERSION-linux-amd64.tar.gz" -C "$SCRIPT_DIR/linux" gosoeth config.ini

echo -e "\n--- Все архивы готовы в папке $DIST_DIR ---"
