package ui

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	Reset  = "\033[0m"
	Red    = "\033[41m" // Red background
	Green  = "\033[42m" // Green background
	White  = "\033[97m" // White text
	Clear  = "\033[2J\033[H" // Clear screen + move to top
	Bold   = "\033[1m"
)

type TerminalUI struct {
	status      string
	success     bool
	sensors     string
	logs        []string
	timestamp   string
	mu          sync.Mutex
	maxLogLines int
}

func NewTerminalUI() *TerminalUI {
	return &TerminalUI{
		status:      "Inicializando sistema...",
		success:     false,
		sensors:     "Sensores: --",
		logs:        []string{"Esperando actividad..."},
		maxLogLines: 10,
	}
}

func (t *TerminalUI) Start() {
	// Start timestamp updater
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			t.mu.Lock()
			t.timestamp = time.Now().Format("15:04:05 - 02/01/2006")
			t.mu.Unlock()
			t.Render()
		}
	}()

	// Initial render
	t.Render()
}

func (t *TerminalUI) Render() {
	t.mu.Lock()
	defer t.mu.Unlock()

	fmt.Print(Clear)

	// Determine background color and icon
	bg := Red
	icon := "✗"
	statusText := "ERROR"
	if t.success {
		bg = Green
		icon = "✓"
		statusText = "EXITOSA"
	}

	// Top border
	fmt.Println("╔════════════════════════════════════════════════╗")

	// Header with sensor status
	headerLeft := "Insectius Monitor"
	headerRight := t.sensors
	padding := 48 - len(headerLeft) - len(headerRight)
	if padding < 0 {
		padding = 0
	}
	fmt.Printf("║ %s%s%s ║\n", headerLeft, strings.Repeat(" ", padding), headerRight)

	// Separator
	fmt.Println("╠════════════════════════════════════════════════╣")

	// Status section with colored background (centered)
	fmt.Printf("║                                                ║\n")

	// Center the icon and status
	iconLine := fmt.Sprintf("%s     %s     %s", bg+White, icon, Reset)
	iconPadding := (48 - 11) / 2 // Account for ANSI codes visually
	fmt.Printf("║%s%s%s║\n", strings.Repeat(" ", iconPadding), iconLine, strings.Repeat(" ", 48-iconPadding-11))

	statusLine := truncate(statusText, 20)
	statusPadding := (48 - len(statusLine)) / 2
	fmt.Printf("║%s%s%s║\n", strings.Repeat(" ", statusPadding), statusLine, strings.Repeat(" ", 48-statusPadding-len(statusLine)))

	fmt.Printf("║                                                ║\n")

	// Timestamp (centered)
	tsLine := t.timestamp
	if tsLine == "" {
		tsLine = time.Now().Format("15:04:05 - 02/01/2006")
	}
	tsPadding := (48 - len(tsLine)) / 2
	fmt.Printf("║%s%s%s║\n", strings.Repeat(" ", tsPadding), tsLine, strings.Repeat(" ", 48-tsPadding-len(tsLine)))

	fmt.Printf("║                                                ║\n")

	// Activity section separator
	fmt.Println("╠════════════════════════════════════════════════╣")

	// Activity title
	activityTitle := "Actividad del Sistema"
	activityPadding := (48 - len(activityTitle)) / 2
	fmt.Printf("║%s%s%s%s║\n", strings.Repeat(" ", activityPadding), Bold, activityTitle, Reset+strings.Repeat(" ", 48-activityPadding-len(activityTitle)))

	fmt.Println("╠════════════════════════════════════════════════╣")

	// Activity logs
	displayLogs := t.logs
	if len(displayLogs) > t.maxLogLines {
		displayLogs = displayLogs[:t.maxLogLines]
	}

	for _, log := range displayLogs {
		fmt.Printf("║ %-46s ║\n", truncate(log, 46))
	}

	// Fill remaining lines if needed
	for i := len(displayLogs); i < t.maxLogLines; i++ {
		fmt.Printf("║%s║\n", strings.Repeat(" ", 48))
	}

	// Bottom border
	fmt.Println("╚════════════════════════════════════════════════╝")
}

func (t *TerminalUI) UpdateStatus(success bool, msg string) {
	t.mu.Lock()
	t.success = success
	t.status = msg
	t.mu.Unlock()
	t.Render()
}

func (t *TerminalUI) AddLog(msg string) {
	t.mu.Lock()
	// Add to beginning of logs
	t.logs = append([]string{msg}, t.logs...)
	if len(t.logs) > t.maxLogLines*2 {
		t.logs = t.logs[:t.maxLogLines*2]
	}
	t.mu.Unlock()
	t.Render()
}

func (t *TerminalUI) UpdateSensors(online, total int) {
	t.mu.Lock()
	if online == total {
		t.sensors = fmt.Sprintf("Sensores: %d/%d ✓", online, total)
	} else {
		t.sensors = fmt.Sprintf("Sensores: %d/%d", online, total)
	}
	t.mu.Unlock()
	t.Render()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
