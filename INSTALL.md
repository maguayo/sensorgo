# Instalación en Debian/Ubuntu

Guía completa para instalar RuuviTag Monitor en sistemas Debian/Ubuntu con auto-inicio al arranque.

## Requisitos Previos

### Sistema Operativo
- Ubuntu 20.04 LTS o superior
- Debian 11 (Bullseye) o superior

### Hardware
- Adaptador Bluetooth (built-in o USB)
- Pantalla conectada (para GUI fullscreen)

### Software
- Go 1.21 o superior
- Bluetooth habilitado

## Instalación Rápida

### 1. Clonar o copiar el proyecto

```bash
cd /tmp
# Copiar todos los archivos del proyecto a este directorio
```

### 2. Dar permisos de ejecución a los scripts

```bash
chmod +x build.sh install.sh uninstall.sh
```

### 3. Compilar (opcional, el instalador lo hace automáticamente)

```bash
./build.sh
```

### 4. Instalar

```bash
sudo ./install.sh
```

El script de instalación hará:
- ✅ Verificar dependencias
- ✅ Instalar paquetes necesarios (bluetooth, bluez)
- ✅ Compilar el binario si no existe
- ✅ Copiar binario a `/opt/insectius-monitor/`
- ✅ Configurar servicio systemd
- ✅ Habilitar auto-inicio
- ✅ Configurar permisos de Bluetooth

### 5. Configurar API Key

**IMPORTANTE**: Crea el archivo `~/.insectius-monitor` con tu API key:

```bash
echo 'tu-api-key-aqui' > ~/.insectius-monitor
chmod 600 ~/.insectius-monitor
```

El archivo debe estar en el directorio home del usuario que ejecutará el servicio.

### 6. Registrar sensores (primera vez)

**IMPORTANTE**: Antes de iniciar el servicio, debes registrar tus sensores RuuviTag.

```bash
cd /opt/insectius-monitor
sudo -u $USER ./insectius-monitor
```

- Espera 10 segundos mientras escanea sensores
- Verás los sensores detectados en pantalla
- Presiona `Ctrl+C` cuando veas "✅ Registro completado"

Esto creará el archivo `authorized_sensors.json` con tus sensores.

### 7. Iniciar el servicio

```bash
sudo systemctl start insectius-monitor
```

### 8. Verificar que funciona

```bash
sudo systemctl status insectius-monitor
```

Deberías ver:
```
● insectius-monitor.service - RuuviTag Monitor
   Loaded: loaded (/etc/systemd/system/insectius-monitor.service; enabled)
   Active: active (running) since ...
```

## Comandos Útiles

### Gestión del Servicio

```bash
# Iniciar
sudo systemctl start insectius-monitor

# Detener
sudo systemctl stop insectius-monitor

# Reiniciar
sudo systemctl restart insectius-monitor

# Ver estado
sudo systemctl status insectius-monitor

# Habilitar auto-inicio
sudo systemctl enable insectius-monitor

# Deshabilitar auto-inicio
sudo systemctl disable insectius-monitor
```

### Ver Logs

```bash
# Ver logs en tiempo real
sudo journalctl -u insectius-monitor -f

# Ver últimos 50 logs
sudo journalctl -u insectius-monitor -n 50

# Ver logs desde hoy
sudo journalctl -u insectius-monitor --since today

# Ver logs con timestamp
sudo journalctl -u insectius-monitor -o short-iso
```

### Re-registrar Sensores

Si necesitas cambiar los sensores autorizados:

```bash
# Detener servicio
sudo systemctl stop insectius-monitor

# Re-registrar
cd /opt/insectius-monitor
sudo -u $USER ./insectius-monitor -reregister

# Reiniciar servicio
sudo systemctl start insectius-monitor
```

## Auto-inicio al Arranque

El servicio está configurado para iniciar automáticamente al arrancar el sistema.

**Características:**
- ✅ Inicia después de que la red y Bluetooth estén disponibles
- ✅ Espera a que el servidor gráfico esté listo
- ✅ Se reinicia automáticamente si falla
- ✅ Espera 10 segundos entre reintentos

**Para verificar el auto-inicio:**
```bash
sudo systemctl is-enabled insectius-monitor
```

Debe mostrar: `enabled`

**Para probar el auto-inicio:**
```bash
sudo reboot
```

Después del reinicio, el programa debería estar ejecutándose automáticamente.

## Ubicación de Archivos

```
/opt/insectius-monitor/
├── insectius-monitor           # Binario ejecutable
└── authorized_sensors.json     # Configuración de sensores

~/.insectius-monitor            # API key para autenticación (crear manualmente)

/etc/systemd/system/
└── insectius-monitor.service   # Archivo de servicio systemd
```

## Permisos

El programa necesita permisos especiales para acceder a Bluetooth:

1. **Grupo bluetooth**: El usuario se añade al grupo `bluetooth`
2. **Capabilities**: Se configuran capabilities en el binario:
   - `CAP_NET_RAW`: Para acceso a raw sockets (Bluetooth)
   - `CAP_NET_ADMIN`: Para administración de red

**Verificar capabilities:**
```bash
getcap /opt/insectius-monitor/insectius-monitor
```

Debe mostrar:
```
/opt/insectius-monitor/insectius-monitor = cap_net_admin,cap_net_raw+eip
```

## Troubleshooting

### El servicio no inicia

**Verificar logs:**
```bash
sudo journalctl -u insectius-monitor -n 50
```

**Errores comunes:**

**Error: "Permission denied" o "Operation not permitted"**
```bash
# Re-configurar permisos
sudo setcap 'cap_net_raw,cap_net_admin+eip' /opt/insectius-monitor/insectius-monitor
sudo systemctl restart insectius-monitor
```

**Error: "No Bluetooth adapter found"**
```bash
# Verificar que Bluetooth está habilitado
sudo systemctl status bluetooth
sudo hciconfig

# Reiniciar Bluetooth
sudo systemctl restart bluetooth
```

**Error: "No se encontraron sensores"**
- Verifica que los sensores RuuviTag estén encendidos y cerca
- Ejecuta manualmente para probar:
  ```bash
  cd /opt/insectius-monitor
  sudo -u $USER ./insectius-monitor
  ```

### La GUI no se muestra

**Verificar display:**
```bash
echo $DISPLAY
# Debe mostrar :0 o similar
```

**Si usas Wayland en lugar de X11:**
Edita `/etc/systemd/system/insectius-monitor.service` y añade:
```ini
Environment="WAYLAND_DISPLAY=wayland-0"
```

Luego:
```bash
sudo systemctl daemon-reload
sudo systemctl restart insectius-monitor
```

### Error de Autenticación (HTTP 401)

Si ves errores HTTP 401:

```bash
# Verificar que existe el archivo
ls -la ~/.insectius-monitor

# Ver el contenido (asegúrate de que es correcto)
cat ~/.insectius-monitor

# Crear o actualizar API key
echo 'tu-api-key-correcta' > ~/.insectius-monitor
chmod 600 ~/.insectius-monitor

# Reiniciar servicio
sudo systemctl restart insectius-monitor
```

**Nota:** Si el servicio corre como otro usuario, el archivo debe estar en el home de ese usuario:
```bash
sudo -u $SERVICE_USER bash -c "echo 'tu-api-key' > ~/.insectius-monitor"
sudo -u $SERVICE_USER chmod 600 ~/.insectius-monitor
```

### Errores HTTP 400

Ver la sección de debugging en el README.md principal. Los logs mostrarán:
- URL del endpoint
- Payload enviado
- Response del servidor

## Desinstalación

Para desinstalar completamente:

```bash
sudo ./uninstall.sh
```

Opciones:
- Eliminar todo (binario + configuración)
- Mantener configuración de sensores

## Actualización

Para actualizar a una nueva versión:

```bash
# 1. Detener servicio
sudo systemctl stop insectius-monitor

# 2. Compilar nueva versión
./build.sh

# 3. Copiar binario
sudo cp insectius-monitor /opt/insectius-monitor/

# 4. Re-configurar permisos
sudo setcap 'cap_net_raw,cap_net_admin+eip' /opt/insectius-monitor/insectius-monitor

# 5. Reiniciar servicio
sudo systemctl start insectius-monitor
```

## Seguridad

**Recomendaciones:**

1. ✅ El servicio corre como usuario no-root
2. ✅ Solo tiene los permisos mínimos necesarios (capabilities)
3. ✅ Los sensores están en whitelist (authorized_sensors.json)
4. ⚠️ Asegúrate de que la API (go.larvai.com) usa HTTPS
5. ⚠️ Considera añadir autenticación a la API

## Soporte

Para problemas o preguntas:
1. Revisa los logs: `sudo journalctl -u insectius-monitor -f`
2. Verifica el estado: `sudo systemctl status insectius-monitor`
3. Consulta EXPECTED_API.md para problemas de API

---

**Última actualización:** 2026-02-03
