// +build ignore

// Test script to verify Bluetooth scanning works
// Run with: go run test_scan.go

package main

import (
	"fmt"
	"time"

	"tinygo.org/x/bluetooth"
)

func main() {
	fmt.Println("=== Test de Bluetooth ===")
	fmt.Println()

	adapter := bluetooth.DefaultAdapter

	fmt.Println("1. Verificando adaptador...")
	fmt.Printf("   Adaptador: %+v\n", adapter)

	fmt.Println()
	fmt.Println("2. Habilitando adaptador...")
	err := adapter.Enable()
	if err != nil {
		fmt.Printf("   ❌ Error Enable(): %v\n", err)
		fmt.Println()
		fmt.Println("   Intentando de nuevo en 5 segundos...")
		time.Sleep(5 * time.Second)
		err = adapter.Enable()
		if err != nil {
			fmt.Printf("   ❌ Error Enable() segundo intento: %v\n", err)
			return
		}
	}
	fmt.Println("   ✅ Enable() exitoso")

	fmt.Println()
	fmt.Println("3. Esperando 5 segundos...")
	time.Sleep(5 * time.Second)

	fmt.Println()
	fmt.Println("4. Iniciando escaneo (10 segundos)...")
	fmt.Println("   (Deberías ver dispositivos aparecer)")
	fmt.Println()

	// Stop scan after 10 seconds
	go func() {
		time.Sleep(10 * time.Second)
		adapter.StopScan()
		fmt.Println()
		fmt.Println("   Escaneo detenido después de 10 segundos")
	}()

	count := 0
	err = adapter.Scan(func(adapter *bluetooth.Adapter, device bluetooth.ScanResult) {
		count++
		fmt.Printf("   📡 Dispositivo encontrado: %s (%s) RSSI: %d\n",
			device.LocalName(), device.Address.String(), device.RSSI)

		// Check if it's a RuuviTag
		if len(device.LocalName()) >= 5 && device.LocalName()[:5] == "Ruuvi" {
			fmt.Printf("      ⭐ ¡RuuviTag detectado!\n")
		}

		// Check manufacturer data
		for _, mfg := range device.ManufacturerData() {
			if mfg.CompanyID == 0x0499 {
				fmt.Printf("      ⭐ ¡RuuviTag detectado por Manufacturer ID!\n")
			}
		}
	})

	if err != nil {
		fmt.Printf("   ❌ Error Scan(): %v\n", err)
	}

	fmt.Println()
	fmt.Printf("=== Escaneo terminado. Dispositivos encontrados: %d ===\n", count)
}
