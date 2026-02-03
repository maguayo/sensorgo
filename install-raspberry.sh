#!/bin/bash
# Complete installation script for Insectius Monitor on Raspberry Pi OS
# Handles everything: Go installation, compilation, systemd setup

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${GREEN}═══════════════════════════════════════════════${NC}"
echo -e "${GREEN}   Insectius Monitor - Raspberry Pi Install   ${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════${NC}"
echo ""

# Must run as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}❌ Este script debe ejecutarse como root${NC}"
    echo "   Usa: sudo ./install-raspberry.sh"
    exit 1
fi

# Get real user
REAL_USER=${SUDO_USER:-$USER}
if [ "$REAL_USER" = "root" ]; then
    echo -e "${YELLOW}⚠️  No se detectó usuario no-root${NC}"
    read -p "Introduce el nombre de usuario: " REAL_USER
fi

echo -e "${BLUE}📦 Usuario: $REAL_USER${NC}"
echo -e "${BLUE}🏠 Directorio: $(pwd)${NC}"
echo ""

# ============================================================================
# 1. INSTALL GO IF NOT PRESENT
# ============================================================================

GO_VERSION="1.21.6"
GO_INSTALLED=false

if command -v go &> /dev/null; then
    echo -e "${GREEN}✓ Go ya está instalado: $(go version)${NC}"
    GO_INSTALLED=true
elif [ -x "/usr/local/go/bin/go" ]; then
    export PATH=$PATH:/usr/local/go/bin
    echo -e "${GREEN}✓ Go encontrado en /usr/local/go${NC}"
    GO_INSTALLED=true
fi

if [ "$GO_INSTALLED" = false ]; then
    echo -e "${YELLOW}📦 Go no está instalado. Instalando...${NC}"

    # Detect architecture
    ARCH=$(uname -m)
    echo "   Arquitectura: $ARCH"

    if [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
        GO_TARBALL="go${GO_VERSION}.linux-arm64.tar.gz"
    elif [ "$ARCH" = "armv7l" ] || [ "$ARCH" = "armv6l" ]; then
        GO_TARBALL="go${GO_VERSION}.linux-armv6l.tar.gz"
    else
        echo -e "${RED}❌ Arquitectura no soportada: $ARCH${NC}"
        exit 1
    fi

    # Download and install Go
    echo "   Descargando Go ${GO_VERSION}..."
    cd /tmp
    wget -q --show-progress "https://go.dev/dl/${GO_TARBALL}"

    echo "   Instalando Go..."
    rm -rf /usr/local/go
    tar -C /usr/local -xzf "${GO_TARBALL}"
    rm "${GO_TARBALL}"

    # Add to PATH
    export PATH=$PATH:/usr/local/go/bin

    # Add to bashrc for user
    USER_HOME=$(eval echo ~$REAL_USER)
    if ! grep -q "/usr/local/go/bin" "$USER_HOME/.bashrc"; then
        echo "" >> "$USER_HOME/.bashrc"
        echo "# Go Programming Language" >> "$USER_HOME/.bashrc"
        echo 'export PATH=$PATH:/usr/local/go/bin' >> "$USER_HOME/.bashrc"
        echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> "$USER_HOME/.bashrc"
    fi

    echo -e "${GREEN}✓ Go ${GO_VERSION} instalado${NC}"
fi

# Verify Go works
if ! command -v go &> /dev/null; then
    export PATH=$PATH:/usr/local/go/bin
fi

echo -e "${GREEN}✓ Go version: $(go version)${NC}"
echo ""

# ============================================================================
# 2. INSTALL SYSTEM DEPENDENCIES
# ============================================================================

echo "📦 Instalando dependencias del sistema..."
apt-get update -qq

# Essential packages
apt-get install -y bluetooth bluez wget

# Optional packages (don't fail if missing)
apt-get install -y gcc pkg-config 2>/dev/null || true

echo -e "${GREEN}✓ Dependencias instaladas${NC}"
echo ""

# ============================================================================
# 3. CHECK API KEY
# ============================================================================

USER_HOME=$(eval echo ~$REAL_USER)
API_KEY_FILE="$USER_HOME/.insectius-monitor"

if [ ! -f "$API_KEY_FILE" ]; then
    echo -e "${YELLOW}⚠️  No se encontró API key en $API_KEY_FILE${NC}"
    echo ""
    read -p "¿Quieres introducir la API key ahora? (y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        read -p "Introduce la API key: " API_KEY
        echo "$API_KEY" > "$API_KEY_FILE"
        chmod 600 "$API_KEY_FILE"
        chown $REAL_USER:$REAL_USER "$API_KEY_FILE"
        echo -e "${GREEN}✓ API key guardada${NC}"
    else
        echo -e "${YELLOW}⚠️  Recuerda crear $API_KEY_FILE antes de usar el servicio${NC}"
    fi
else
    echo -e "${GREEN}✓ API key encontrada${NC}"
fi
echo ""

# ============================================================================
# 4. COMPILE BINARY
# ============================================================================

echo "🔨 Compilando binario..."
echo "   (Esto puede tomar varios minutos en Raspberry Pi...)"

# Get source directory
SOURCE_DIR=$(pwd)
cd "$SOURCE_DIR"

# Clean old binary
rm -f insectius-monitor

# Download dependencies
echo "   Descargando dependencias Go..."
sudo -u $REAL_USER env PATH=$PATH go mod download 2>/dev/null || true

# Compile with CGO disabled for better compatibility
echo "   Compilando..."
if sudo -u $REAL_USER env PATH=$PATH CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o insectius-monitor main.go 2>&1 | tee /tmp/build.log; then
    echo -e "${GREEN}✓ Compilación exitosa${NC}"
elif sudo -u $REAL_USER env PATH=$PATH CGO_ENABLED=0 go build -o insectius-monitor main.go 2>&1 | tee /tmp/build.log; then
    echo -e "${GREEN}✓ Compilación exitosa (sin GOARCH)${NC}"
else
    echo -e "${RED}❌ Error en compilación${NC}"
    echo "Ver: /tmp/build.log"
    tail -20 /tmp/build.log
    exit 1
fi

# Verify binary
if [ ! -f "insectius-monitor" ]; then
    echo -e "${RED}❌ Binario no creado${NC}"
    exit 1
fi

# Test binary
if ! ./insectius-monitor -h &>/dev/null && ! file insectius-monitor | grep -q "ARM"; then
    echo -e "${RED}❌ El binario no es válido para ARM${NC}"
    file insectius-monitor
    exit 1
fi

echo -e "${GREEN}✓ Binario válido: $(file insectius-monitor | cut -d: -f2)${NC}"
echo ""

# ============================================================================
# 5. INSTALL TO /opt
# ============================================================================

echo "📁 Instalando en /opt/insectius-monitor..."

# Create directory
mkdir -p /opt/insectius-monitor
chown $REAL_USER:$REAL_USER /opt/insectius-monitor

# Copy binary
cp insectius-monitor /opt/insectius-monitor/
chmod +x /opt/insectius-monitor/insectius-monitor
chown $REAL_USER:$REAL_USER /opt/insectius-monitor/insectius-monitor

# Copy sensor config if exists
if [ -f "authorized_sensors.json" ]; then
    cp authorized_sensors.json /opt/insectius-monitor/
    chown $REAL_USER:$REAL_USER /opt/insectius-monitor/authorized_sensors.json
    echo -e "${GREEN}✓ Configuración de sensores copiada${NC}"
else
    echo -e "${YELLOW}⚠️  No hay configuración de sensores (se creará en primer uso)${NC}"
fi

echo -e "${GREEN}✓ Archivos copiados a /opt/insectius-monitor${NC}"
echo ""

# ============================================================================
# 6. SETUP SYSTEMD SERVICE
# ============================================================================

echo "🔧 Configurando servicio systemd..."

# Create service file
cat > /etc/systemd/system/insectius-monitor.service <<EOF
[Unit]
Description=Insectius Monitor - Sensor Data Collection and API Sync
After=network.target bluetooth.target
Wants=bluetooth.target

[Service]
Type=simple
User=$REAL_USER
WorkingDirectory=/opt/insectius-monitor
ExecStart=/opt/insectius-monitor/insectius-monitor
Restart=always
RestartSec=10

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=insectius-monitor

# Bluetooth permissions
AmbientCapabilities=CAP_NET_RAW CAP_NET_ADMIN
CapabilityBoundingSet=CAP_NET_RAW CAP_NET_ADMIN

[Install]
WantedBy=multi-user.target
EOF

echo -e "${GREEN}✓ Servicio creado${NC}"

# Configure Bluetooth permissions
echo "🔐 Configurando permisos de Bluetooth..."
usermod -a -G bluetooth $REAL_USER 2>/dev/null || true

# Set capabilities
setcap 'cap_net_raw,cap_net_admin+eip' /opt/insectius-monitor/insectius-monitor 2>/dev/null || \
    echo -e "${YELLOW}⚠️  No se pudieron establecer capabilities (no crítico)${NC}"

# Reload systemd
systemctl daemon-reload

# Enable service
systemctl enable insectius-monitor.service

echo -e "${GREEN}✓ Servicio configurado y habilitado${NC}"
echo ""

# ============================================================================
# 7. SENSOR REGISTRATION (OPTIONAL)
# ============================================================================

if [ ! -f "/opt/insectius-monitor/authorized_sensors.json" ]; then
    echo -e "${YELLOW}═══════════════════════════════════════════════${NC}"
    echo -e "${YELLOW}  REGISTRO DE SENSORES REQUERIDO              ${NC}"
    echo -e "${YELLOW}═══════════════════════════════════════════════${NC}"
    echo ""
    echo "No hay sensores registrados todavía."
    echo ""
    read -p "¿Quieres escanear sensores ahora? (y/n): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo ""
        echo "Escaneando sensores durante 10 segundos..."
        echo "(Asegúrate de que los RuuviTags estén encendidos y cerca)"
        echo ""
        cd /opt/insectius-monitor
        timeout 15 sudo -u $REAL_USER ./insectius-monitor || true
        echo ""
        if [ -f "authorized_sensors.json" ]; then
            echo -e "${GREEN}✓ Sensores registrados${NC}"
        else
            echo -e "${YELLOW}⚠️  No se registraron sensores${NC}"
        fi
    else
        echo -e "${YELLOW}⚠️  Registra sensores manualmente antes de iniciar el servicio:${NC}"
        echo "   cd /opt/insectius-monitor"
        echo "   sudo -u $REAL_USER ./insectius-monitor"
    fi
    echo ""
fi

# ============================================================================
# 8. SUMMARY
# ============================================================================

echo -e "${GREEN}═══════════════════════════════════════════════${NC}"
echo -e "${GREEN}  ✅ INSTALACIÓN COMPLETADA                    ${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════${NC}"
echo ""
echo "📋 Estado:"
echo "  ✓ Go instalado"
echo "  ✓ Binario compilado"
echo "  ✓ Servicio configurado"
echo "  ✓ Auto-inicio habilitado"
echo ""
echo "🚀 Comandos útiles:"
echo ""
echo "  Iniciar servicio:"
echo "    sudo systemctl start insectius-monitor"
echo ""
echo "  Ver estado:"
echo "    sudo systemctl status insectius-monitor"
echo ""
echo "  Ver logs en tiempo real:"
echo "    sudo journalctl -u insectius-monitor -f"
echo ""
echo "  Detener servicio:"
echo "    sudo systemctl stop insectius-monitor"
echo ""
echo "  Reiniciar servicio:"
echo "    sudo systemctl restart insectius-monitor"
echo ""
echo -e "${YELLOW}⚠️  IMPORTANTE:${NC}"
echo "  1. El usuario $REAL_USER debe cerrar sesión y volver a entrar"
echo "     para que los permisos de Bluetooth tomen efecto"
echo ""
if [ ! -f "/opt/insectius-monitor/authorized_sensors.json" ]; then
    echo "  2. Registra sensores antes de iniciar:"
    echo "     cd /opt/insectius-monitor && sudo -u $REAL_USER ./insectius-monitor"
    echo ""
fi
echo "  3. Inicia el servicio:"
echo "     sudo systemctl start insectius-monitor"
echo ""
echo "  4. Verifica que funciona:"
echo "     sudo journalctl -u insectius-monitor -f"
echo ""
