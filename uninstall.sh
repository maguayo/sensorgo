#!/bin/bash
# Script de desinstalación para RuuviTag Monitor

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${YELLOW}═══════════════════════════════════════════════${NC}"
echo -e "${YELLOW}  RuuviTag Monitor - Desinstalación          ${NC}"
echo -e "${YELLOW}═══════════════════════════════════════════════${NC}"
echo ""

# Verificar que se ejecuta como root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}❌ Error: Este script debe ejecutarse como root${NC}"
    echo "   Usa: sudo ./uninstall.sh"
    exit 1
fi

# Confirmar desinstalación
read -p "¿Estás seguro de que quieres desinstalar RuuviTag Monitor? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Desinstalación cancelada."
    exit 0
fi

# Detener servicio
echo "🛑 Deteniendo servicio..."
systemctl stop insectius-monitor.service 2>/dev/null || true

# Deshabilitar servicio
echo "❌ Deshabilitando auto-inicio..."
systemctl disable insectius-monitor.service 2>/dev/null || true

# Eliminar archivo de servicio
echo "🗑️  Eliminando servicio systemd..."
rm -f /etc/systemd/system/insectius-monitor.service

# Recargar systemd
echo "🔄 Recargando systemd..."
systemctl daemon-reload

# Preguntar si eliminar configuración
echo ""
read -p "¿Eliminar configuración de sensores? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "🗑️  Eliminando directorio de instalación..."
    rm -rf /opt/insectius-monitor
else
    echo "📋 Manteniendo configuración en /opt/insectius-monitor/"
    echo "🗑️  Eliminando solo el binario..."
    rm -f /opt/insectius-monitor/insectius-monitor
fi

echo ""
echo -e "${GREEN}✅ Desinstalación completada${NC}"
echo ""
