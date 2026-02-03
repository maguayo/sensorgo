#!/bin/bash
# Script de compilación para RuuviTag Monitor

set -e

echo "🔨 Compilando RuuviTag Monitor para Linux..."

# Compilar para Linux AMD64
GOOS=linux GOARCH=amd64 go build -o insectius-monitor main.go

echo "✅ Binario compilado: insectius-monitor"
echo ""
echo "Para instalar en Debian/Ubuntu, ejecuta:"
echo "  sudo ./install.sh"
