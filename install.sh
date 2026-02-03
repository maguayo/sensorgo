#!/bin/bash
# Script de instalación para Insectius Monitor en Debian/Ubuntu
# Modo robusto: no falla ante errores de dependencias

# No usar set -e para permitir continuar si hay errores no críticos

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}═══════════════════════════════════════════════${NC}"
echo -e "${GREEN}  RuuviTag Monitor - Instalación para Ubuntu  ${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════${NC}"
echo ""

# Verificar que se ejecuta como root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}❌ Error: Este script debe ejecutarse como root${NC}"
    echo "   Usa: sudo ./install.sh"
    exit 1
fi

# Obtener el usuario real (no root)
REAL_USER=${SUDO_USER:-$USER}
if [ "$REAL_USER" = "root" ]; then
    echo -e "${YELLOW}⚠️  Advertencia: No se detectó un usuario no-root${NC}"
    read -p "Introduce el nombre de usuario para ejecutar el servicio: " REAL_USER
fi

echo -e "${GREEN}📦 Instalando para el usuario: $REAL_USER${NC}"
echo ""

# Verificar dependencias
echo "🔍 Verificando dependencias..."

# Buscar Go en rutas comunes
GO_PATH=""
if command -v go &> /dev/null; then
    GO_PATH=$(command -v go)
elif [ -x "/usr/local/go/bin/go" ]; then
    GO_PATH="/usr/local/go/bin/go"
    export PATH=$PATH:/usr/local/go/bin
elif [ -x "/usr/bin/go" ]; then
    GO_PATH="/usr/bin/go"
fi

if [ -z "$GO_PATH" ]; then
    echo -e "${RED}❌ Go no está instalado${NC}"
    echo "   Ejecuta: ./install-go-ubuntu.sh"
    exit 1
fi

echo -e "${GREEN}✓ Go encontrado en: $GO_PATH${NC}"
$GO_PATH version

# Instalar dependencias del sistema (no falla si hay problemas)
echo "📦 Instalando dependencias del sistema..."
echo "  (Continuará aunque algunas dependencias fallen)"
apt-get update -qq || true

# Intentar arreglar paquetes rotos primero
echo "  🔧 Arreglando paquetes rotos..."
apt-get install -f -y || true

# Instalar dependencias de Bluetooth una por una
echo "  📡 Instalando Bluetooth..."
apt-get install -y bluetooth 2>/dev/null || echo -e "${YELLOW}⚠️  bluetooth ya instalado o no disponible${NC}"
apt-get install -y bluez 2>/dev/null || echo -e "${YELLOW}⚠️  bluez ya instalado o no disponible${NC}"
apt-get install -y libbluetooth-dev 2>/dev/null || echo -e "${YELLOW}⚠️  libbluetooth-dev no disponible (no crítico)${NC}"

# Instalar dependencias gráficas para Fyne (no crítico si fallan)
echo "  🖥️  Instalando dependencias gráficas..."
apt-get install -y libgl1-mesa-dev 2>/dev/null || true
apt-get install -y xorg-dev 2>/dev/null || true
apt-get install -y libx11-dev 2>/dev/null || true
apt-get install -y libxcursor-dev 2>/dev/null || true
apt-get install -y libxrandr-dev 2>/dev/null || true
apt-get install -y libxinerama-dev 2>/dev/null || true
apt-get install -y libxi-dev 2>/dev/null || true
apt-get install -y libglfw3-dev 2>/dev/null || true
apt-get install -y libxxf86vm-dev 2>/dev/null || true
apt-get install -y gcc 2>/dev/null || true
apt-get install -y pkg-config 2>/dev/null || true

echo -e "${GREEN}✓ Dependencias instaladas (o ya presentes)${NC}"

# Compilar el binario si no existe
if [ ! -f "insectius-monitor" ]; then
    echo "🔨 Compilando binario..."
    echo "  (Esto puede tomar varios minutos en Raspberry Pi...)"

    if sudo -u $REAL_USER env PATH=$PATH CGO_ENABLED=1 $GO_PATH build -o insectius-monitor main.go 2>&1 | tee /tmp/build.log; then
        echo -e "${GREEN}✓ Compilación exitosa${NC}"
    else
        echo -e "${RED}❌ Error en la compilación${NC}"
        echo "  Ver detalles en: /tmp/build.log"
        echo ""
        echo "Últimas líneas del error:"
        tail -20 /tmp/build.log
        echo ""
        echo -e "${YELLOW}💡 Intenta instalar las dependencias manualmente:${NC}"
        echo "  sudo apt-get install -y build-essential libgl1-mesa-dev xorg-dev"
        exit 1
    fi
fi

# Verificar que el binario existe
if [ ! -f "insectius-monitor" ]; then
    echo -e "${RED}❌ Error: No se encontró el binario insectius-monitor${NC}"
    echo "  Compílalo manualmente con: go build -o insectius-monitor main.go"
    exit 1
fi

# Crear directorio de instalación
echo "📁 Creando directorio de instalación..."
mkdir -p /opt/insectius-monitor
chown $REAL_USER:$REAL_USER /opt/insectius-monitor

# Copiar binario
echo "📋 Copiando binario..."
cp insectius-monitor /opt/insectius-monitor/
chmod +x /opt/insectius-monitor/insectius-monitor
chown $REAL_USER:$REAL_USER /opt/insectius-monitor/insectius-monitor

# Copiar archivo de configuración si existe
if [ -f "authorized_sensors.json" ]; then
    echo "⚙️  Copiando configuración de sensores..."
    cp authorized_sensors.json /opt/insectius-monitor/
    chown $REAL_USER:$REAL_USER /opt/insectius-monitor/authorized_sensors.json
else
    echo -e "${YELLOW}⚠️  No se encontró authorized_sensors.json${NC}"
    echo "   Ejecuta el programa manualmente una vez para registrar sensores:"
    echo "   sudo -u $REAL_USER /opt/insectius-monitor/insectius-monitor"
fi

# Crear archivo de servicio systemd
echo "🔧 Configurando servicio systemd..."
cat insectius-monitor.service | sed "s/%USERNAME%/$REAL_USER/g" > /etc/systemd/system/insectius-monitor.service

# Dar permisos de Bluetooth al usuario
echo "🔐 Configurando permisos de Bluetooth..."
usermod -a -G bluetooth $REAL_USER

# Configurar capabilities para el binario (acceso Bluetooth sin root)
setcap 'cap_net_raw,cap_net_admin+eip' /opt/insectius-monitor/insectius-monitor

# Recargar systemd
echo "🔄 Recargando systemd..."
systemctl daemon-reload

# Habilitar servicio para auto-inicio
echo "✅ Habilitando auto-inicio..."
systemctl enable insectius-monitor.service

echo ""
echo -e "${GREEN}═══════════════════════════════════════════════${NC}"
echo -e "${GREEN}  ✅ Instalación completada exitosamente      ${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════${NC}"
echo ""
echo "📋 Comandos útiles:"
echo ""
echo "  Iniciar servicio:"
echo "    sudo systemctl start insectius-monitor"
echo ""
echo "  Detener servicio:"
echo "    sudo systemctl stop insectius-monitor"
echo ""
echo "  Ver estado:"
echo "    sudo systemctl status insectius-monitor"
echo ""
echo "  Ver logs:"
echo "    sudo journalctl -u insectius-monitor -f"
echo ""
echo "  Deshabilitar auto-inicio:"
echo "    sudo systemctl disable insectius-monitor"
echo ""
echo -e "${YELLOW}⚠️  IMPORTANTE:${NC}"
echo "  1. El usuario $REAL_USER debe cerrar sesión y volver a entrar"
echo "     para que los permisos de Bluetooth tomen efecto"
echo ""
if [ ! -f "/opt/insectius-monitor/authorized_sensors.json" ]; then
    echo "  2. Antes de iniciar el servicio, registra tus sensores:"
    echo "     cd /opt/insectius-monitor && sudo -u $REAL_USER ./insectius-monitor"
    echo "     (Espera 10 segundos para que detecte los sensores, luego Ctrl+C)"
    echo ""
fi
echo "  3. Para iniciar el servicio ahora:"
echo "     sudo systemctl start insectius-monitor"
echo ""
