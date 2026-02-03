#!/bin/bash
# Script para instalar Go en Ubuntu/Debian

set -e

GO_VERSION="1.21.6"

echo "📦 Instalando Go ${GO_VERSION} en Ubuntu..."

# Descargar Go
cd /tmp
wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz

# Eliminar instalación anterior si existe
sudo rm -rf /usr/local/go

# Extraer nueva versión
sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz

# Limpiar
rm go${GO_VERSION}.linux-amd64.tar.gz

# Añadir al PATH
if ! grep -q "/usr/local/go/bin" ~/.bashrc; then
    echo "" >> ~/.bashrc
    echo "# Go" >> ~/.bashrc
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
    echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
fi

# Aplicar cambios
export PATH=$PATH:/usr/local/go/bin

echo ""
echo "✅ Go instalado correctamente!"
echo ""
echo "Verifica la instalación:"
echo "  go version"
echo ""
echo "Si no funciona, ejecuta:"
echo "  source ~/.bashrc"
echo "  go version"
