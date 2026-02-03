#!/bin/bash
# Script de compilación para Raspberry Pi ARM64

set -e

GREEN='\033[0;32m'
NC='\033[0m'

echo -e "${GREEN}🔨 Compilando para Raspberry Pi (ARM64)...${NC}"

# Compilar para Linux ARM64 (sin CGO para portabilidad)
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o insectius-monitor main.go

echo -e "${GREEN}✅ Binario compilado: insectius-monitor${NC}"

# Mostrar tamaño del binario
SIZE=$(ls -lh insectius-monitor | awk '{print $5}')
echo -e "${GREEN}📦 Tamaño del binario: $SIZE${NC}"
echo ""
echo "Para transferir al Raspberry Pi:"
echo "  scp insectius-monitor pi@raspberry:/home/pi/"
echo ""
echo "En el Raspberry Pi, ejecuta:"
echo "  sudo ./install.sh"
