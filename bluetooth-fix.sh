#!/bin/bash
# Script de diagnóstico y reparación de Bluetooth para Raspberry Pi

set +e  # No salir si hay errores

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}════════════════════════════════════════${NC}"
echo -e "${BLUE}  Diagnóstico y Reparación de Bluetooth ${NC}"
echo -e "${BLUE}════════════════════════════════════════${NC}"
echo ""

# Función para verificar
check() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $1"
        return 0
    else
        echo -e "${RED}✗${NC} $1"
        return 1
    fi
}

# 1. Verificar servicio Bluetooth
echo "1️⃣  Verificando servicio Bluetooth..."
systemctl is-active bluetooth &>/dev/null
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓${NC} Servicio bluetooth activo"
else
    echo -e "${RED}✗${NC} Servicio bluetooth NO activo"
    echo "   Intentando iniciar..."
    sudo systemctl start bluetooth
    sleep 2
    systemctl is-active bluetooth &>/dev/null && echo -e "${GREEN}✓${NC} Servicio iniciado" || echo -e "${RED}✗${NC} No se pudo iniciar"
fi
echo ""

# 2. Verificar rfkill
echo "2️⃣  Verificando bloqueos rfkill..."
if command -v rfkill &> /dev/null; then
    rfkill list bluetooth
    if rfkill list bluetooth | grep -q "blocked: yes"; then
        echo -e "${YELLOW}⚠${NC}  Bluetooth está bloqueado"
        echo "   Desbloqueando..."
        sudo rfkill unblock bluetooth
        sleep 1
        rfkill list bluetooth | grep -q "blocked: no" && echo -e "${GREEN}✓${NC} Desbloqueado" || echo -e "${RED}✗${NC} Sigue bloqueado"
    else
        echo -e "${GREEN}✓${NC} Bluetooth no está bloqueado"
    fi
else
    echo -e "${YELLOW}⚠${NC}  rfkill no disponible"
fi
echo ""

# 3. Verificar adaptador Bluetooth
echo "3️⃣  Verificando adaptador Bluetooth..."
if command -v hciconfig &> /dev/null; then
    hciconfig -a
    if hciconfig | grep -q "hci0"; then
        echo -e "${GREEN}✓${NC} Adaptador hci0 detectado"

        # Verificar si está UP
        if hciconfig hci0 | grep -q "UP RUNNING"; then
            echo -e "${GREEN}✓${NC} Adaptador está UP y RUNNING"
        else
            echo -e "${YELLOW}⚠${NC}  Adaptador está DOWN"
            echo "   Levantando adaptador..."
            sudo hciconfig hci0 up
            sleep 2
            hciconfig hci0 | grep -q "UP RUNNING" && echo -e "${GREEN}✓${NC} Adaptador levantado" || echo -e "${RED}✗${NC} No se pudo levantar"
        fi
    else
        echo -e "${RED}✗${NC} NO se detectó adaptador Bluetooth"
        echo "   Posible problema de hardware"
    fi
else
    echo -e "${YELLOW}⚠${NC}  hciconfig no disponible, instalando bluez..."
    sudo apt-get install -y bluez
fi
echo ""

# 4. Test de escaneo
echo "4️⃣  Probando escaneo Bluetooth..."
timeout 5 sudo hcitool lescan &>/dev/null
if [ $? -eq 124 ] || [ $? -eq 0 ]; then
    echo -e "${GREEN}✓${NC} Escaneo funciona correctamente"
else
    echo -e "${RED}✗${NC} Escaneo falló"
    echo "   Reiniciando Bluetooth..."
    sudo systemctl restart bluetooth
    sleep 3
fi
echo ""

# 5. Verificar permisos
echo "5️⃣  Verificando permisos..."
USER=${SUDO_USER:-$USER}
if groups $USER | grep -q bluetooth; then
    echo -e "${GREEN}✓${NC} Usuario $USER está en grupo bluetooth"
else
    echo -e "${YELLOW}⚠${NC}  Usuario $USER NO está en grupo bluetooth"
    echo "   Agregando al grupo..."
    sudo usermod -a -G bluetooth $USER
    echo -e "${GREEN}✓${NC} Agregado (requiere logout/login)"
fi
echo ""

# 6. Verificar capabilities del binario
echo "6️⃣  Verificando capabilities del binario..."
if [ -f "/opt/insectius-monitor/insectius-monitor" ]; then
    CAPS=$(getcap /opt/insectius-monitor/insectius-monitor 2>/dev/null)
    if echo "$CAPS" | grep -q "cap_net_raw,cap_net_admin"; then
        echo -e "${GREEN}✓${NC} Capabilities correctas"
    else
        echo -e "${YELLOW}⚠${NC}  Capabilities faltantes"
        echo "   Configurando..."
        sudo setcap 'cap_net_raw,cap_net_admin+eip' /opt/insectius-monitor/insectius-monitor
        echo -e "${GREEN}✓${NC} Configuradas"
    fi
else
    echo -e "${YELLOW}⚠${NC}  Binario no encontrado en /opt/insectius-monitor/"
fi
echo ""

# 7. Verificar otros procesos usando Bluetooth
echo "7️⃣  Verificando procesos usando Bluetooth..."
BT_PROCS=$(lsof -t /dev/rfcomm* 2>/dev/null | wc -l)
if [ $BT_PROCS -gt 0 ]; then
    echo -e "${YELLOW}⚠${NC}  Hay $BT_PROCS proceso(s) usando Bluetooth"
    lsof /dev/rfcomm* 2>/dev/null || true
else
    echo -e "${GREEN}✓${NC} No hay conflictos con otros procesos"
fi
echo ""

# 8. Logs del sistema
echo "8️⃣  Últimos errores de Bluetooth en logs..."
dmesg | grep -i bluetooth | tail -5
echo ""

# 9. Estado del servicio insectius-monitor
echo "9️⃣  Estado del servicio..."
if systemctl is-active insectius-monitor &>/dev/null; then
    echo -e "${GREEN}✓${NC} Servicio insectius-monitor activo"
    echo ""
    echo "Últimas líneas del log:"
    sudo journalctl -u insectius-monitor -n 5 --no-pager
else
    echo -e "${YELLOW}⚠${NC}  Servicio insectius-monitor NO activo"
fi
echo ""

# Resumen y recomendaciones
echo -e "${BLUE}════════════════════════════════════════${NC}"
echo -e "${BLUE}  RECOMENDACIONES                       ${NC}"
echo -e "${BLUE}════════════════════════════════════════${NC}"
echo ""
echo "🔧 Acciones a realizar:"
echo ""
echo "1. Reiniciar servicio Bluetooth:"
echo "   sudo systemctl restart bluetooth"
echo ""
echo "2. Detener el servicio insectius-monitor:"
echo "   sudo systemctl stop insectius-monitor"
echo ""
echo "3. Probar manualmente el escaneo:"
echo "   sudo hcitool lescan"
echo "   (Ctrl+C para detener)"
echo ""
echo "4. Si el escaneo manual funciona, reiniciar el servicio:"
echo "   sudo systemctl start insectius-monitor"
echo ""
echo "5. Ver logs en tiempo real:"
echo "   sudo journalctl -u insectius-monitor -f"
echo ""

# Opción de fix automático
echo ""
read -p "¿Quieres que intente reparar automáticamente? (y/n): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo ""
    echo -e "${YELLOW}Reparando...${NC}"

    # Detener servicio
    echo "• Deteniendo servicio..."
    sudo systemctl stop insectius-monitor

    # Reiniciar Bluetooth
    echo "• Reiniciando Bluetooth..."
    sudo systemctl stop bluetooth
    sleep 2
    sudo rfkill unblock bluetooth
    sleep 1
    sudo systemctl start bluetooth
    sleep 3

    # Levantar adaptador
    echo "• Levantando adaptador..."
    sudo hciconfig hci0 down
    sleep 1
    sudo hciconfig hci0 up
    sleep 2

    # Test
    echo "• Probando escaneo..."
    timeout 5 sudo hcitool lescan 2>&1 | head -5

    # Reiniciar servicio
    echo "• Iniciando servicio..."
    sudo systemctl start insectius-monitor
    sleep 3

    echo ""
    echo -e "${GREEN}✓ Reparación completada${NC}"
    echo ""
    echo "Verifica los logs:"
    echo "  sudo journalctl -u insectius-monitor -f"
fi

echo ""
