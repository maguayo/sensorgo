package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sensorsgo/ui"
	"sync"
	"time"

	"tinygo.org/x/bluetooth"
)

const (
	configFile   = "authorized_sensors.json"
	apiURL       = "https://go.larvai.com/api/v1/sensors"
	sendInterval = 5 * time.Minute
)

var (
	terminalUI    *ui.TerminalUI
	lastSeenMap   map[string]time.Time
	lastSeenMutex sync.Mutex
	onlineTimeout = 2 * time.Minute // Sensor offline si no se ve en 2 minutos
	apiKey        string            // API key para autenticación
)

// RuuviData contiene los datos parseados del sensor
type RuuviData struct {
	Temperature float64
	Humidity    float64
	Pressure    float64
	Battery     uint16
	TxPower     int8
	MAC         string
}

// AuthorizedSensor representa un sensor autorizado
type AuthorizedSensor struct {
	MAC         string    `json:"mac"`
	Name        string    `json:"name"`
	RegisteredAt time.Time `json:"registered_at"`
}

// Config contiene la configuración de sensores autorizados
type Config struct {
	Sensors []AuthorizedSensor `json:"authorized_sensors"`
}

// SensorPayload representa los datos a enviar a la API
type SensorPayload struct {
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
	Battery     uint16  `json:"battery"`
	Hostname    string  `json:"hostname"`
}

func main() {
	// Flags de línea de comandos
	reregister := flag.Bool("reregister", false, "Re-registrar sensores (sobrescribe la lista actual)")
	flag.Parse()

	// Cargar API key
	err := loadAPIKey()
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		fmt.Println("\n💡 Crea un archivo ~/.insectius-monitor con tu API key:")
		fmt.Println("   echo 'tu-api-key-aqui' > ~/.insectius-monitor")
		fmt.Println("   chmod 600 ~/.insectius-monitor")
		return
	}

	adapter := bluetooth.DefaultAdapter

	fmt.Println("🔍 DEBUG: Intentando habilitar adaptador Bluetooth...")

	// Intentar habilitar Bluetooth con reintentos más largos
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		fmt.Printf("   Intento %d/%d...\n", i+1, maxRetries)
		err = adapter.Enable()
		if err == nil {
			fmt.Println("✅ DEBUG: adapter.Enable() exitoso")
			break
		}
		fmt.Printf("   Error: %v\n", err)
		if i < maxRetries-1 {
			fmt.Printf("⚠️  Reintentando en 3 segundos...\n")
			time.Sleep(3 * time.Second)
		}
	}
	if err != nil {
		fmt.Printf("❌ Error habilitando Bluetooth después de %d intentos: %v\n", maxRetries, err)
		return
	}

	// Esperar más tiempo a que Bluetooth esté completamente listo
	fmt.Println("⏳ Esperando 10 segundos para que Bluetooth esté listo...")
	time.Sleep(10 * time.Second)
	fmt.Println("✅ Espera completada")

	// Verificar si existe el archivo de configuración
	config, firstRun := loadConfig()

	if *reregister {
		fmt.Println("🔄 Modo re-registro activado. Se sobrescribirá la lista actual de sensores.")
		firstRun = true
		config = &Config{Sensors: []AuthorizedSensor{}}
	}

	if firstRun {
		fmt.Println("🆕 Primera ejecución detectada.")
		fmt.Println("🔍 Escaneando sensores RuuviTag para registrarlos...")
		fmt.Println("⏱️  Escaneando durante 10 segundos...")

		foundSensors := make(map[string]AuthorizedSensor)

		// Timer para detener el escaneo después de 10 segundos
		go func() {
			time.Sleep(10 * time.Second)
			adapter.StopScan()
		}()

		// Escanear hasta que se detenga
		err = adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
			if isRuuviTag(device) {
				mac := device.Address.String()
				if _, exists := foundSensors[mac]; !exists {
					sensor := AuthorizedSensor{
						MAC:         mac,
						Name:        device.LocalName(),
						RegisteredAt: time.Now(),
					}
					foundSensors[mac] = sensor
					fmt.Printf("✅ Sensor registrado: %s (%s)\n", sensor.Name, sensor.MAC)
				}
			}
		})

		if err != nil {
			fmt.Printf("❌ Error escaneando: %v\n", err)
			return
		}

		// Guardar sensores encontrados
		for _, sensor := range foundSensors {
			config.Sensors = append(config.Sensors, sensor)
		}

		if len(config.Sensors) == 0 {
			fmt.Println("❌ No se encontraron sensores RuuviTag. Asegúrate de que estén encendidos y cerca.")
			return
		}

		err = saveConfig(config)
		if err != nil {
			fmt.Printf("❌ Error guardando configuración: %v\n", err)
			return
		}

		fmt.Printf("\n✅ Registro completado. %d sensores autorizados guardados en %s\n", len(config.Sensors), configFile)
		fmt.Println("\n📋 Sensores autorizados:")
		for i, sensor := range config.Sensors {
			fmt.Printf("   %d. %s (%s)\n", i+1, sensor.Name, sensor.MAC)
		}
		fmt.Println("\n🔒 A partir de ahora, solo se leerán datos de estos sensores.")
		fmt.Println("💡 Para re-registrar sensores, ejecuta: go run main.go -reregister")
		return
	}

	// Modo normal: iniciar terminal UI y escaneo
	startMonitoring(adapter, config)
}

// startMonitoring inicia el monitoreo de sensores y la GUI
func startMonitoring(adapter *bluetooth.Adapter, config *Config) {
	fmt.Printf("🔒 Modo seguro: solo se leerán %d sensores autorizados\n", len(config.Sensors))
	fmt.Println("📋 Sensores autorizados:")
	for i, sensor := range config.Sensors {
		fmt.Printf("   %d. %s (%s)\n", i+1, sensor.Name, sensor.MAC)
	}
	fmt.Printf("\n🔍 Escaneando sensores y enviando datos a la API cada %v...\n", sendInterval)

	// Inicializar mapa de última vez visto
	lastSeenMap = make(map[string]time.Time)

	// Crear mapa de sensores autorizados para búsqueda rápida
	authorizedMACs := make(map[string]bool)
	for _, sensor := range config.Sensors {
		authorizedMACs[sensor.MAC] = true
	}

	// Mapa para almacenar las últimas lecturas de cada sensor
	var lastReadings = make(map[string]*RuuviData)
	var mu sync.Mutex

	// Variable para controlar si es la primera sincronización
	firstSync := true

	// Goroutine para enviar datos (inmediato y luego cada 5 minutos)
	go func() {
		addLog(fmt.Sprintf("⏰ Sincronización automática cada %v", sendInterval))

		// Función para sincronizar
		syncData := func() {
			if firstSync {
				addLog("🔄 Ejecutando primera sincronización...")
				firstSync = false
			} else {
				addLog("🔄 Iniciando sincronización programada...")
			}

			mu.Lock()
			count := 0
			for mac, data := range lastReadings {
				if data != nil {
					count++
					go sendToAPI(mac, data)
				}
			}
			mu.Unlock()

			if count == 0 {
				addLog("⚠️  No hay datos para sincronizar")
			} else {
				addLog(fmt.Sprintf("📤 Sincronizando %d sensor(es)", count))
			}
		}

		// Esperar 10 segundos para recolectar datos, luego primera sincronización
		time.Sleep(10 * time.Second)
		syncData()

		// Luego continuar cada 5 minutos
		ticker := time.NewTicker(sendInterval)
		defer ticker.Stop()

		for range ticker.C {
			syncData()
		}
	}()

	// Goroutine para verificar estado de sensores cada 5 minutos
	go func() {
		ticker := time.NewTicker(sendInterval)
		defer ticker.Stop()

		for range ticker.C {
			updateSensorStatus(config)
		}
	}()

	// Goroutine para escanear dispositivos
	go func() {
		addLog("🔍 Iniciando escaneo de sensores...")
		fmt.Println("🔍 DEBUG: Iniciando goroutine de escaneo...")

		// Intentar escaneo con reintentos
		maxScanRetries := 5
		for attempt := 0; attempt < maxScanRetries; attempt++ {
			fmt.Printf("🔍 DEBUG: Intento de escaneo %d/%d\n", attempt+1, maxScanRetries)

			err := adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
			// Buscar RuuviTag en el nombre o en manufacturer data
			if isRuuviTag(device) {
				mac := device.Address.String()

				// Verificar si el sensor está autorizado
				if !authorizedMACs[mac] {
					// Sensor no autorizado, ignorar
					return
				}

				// Parsear datos del manufacturer data
				if data := parseRuuviData(device); data != nil {
					// Marcar sensor como online
					markSensorOnline(mac)

					// Actualizar última lectura
					mu.Lock()
					lastReadings[mac] = data
					mu.Unlock()

					sensorName := device.LocalName()
					if sensorName == "" {
						sensorName = mac[:17] // Usar MAC si no hay nombre
					}

					addLog(fmt.Sprintf("📡 %s detectado", sensorName))
					addLog(fmt.Sprintf("📊 Datos: %.1f°C, %.1f%% humedad, %dmV", data.Temperature, data.Humidity, data.Battery))

					// Actualizar estado de sensores
					updateSensorStatus(config)

					fmt.Printf("\n📡 Sensor: %s\n", device.LocalName())
					fmt.Printf("   🌡️  Temperatura: %.2f °C\n", data.Temperature)
					fmt.Printf("   💧 Humedad: %.2f %%\n", data.Humidity)
					fmt.Printf("   📊 Presión: %.2f hPa\n", data.Pressure)
					fmt.Printf("   🔋 Batería: %d mV\n", data.Battery)
				}
			}
			})

			if err != nil {
				fmt.Printf("❌ Error escaneando (intento %d): %v\n", attempt+1, err)
				addLog(fmt.Sprintf("❌ Error en escaneo: %v", err))

				if attempt < maxScanRetries-1 {
					fmt.Printf("⏳ Esperando 5 segundos antes de reintentar...\n")
					time.Sleep(5 * time.Second)
					continue
				}
			} else {
				// Scan terminó sin error (no debería pasar normalmente)
				break
			}
		}
		fmt.Println("❌ DEBUG: Todos los intentos de escaneo fallaron")
	}()

	// Iniciar terminal UI
	startTerminalUI()
}

// startTerminalUI inicia la interfaz de terminal
func startTerminalUI() {
	terminalUI = ui.NewTerminalUI()
	terminalUI.Start()

	// Block forever
	select {}
}

// sendToAPI envía los datos del sensor a la API
func sendToAPI(sensorUUID string, data *RuuviData) {
	addLog(fmt.Sprintf("📤 Enviando datos a la API (Temp: %.1f°C, Hum: %.1f%%, Bat: %dmV)", data.Temperature, data.Humidity, data.Battery))

	// Obtener hostname del sistema
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	payload := SensorPayload{
		Temperature: data.Temperature,
		Humidity:    data.Humidity,
		Battery:     data.Battery,
		Hostname:    hostname,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("⚠️  Error serializando datos para %s: %v\n", sensorUUID, err)
		addLog("❌ Error serializando datos")
		updateGUIStatus(false)
		return
	}

	url := fmt.Sprintf("%s/%s", apiURL, sensorUUID)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("⚠️  Error creando request para %s: %v\n", sensorUUID, err)
		addLog("❌ Error creando request HTTP")
		updateGUIStatus(false)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("⚠️  Error enviando datos para %s: %v\n", sensorUUID, err)
		addLog(fmt.Sprintf("❌ Error de conexión: %v", err))
		updateGUIStatus(false)
		return
	}
	defer resp.Body.Close()

	// Leer el response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("⚠️  Error leyendo response body: %v\n", err)
	}
	bodyString := string(bodyBytes)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("✅ Datos enviados para sensor %s (Temp: %.2f°C, Hum: %.2f%%, Bat: %dmV)\n",
			sensorUUID, data.Temperature, data.Humidity, data.Battery)
		addLog(fmt.Sprintf("✅ Datos enviados exitosamente (HTTP %d)", resp.StatusCode))

		// Mostrar response si hay contenido
		if len(bodyString) > 0 && bodyString != "{}" {
			fmt.Printf("   Response: %s\n", bodyString)
		}

		updateGUIStatus(true)
	} else {
		// Error HTTP - mostrar detalles completos
		fmt.Printf("\n❌ Error HTTP %d al enviar datos para %s\n", resp.StatusCode, sensorUUID)
		fmt.Printf("   URL: %s\n", url)
		fmt.Printf("   Payload enviado: %s\n", string(jsonData))
		fmt.Printf("   Response body: %s\n", bodyString)

		// Intentar parsear el error como JSON para mostrarlo mejor
		var errorResponse map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &errorResponse); err == nil {
			if msg, ok := errorResponse["message"].(string); ok {
				addLog(fmt.Sprintf("❌ HTTP %d: %s", resp.StatusCode, msg))
			} else {
				addLog(fmt.Sprintf("❌ HTTP %d - Ver consola para detalles", resp.StatusCode))
			}
		} else {
			// No es JSON, mostrar el body tal cual (truncado si es muy largo)
			truncated := bodyString
			if len(truncated) > 100 {
				truncated = truncated[:100] + "..."
			}
			addLog(fmt.Sprintf("❌ HTTP %d: %s", resp.StatusCode, truncated))
		}

		updateGUIStatus(false)
	}
}

// updateGUIStatus actualiza el estado visual de la UI
func updateGUIStatus(success bool) {
	if terminalUI != nil {
		msg := "Ultima sincronizacion"
		terminalUI.UpdateStatus(success, msg)
	}
}

// addLog añade un mensaje al log de actividad
func addLog(message string) {
	timestamp := time.Now().Format("15:04:05")
	logLine := fmt.Sprintf("[%s] %s", timestamp, message)

	if terminalUI != nil {
		terminalUI.AddLog(logLine)
	}
}

// updateSensorStatus actualiza el widget de estado de sensores
func updateSensorStatus(config *Config) {
	lastSeenMutex.Lock()
	defer lastSeenMutex.Unlock()

	if terminalUI == nil {
		return
	}

	now := time.Now()
	online := 0

	for _, sensor := range config.Sensors {
		if lastSeen, exists := lastSeenMap[sensor.MAC]; exists {
			if now.Sub(lastSeen) < onlineTimeout {
				online++
			}
		}
	}

	total := len(config.Sensors)
	terminalUI.UpdateSensors(online, total)
}

// markSensorOnline marca un sensor como visto recientemente
func markSensorOnline(mac string) {
	lastSeenMutex.Lock()
	defer lastSeenMutex.Unlock()

	lastSeenMap[mac] = time.Now()
}

// loadAPIKey carga la API key desde el archivo ~/.insectius-monitor
func loadAPIKey() error {
	// Obtener el directorio home del usuario
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error obteniendo directorio home: %w", err)
	}

	apiKeyPath := homeDir + "/.insectius-monitor"

	data, err := os.ReadFile(apiKeyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("archivo %s no encontrado. Crea el archivo con tu API key", apiKeyPath)
		}
		return fmt.Errorf("error leyendo %s: %w", apiKeyPath, err)
	}

	// Limpiar espacios y saltos de línea
	apiKey = string(bytes.TrimSpace(data))

	if apiKey == "" {
		return fmt.Errorf("el archivo %s está vacío. Añade tu API key", apiKeyPath)
	}

	fmt.Printf("✅ API key cargada desde %s\n", apiKeyPath)
	return nil
}

// loadConfig carga la configuración desde el archivo JSON
// Retorna la configuración y un booleano indicando si es la primera ejecución
func loadConfig() (*Config, bool) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Primera ejecución
			return &Config{Sensors: []AuthorizedSensor{}}, true
		}
		fmt.Printf("⚠️  Error leyendo archivo de configuración: %v\n", err)
		return &Config{Sensors: []AuthorizedSensor{}}, true
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		fmt.Printf("⚠️  Error parseando configuración: %v\n", err)
		return &Config{Sensors: []AuthorizedSensor{}}, true
	}

	if len(config.Sensors) == 0 {
		return &config, true
	}

	return &config, false
}

// saveConfig guarda la configuración en el archivo JSON
func saveConfig(config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializando configuración: %w", err)
	}

	err = os.WriteFile(configFile, data, 0644)
	if err != nil {
		return fmt.Errorf("error guardando archivo: %w", err)
	}

	return nil
}

// isRuuviTag verifica si el dispositivo es un RuuviTag
func isRuuviTag(device bluetooth.ScanResult) bool {
	// Verificar por nombre
	name := device.LocalName()
	if len(name) >= 5 && name[:5] == "Ruuvi" {
		return true
	}

	// Verificar por Manufacturer ID (0x0499 para Ruuvi Innovations Ltd)
	if device.ManufacturerData() != nil {
		for _, mfg := range device.ManufacturerData() {
			if mfg.CompanyID == 0x0499 {
				return true
			}
		}
	}

	return false
}

// parseRuuviData parsea los datos del formato RAWv2 (más común)
func parseRuuviData(device bluetooth.ScanResult) *RuuviData {
	mfgData := device.ManufacturerData()
	if mfgData == nil {
		return nil
	}

	for _, mfg := range mfgData {
		if mfg.CompanyID == 0x0499 && len(mfg.Data) >= 24 {
			data := mfg.Data

			// Verificar formato RAWv2 (0x05)
			if data[0] != 0x05 {
				continue
			}

			result := &RuuviData{}

			// Temperatura (bytes 1-2): signed int16, escala 0.005
			tempRaw := int16(binary.BigEndian.Uint16(data[1:3]))
			result.Temperature = float64(tempRaw) * 0.005

			// Humedad (bytes 3-4): uint16, escala 0.0025
			humRaw := binary.BigEndian.Uint16(data[3:5])
			result.Humidity = float64(humRaw) * 0.0025

			// Presión (bytes 5-6): uint16, offset 50000 Pa
			pressRaw := binary.BigEndian.Uint16(data[5:7])
			result.Pressure = (float64(pressRaw) + 50000.0) / 100.0 // Convertir a hPa

			// Batería (bytes 13-14): primeros 11 bits en mV
			battRaw := binary.BigEndian.Uint16(data[13:15])
			result.Battery = (battRaw >> 5) + 1600

			// TX Power (byte 14): últimos 5 bits, signed
			txPowerRaw := int8(data[14] & 0x1F)
			if txPowerRaw&0x10 != 0 {
				txPowerRaw |= ^0x1F
			}
			result.TxPower = txPowerRaw * 2

			// MAC address (bytes 18-23)
			result.MAC = fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X",
				data[18], data[19], data[20], data[21], data[22], data[23])

			return result
		}
	}

	return nil
}
