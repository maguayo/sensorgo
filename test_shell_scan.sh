#!/bin/bash
# Test scanning with shell commands (bypasses Go library)

echo "=== Test de escaneo con shell ==="
echo ""

echo "1. Verificando bluetoothctl..."
if ! command -v bluetoothctl &> /dev/null; then
    echo "❌ bluetoothctl no encontrado"
    exit 1
fi
echo "✅ bluetoothctl disponible"
echo ""

echo "2. Verificando adaptador..."
bluetoothctl show | head -5
echo ""

echo "3. Encendiendo adaptador..."
bluetoothctl power on
sleep 2
echo ""

echo "4. Escaneando dispositivos BLE (10 segundos)..."
echo "   (Buscando RuuviTags...)"
echo ""

# Start scan in background, capture output
timeout 10 bluetoothctl scan on &
SCAN_PID=$!

sleep 10
kill $SCAN_PID 2>/dev/null

echo ""
echo "5. Listando dispositivos encontrados..."
bluetoothctl devices | while read line; do
    echo "   $line"
    # Check if it's a RuuviTag
    if echo "$line" | grep -qi "ruuvi"; then
        echo "      ⭐ ¡RuuviTag!"
    fi
done

echo ""
echo "=== Test completado ==="
