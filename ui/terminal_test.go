package ui

import (
	"testing"
	"time"
)

// TestTerminalUIRender is a manual test to visualize the terminal UI
// Run with: go test -v ./ui -run TestTerminalUIRender
func TestTerminalUIRender(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping visual test in short mode")
	}

	ui := NewTerminalUI()

	// Test error state
	t.Log("Testing ERROR state (red background)...")
	ui.UpdateStatus(false, "Ultima sincronizacion")
	ui.UpdateSensors(1, 2)
	ui.AddLog("[15:30:45] 📡 Ruuvi 39B1 detectado")
	ui.AddLog("[15:30:45] 📊 Datos: 22.5°C, 48.2%, 2800mV")
	ui.AddLog("[15:35:00] 🔄 Iniciando sincronización...")
	ui.AddLog("[15:35:02] ❌ Error de conexión: timeout")
	time.Sleep(2 * time.Second)

	// Test success state
	t.Log("Testing SUCCESS state (green background)...")
	ui.UpdateStatus(true, "Ultima sincronizacion")
	ui.UpdateSensors(2, 2)
	ui.AddLog("[15:35:05] 🔄 Reintentando sincronización...")
	ui.AddLog("[15:35:07] ✅ Datos enviados exitosamente")
	time.Sleep(2 * time.Second)

	// Test with more logs
	t.Log("Testing with multiple log entries...")
	for i := 0; i < 15; i++ {
		ui.AddLog("[15:35:10] 📡 Sensor detectado")
		time.Sleep(200 * time.Millisecond)
	}

	t.Log("Visual test complete. UI should have displayed correctly.")
}

// TestTruncate tests the truncate function
func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10c", 10, "exactly10c"},
		{"this is a very long string", 15, "this is a ve..."},
		{"abc", 2, "ab"},
		{"", 5, ""},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

// TestUpdateSensors tests the sensor status formatting
func TestUpdateSensors(t *testing.T) {
	ui := NewTerminalUI()

	tests := []struct {
		online   int
		total    int
		expected string
	}{
		{2, 2, "Sensores: 2/2 ✓"},
		{1, 2, "Sensores: 1/2"},
		{0, 2, "Sensores: 0/2"},
	}

	for _, tt := range tests {
		ui.UpdateSensors(tt.online, tt.total)
		if ui.sensors != tt.expected {
			t.Errorf("UpdateSensors(%d, %d) set sensors to %q, want %q",
				tt.online, tt.total, ui.sensors, tt.expected)
		}
	}
}
