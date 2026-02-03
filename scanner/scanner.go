package scanner

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// RuuviData contains parsed sensor data
type RuuviData struct {
	MAC         string
	Name        string
	Temperature float64
	Humidity    float64
	Pressure    float64
	Battery     uint16
}

// Scanner handles BLE scanning using shell commands
type Scanner struct {
	running     bool
	stopChan    chan struct{}
	mu          sync.Mutex
	lastData    map[string]*RuuviData
	onData      func(*RuuviData)
}

// NewScanner creates a new shell-based BLE scanner
func NewScanner() *Scanner {
	return &Scanner{
		stopChan: make(chan struct{}),
		lastData: make(map[string]*RuuviData),
	}
}

// SetCallback sets the callback function for when data is received
func (s *Scanner) SetCallback(callback func(*RuuviData)) {
	s.onData = callback
}

// GetLastData returns the last reading for a sensor
func (s *Scanner) GetLastData(mac string) *RuuviData {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastData[mac]
}

// GetAllData returns all last readings
func (s *Scanner) GetAllData() map[string]*RuuviData {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make(map[string]*RuuviData)
	for k, v := range s.lastData {
		result[k] = v
	}
	return result
}

// Start begins scanning for RuuviTag devices
func (s *Scanner) Start() error {
	s.running = true

	// Use btmon to capture BLE advertisements
	go s.runBtmon()

	return nil
}

// Stop stops the scanner
func (s *Scanner) Stop() {
	s.running = false
	close(s.stopChan)
}

// runBtmon runs btmon to capture BLE data
func (s *Scanner) runBtmon() {
	for s.running {
		// Run hcitool lescan in background and capture with hcidump
		s.scanWithHcidump()

		if s.running {
			time.Sleep(2 * time.Second)
		}
	}
}

// scanWithHcidump uses hcitool and hcidump to capture BLE advertisements
func (s *Scanner) scanWithHcidump() {
	// Start lescan in background
	lescan := exec.Command("hcitool", "lescan", "--duplicates", "--passive")
	lescan.Start()

	// Start hcidump to capture raw data
	hcidump := exec.Command("hcidump", "--raw")
	stdout, err := hcidump.StdoutPipe()
	if err != nil {
		fmt.Printf("Error creating hcidump pipe: %v\n", err)
		lescan.Process.Kill()
		return
	}

	if err := hcidump.Start(); err != nil {
		fmt.Printf("Error starting hcidump: %v\n", err)
		lescan.Process.Kill()
		return
	}

	// Read output
	scanner := bufio.NewScanner(stdout)
	var currentPacket bytes.Buffer

	go func() {
		for scanner.Scan() {
			line := scanner.Text()

			// hcidump output format: lines starting with > or < are packet starts
			if strings.HasPrefix(line, ">") || strings.HasPrefix(line, "<") {
				// Process previous packet
				if currentPacket.Len() > 0 {
					s.processPacket(currentPacket.String())
				}
				currentPacket.Reset()
				currentPacket.WriteString(line)
			} else {
				currentPacket.WriteString(line)
			}
		}
	}()

	// Run for 30 seconds then restart
	select {
	case <-time.After(30 * time.Second):
	case <-s.stopChan:
	}

	lescan.Process.Kill()
	hcidump.Process.Kill()
}

// processPacket processes a raw HCI packet looking for RuuviTag data
func (s *Scanner) processPacket(packet string) {
	// Remove whitespace and convert to hex bytes
	packet = strings.ReplaceAll(packet, " ", "")
	packet = strings.ReplaceAll(packet, "\n", "")
	packet = strings.ReplaceAll(packet, ">", "")
	packet = strings.ReplaceAll(packet, "<", "")

	// Look for Ruuvi manufacturer ID (0x0499 = little endian 9904)
	if !strings.Contains(strings.ToLower(packet), "9904") {
		return
	}

	// Try to parse the packet
	data, err := hex.DecodeString(packet)
	if err != nil {
		return
	}

	// Find Ruuvi data (manufacturer ID 0x0499 followed by format 0x05)
	for i := 0; i < len(data)-26; i++ {
		if data[i] == 0x99 && data[i+1] == 0x04 && data[i+2] == 0x05 {
			ruuvi := s.parseRuuviRAWv2(data[i+2:])
			if ruuvi != nil {
				s.mu.Lock()
				s.lastData[ruuvi.MAC] = ruuvi
				s.mu.Unlock()

				if s.onData != nil {
					s.onData(ruuvi)
				}
			}
			return
		}
	}
}

// parseRuuviRAWv2 parses RAWv2 format data
func (s *Scanner) parseRuuviRAWv2(data []byte) *RuuviData {
	if len(data) < 24 {
		return nil
	}

	// Format byte should be 0x05
	if data[0] != 0x05 {
		return nil
	}

	ruuvi := &RuuviData{}

	// Temperature (bytes 1-2): signed int16, scale 0.005
	tempRaw := int16(uint16(data[1])<<8 | uint16(data[2]))
	ruuvi.Temperature = float64(tempRaw) * 0.005

	// Humidity (bytes 3-4): uint16, scale 0.0025
	humRaw := uint16(data[3])<<8 | uint16(data[4])
	ruuvi.Humidity = float64(humRaw) * 0.0025

	// Pressure (bytes 5-6): uint16, offset 50000 Pa
	pressRaw := uint16(data[5])<<8 | uint16(data[6])
	ruuvi.Pressure = (float64(pressRaw) + 50000.0) / 100.0

	// Battery (bytes 13-14): first 11 bits in mV
	battRaw := uint16(data[11])<<8 | uint16(data[12])
	ruuvi.Battery = (battRaw >> 5) + 1600

	// MAC address (bytes 18-23)
	if len(data) >= 24 {
		ruuvi.MAC = fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X",
			data[18], data[19], data[20], data[21], data[22], data[23])
	}

	return ruuvi
}
