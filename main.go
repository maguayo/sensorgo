package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"image/color"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"tinygo.org/x/bluetooth"
)

const (
	configFile   = "authorized_sensors.json"
	apiURL       = "https://go.larvai.com/api/v1/sensors"
	sendInterval = 5 * time.Minute
)

var (
	lastHTTPStatus bool // true = success, false = error
	statusLabel    *canvas.Text
	statusIcon     *canvas.Text
	statusMutex    sync.Mutex
	activityLog    *widget.Label
	logMessages    []string
	logMutex       sync.Mutex
	maxLogLines    = 15
	sensorStatus   *widget.Label
	lastSeenMap    map[string]time.Time
	lastSeenMutex  sync.Mutex
	onlineTimeout  = 2 * time.Minute // Sensor offline si no se ve en 2 minutos
	background     *canvas.Rectangle
	timestampLabel *canvas.Text
	apiKey         string // API key para autenticación
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
	err = adapter.Enable()
	if err != nil {
		fmt.Printf("❌ Error habilitando Bluetooth: %v\n", err)
		return
	}

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

	// Modo normal: iniciar GUI y escaneo
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
			fmt.Printf("❌ Error escaneando: %v\n", err)
		}
	}()

	// Iniciar GUI
	startGUI()
}

// startGUI inicia la interfaz gráfica
func startGUI() {
	myApp := app.New()
	myWindow := myApp.NewWindow("RuuviTag Monitor")

	// Crear fondo de color (gris por defecto, cambiará a verde/rojo)
	background = canvas.NewRectangle(color.RGBA{R: 50, G: 50, B: 50, A: 255})

	// Crear icono grande
	statusIcon = canvas.NewText("...", color.RGBA{R: 200, G: 200, B: 200, A: 255})
	statusIcon.TextSize = 200
	statusIcon.Alignment = fyne.TextAlignCenter

	// Crear label de estado
	statusLabel = canvas.NewText("Inicializando sistema...", color.RGBA{R: 200, G: 200, B: 200, A: 255})
	statusLabel.TextSize = 20
	statusLabel.Alignment = fyne.TextAlignCenter

	// Crear timestamp label
	timestampLabel = canvas.NewText("", color.RGBA{R: 200, G: 200, B: 200, A: 255})
	timestampLabel.TextSize = 16
	timestampLabel.Alignment = fyne.TextAlignCenter

	// Crear widget de estado de sensores (esquina superior derecha)
	sensorStatus = widget.NewLabel("Sensores: --")
	sensorStatus.Alignment = fyne.TextAlignTrailing
	sensorStatus.TextStyle = fyne.TextStyle{Monospace: true}

	// Crear área de logs de actividad con fondo semi-transparente
	activityLog = widget.NewLabel("Esperando actividad...\n")
	activityLog.Alignment = fyne.TextAlignLeading
	activityLog.Wrapping = fyne.TextWrapWord

	// Fondo para los logs
	logBackground := canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: 200})

	// Título de la sección de logs
	logTitle := canvas.NewText("Actividad del Sistema", color.RGBA{R: 255, G: 255, B: 255, A: 255})
	logTitle.TextSize = 16
	logTitle.Alignment = fyne.TextAlignCenter

	// Separador visual
	separator := canvas.NewRectangle(color.RGBA{R: 255, G: 255, B: 255, A: 100})
	separator.SetMinSize(fyne.NewSize(0, 2))

	// Actualizar timestamp cada segundo
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			timestampLabel.Text = time.Now().Format("15:04:05 - 02/01/2006")
			timestampLabel.Refresh()
		}
	}()

	// Header con estado de sensores en la esquina
	headerRight := container.NewVBox(sensorStatus)

	// Layout con logs en la parte inferior
	topSection := container.NewVBox(
		container.NewCenter(statusIcon),
		container.NewCenter(statusLabel),
		container.NewCenter(timestampLabel),
	)

	// Logs con fondo oscuro
	logsWithBackground := container.NewStack(
		logBackground,
		container.NewVBox(
			separator,
			container.NewCenter(logTitle),
			activityLog,
		),
	)

	bottomSection := logsWithBackground

	content := container.NewBorder(
		topSection,    // top
		bottomSection, // bottom
		nil,           // left
		headerRight,   // right (sensor status)
		nil,           // center (vacío)
	)

	// Stack: background detrás, contenido delante
	finalContent := container.NewStack(background, content)

	myWindow.SetContent(finalContent)
	myWindow.SetFullScreen(true)
	myWindow.ShowAndRun()
}

// sendToAPI envía los datos del sensor a la API
func sendToAPI(sensorUUID string, data *RuuviData) {
	addLog(fmt.Sprintf("📤 Enviando datos a la API (Temp: %.1f°C, Hum: %.1f%%, Bat: %dmV)", data.Temperature, data.Humidity, data.Battery))

	payload := SensorPayload{
		Temperature: data.Temperature,
		Humidity:    data.Humidity,
		Battery:     data.Battery,
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

// updateGUIStatus actualiza el estado visual de la GUI
func updateGUIStatus(success bool) {
	statusMutex.Lock()
	defer statusMutex.Unlock()

	lastHTTPStatus = success

	if statusIcon != nil && statusLabel != nil && background != nil {
		if success {
			// Éxito: pantalla verde con texto OK
			statusIcon.Text = "OK"
			statusIcon.Color = color.RGBA{R: 255, G: 255, B: 255, A: 255} // Blanco
			statusLabel.Text = "Ultima sincronizacion: EXITOSA"
			statusLabel.Color = color.RGBA{R: 255, G: 255, B: 255, A: 255} // Blanco
			timestampLabel.Color = color.RGBA{R: 255, G: 255, B: 255, A: 255} // Blanco
			background.FillColor = color.RGBA{R: 0, G: 150, B: 0, A: 255} // Verde
		} else {
			// Error: pantalla roja con X
			statusIcon.Text = "X"
			statusIcon.Color = color.RGBA{R: 255, G: 255, B: 255, A: 255} // Blanco
			statusLabel.Text = "Ultima sincronizacion: ERROR"
			statusLabel.Color = color.RGBA{R: 255, G: 255, B: 255, A: 255} // Blanco
			timestampLabel.Color = color.RGBA{R: 255, G: 255, B: 255, A: 255} // Blanco
			background.FillColor = color.RGBA{R: 150, G: 0, B: 0, A: 255} // Rojo
		}
		statusIcon.Refresh()
		statusLabel.Refresh()
		timestampLabel.Refresh()
		background.Refresh()
	}
}

// addLog añade un mensaje al log de actividad
func addLog(message string) {
	logMutex.Lock()
	defer logMutex.Unlock()

	timestamp := time.Now().Format("15:04:05")
	logLine := fmt.Sprintf("[%s] %s", timestamp, message)

	// Añadir al principio de la lista
	logMessages = append([]string{logLine}, logMessages...)

	// Mantener solo las últimas N líneas
	if len(logMessages) > maxLogLines {
		logMessages = logMessages[:maxLogLines]
	}

	// Actualizar UI
	if activityLog != nil {
		fullLog := ""
		for _, line := range logMessages {
			fullLog += line + "\n"
		}
		activityLog.SetText(fullLog)
	}
}

// updateSensorStatus actualiza el widget de estado de sensores
func updateSensorStatus(config *Config) {
	lastSeenMutex.Lock()
	defer lastSeenMutex.Unlock()

	if sensorStatus == nil {
		return
	}

	now := time.Now()
	online := 0
	offline := 0

	for _, sensor := range config.Sensors {
		if lastSeen, exists := lastSeenMap[sensor.MAC]; exists {
			if now.Sub(lastSeen) < onlineTimeout {
				online++
			} else {
				offline++
			}
		} else {
			offline++
		}
	}

	total := len(config.Sensors)
	statusText := fmt.Sprintf("Sensores: %d/%d online", online, total)

	if offline > 0 {
		statusText += fmt.Sprintf("\n⚠️ %d offline", offline)
	}

	sensorStatus.SetText(statusText)
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
