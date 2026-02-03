# RuuviTag Scanner

Programa en Go para escanear y leer datos de sensores RuuviTag por Bluetooth con capa de seguridad.

## Características

- 🔒 **Modo seguro**: Solo lee datos de sensores previamente autorizados
- 🔍 Escanea automáticamente sensores RuuviTag cercanos
- 📊 Lee temperatura, humedad, presión atmosférica y nivel de batería
- 💾 Soporta formato RAWv2 (el más común en sensores RuuviTag)
- ⏱️ Muestra datos en tiempo real
- 🌐 **Envío automático a API**: Envía datos cada 5 minutos a la API de Larvai
- 🖥️ **Interfaz de terminal ASCII**: Muestra el estado de sincronización con la API
  - ✅ Fondo verde cuando la última sincronización fue exitosa
  - ❌ Fondo rojo cuando hubo un error en la sincronización
  - 💻 Compatible con SSH y funciona sin display
  - 📦 Binario ligero (~7.5MB) sin dependencias CGO

## Requisitos

- Go 1.21 o superior
- Bluetooth habilitado en tu sistema
- En macOS: permisos de Bluetooth para la terminal
- En Linux: puede requerir `sudo` o permisos para acceder a Bluetooth

## Instalación

```bash
go mod tidy
```

## Configuración de API Key

**IMPORTANTE**: Antes de ejecutar el programa, crea un archivo `~/.insectius-monitor` con tu API key:

```bash
echo 'tu-api-key-aqui' > ~/.insectius-monitor
chmod 600 ~/.insectius-monitor
```

La API key se usa para autenticación con el servidor API (header `Authorization: Bearer`).

**Seguridad:**
- El archivo está en tu directorio home (~/)
- No compartas este archivo
- Permisos restrictivos (600) para que solo tú puedas leerlo

## Uso

### Primera ejecución (Registro de sensores)

La primera vez que ejecutes el programa, escaneará durante 10 segundos y registrará todos los sensores RuuviTag que encuentre:

```bash
go run main.go
```

Esto creará un archivo `authorized_sensors.json` con la lista de sensores autorizados. **Solo estos sensores serán leídos en ejecuciones futuras**.

### Ejecuciones posteriores (Modo seguro)

En ejecuciones normales, el programa solo mostrará datos de los sensores previamente registrados:

```bash
go run main.go
```

### Re-registrar sensores

Si necesitas añadir o cambiar los sensores autorizados:

```bash
go run main.go -reregister
```

Esto sobrescribirá la lista actual y escaneará nuevamente durante 10 segundos.

Para detener el escaneo en cualquier momento, presiona `Ctrl+C`.

### Ejecutar en segundo plano con screen/tmux

Para mantener el programa ejecutándose después de cerrar la sesión SSH:

**Con screen:**
```bash
# Iniciar nueva sesión
screen -S monitor

# Ejecutar el programa
./insectius-monitor

# Detach (dejar corriendo en segundo plano): Ctrl+A luego D
# Reattach (volver a la sesión): screen -r monitor
```

**Con tmux:**
```bash
# Iniciar nueva sesión
tmux new -s monitor

# Ejecutar el programa
./insectius-monitor

# Detach: Ctrl+B luego D
# Reattach: tmux attach -t monitor
```

**Con systemd (recomendado para producción):**
```bash
# Ver estado
sudo systemctl status insectius-monitor

# Ver logs en tiempo real
journalctl -u insectius-monitor -f
```

## Datos mostrados

- 🌡️ **Temperatura**: en grados Celsius
- 💧 **Humedad**: porcentaje relativo
- 📊 **Presión**: en hectopascales (hPa)
- 🔋 **Batería**: en milivoltios (mV)
- 📶 **TX Power**: potencia de transmisión en dBm

## Interfaz de Terminal

El programa incluye una interfaz de terminal ASCII que muestra:

```
╔════════════════════════════════════════════════╗
║ Insectius Monitor             Sensores: 2/2 ✓ ║
╠════════════════════════════════════════════════╣
║                                                ║
║                     ✓                          ║
║                  EXITOSA                       ║
║                                                ║
║              15:30:45 - 03/02/2026             ║
║                                                ║
╠════════════════════════════════════════════════╣
║              Actividad del Sistema             ║
╠════════════════════════════════════════════════╣
║ [15:30:45] 📡 Ruuvi 39B1 detectado            ║
║ [15:30:45] 📊 Datos: 22.5°C, 48.2%, 2800mV    ║
║ [15:30:50] 📡 Ruuvi 052D detectado            ║
║ [15:35:00] 🔄 Iniciando sincronización...      ║
║ [15:35:02] ✅ Datos enviados exitosamente      ║
╚════════════════════════════════════════════════╝
```

Características:
- **Fondo verde**: Última sincronización exitosa con la API
- **Fondo rojo**: Error en la última sincronización
- **Timestamp**: Hora y fecha actual actualizada cada segundo
- **Estado de sensores** (esquina superior derecha): Muestra cuántos sensores están online/offline
  - Se considera "online" si se ha detectado en los últimos 2 minutos
- **Logs de actividad**: Últimas 10 acciones del sistema en tiempo real

Ventajas:
- ✅ No requiere display gráfico (funciona por SSH)
- ✅ Compatible con screen/tmux para ejecutar en segundo plano
- ✅ Ultra ligero (~7.5MB binary, sin CGO)
- ✅ Cross-compila fácilmente para ARM64 (Raspberry Pi)

La interfaz se actualiza automáticamente cada vez que se envían datos a la API.

## Integración con API

El programa envía automáticamente los datos a la API de Larvai:

- **Endpoint**: `POST https://go.larvai.com/api/v1/sensors/{UUID_SENSOR}`
- **Primera sincronización**: 10 segundos después del inicio
- **Frecuencia**: Cada 5 minutos después de la primera sincronización
- **Datos enviados**:
  ```json
  {
    "temperature": 23.5,
    "humidity": 45.2,
    "battery": 2800
  }
  ```
- **UUID del sensor**: Se utiliza la dirección MAC del dispositivo Bluetooth

El programa continuará escaneando sensores en tiempo real y mostrando datos en la consola, mientras que en segundo plano enviará las últimas lecturas a la API y actualizará la GUI con el estado.

## Capa de Seguridad

El programa implementa una lista blanca de sensores autorizados:

- 🔒 **Primera ejecución**: Registra automáticamente todos los sensores RuuviTag detectados
- 💾 **Persistencia**: Guarda las direcciones MAC en `authorized_sensors.json`
- 🛡️ **Protección**: Solo lee datos de sensores previamente autorizados
- 🔄 **Flexibilidad**: Puedes re-registrar sensores cuando sea necesario

**Beneficios:**
- Evita leer datos de sensores desconocidos en entornos con múltiples RuuviTags
- Protege contra sensores no autorizados
- Útil en espacios compartidos o públicos

## Notas

- Los sensores RuuviTag transmiten datos continuamente sin necesidad de conexión
- El programa detecta automáticamente los dispositivos con nombre "Ruuvi" o Manufacturer ID 0x0499
- Compatible con formato RAWv2 (identificador 0x05)
- El archivo `authorized_sensors.json` se crea automáticamente en la primera ejecución
