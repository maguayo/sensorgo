#!/bin/bash
# Script para instalar Go en Raspberry Pi OS
# Detecta automáticamente la arquitectura (ARM32/ARM64)

set -e

GO_VERSION="1.21.6"

echo "📦 Instalando Go ${GO_VERSION} en Raspberry Pi OS..."

# Detectar arquitectura
ARCH=$(uname -m)
echo "🔍 Arquitectura detectada: $ARCH"

# Determinar el archivo a descargar
if [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
    GO_TARBALL="go${GO_VERSION}.linux-arm64.tar.gz"
    echo "✓ Usando binario ARM64 (64-bit)"
elif [ "$ARCH" = "armv7l" ] || [ "$ARCH" = "armv6l" ]; then
    GO_TARBALL="go${GO_VERSION}.linux-armv6l.tar.gz"
    echo "✓ Usando binario ARMv6 (32-bit, compatible con todos los Raspberry Pi)"
else
    echo "❌ Error: Arquitectura $ARCH no soportada"
    echo "   Arquitecturas soportadas: arm64, aarch64, armv7l, armv6l"
    exit 1
fi

# Descargar Go
echo "⬇️  Descargando Go..."
cd /tmp
wget -q --show-progress https://go.dev/dl/${GO_TARBALL}

# Eliminar instalación anterior si existe
if [ -d "/usr/local/go" ]; then
    echo "🗑️  Eliminando instalación anterior de Go..."
    sudo rm -rf /usr/local/go
fi

# Extraer nueva versión
echo "📂 Extrayendo Go a /usr/local/go..."
sudo tar -C /usr/local -xzf ${GO_TARBALL}

# Limpiar
rm ${GO_TARBALL}

# Configurar PATH en .bashrc
echo "⚙️  Configurando PATH..."
if ! grep -q "/usr/local/go/bin" ~/.bashrc; then
    echo "" >> ~/.bashrc
    echo "# Go Programming Language" >> ~/.bashrc
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
    echo "✓ PATH añadido a ~/.bashrc"
else
    echo "✓ PATH ya configurado en ~/.bashrc"
fi

# Configurar PATH en .profile (por si se usa)
if [ -f ~/.profile ] && ! grep -q "/usr/local/go/bin" ~/.profile; then
    echo "" >> ~/.profile
    echo "# Go Programming Language" >> ~/.profile
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
    echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.profile
    echo "✓ PATH añadido a ~/.profile"
fi

# Aplicar cambios en la sesión actual
export PATH=$PATH:/usr/local/go/bin

# Verificar instalación
echo ""
echo "✅ Go instalado correctamente!"
echo ""

if command -v go &> /dev/null; then
    echo "🎉 Verificación exitosa:"
    go version
    echo ""
    echo "Go está listo para usar."
else
    echo "⚠️  Go instalado pero no disponible en esta sesión."
    echo ""
    echo "Para usar Go ahora, ejecuta:"
    echo "  source ~/.bashrc"
    echo ""
    echo "O cierra y vuelve a abrir la terminal."
fi

echo ""
echo "📋 Información útil:"
echo "  - Go instalado en: /usr/local/go"
echo "  - GOPATH por defecto: ~/go"
echo "  - Verifica la versión: go version"
echo "  - Compila programas: go build"
echo ""
