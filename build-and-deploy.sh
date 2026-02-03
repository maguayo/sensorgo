#!/bin/bash
# Script para compilar en Mac y desplegar en servidor Linux

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}🔨 Compilando para Linux desde macOS...${NC}"

# Compilar para Linux AMD64
GOOS=linux GOARCH=amd64 go build -o insectius-monitor main.go

echo -e "${GREEN}✅ Binario compilado: insectius-monitor${NC}"
echo ""

# Verificar que el binario existe
if [ ! -f "insectius-monitor" ]; then
    echo -e "${RED}❌ Error: No se pudo compilar el binario${NC}"
    exit 1
fi

# Mostrar tamaño del binario
SIZE=$(ls -lh insectius-monitor | awk '{print $5}')
echo -e "${GREEN}📦 Tamaño del binario: $SIZE${NC}"
echo ""

echo -e "${YELLOW}📤 Para desplegar en el servidor:${NC}"
echo ""
echo "1. Transferir el binario:"
echo "   scp insectius-monitor usuario@servidor:/tmp/"
echo ""
echo "2. En el servidor, continúa con la instalación:"
echo "   sudo ./install.sh"
echo ""
