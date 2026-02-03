# Terminal UI Migration - Changelog

## Changes Made

### Problem Solved
Fyne GUI required CGO and OpenGL, making cross-compilation for Raspberry Pi ARM64 impossible without complex toolchains.

### Solution
Replaced Fyne with a pure Go terminal-based UI using ANSI escape codes - no external dependencies beyond stdlib.

## Files Modified

### New Files Created
1. **ui/terminal.go** - Terminal UI implementation using ANSI escape codes
   - `TerminalUI` struct with thread-safe rendering
   - Real-time status updates with colored backgrounds (green/red)
   - Activity log display (last 10 entries)
   - Sensor status display
   - Auto-updating timestamp

2. **build-pi.sh** - Build script specifically for Raspberry Pi ARM64

### Modified Files

1. **main.go**
   - Removed Fyne imports (lines 16-20)
   - Added `sensorsgo/ui` import
   - Removed Fyne-related global variables (statusLabel, statusIcon, background, etc.)
   - Added `terminalUI *ui.TerminalUI` global
   - Replaced `startGUI()` with `startTerminalUI()`
   - Updated `updateGUIStatus()` to use terminal UI
   - Simplified `addLog()` to use terminal UI
   - Updated `updateSensorStatus()` to use terminal UI

2. **go.mod**
   - Removed `fyne.io/fyne/v2` dependency
   - Removed all Fyne-related transitive dependencies
   - Result: Only `tinygo.org/x/bluetooth` and minimal stdlib dependencies remain

3. **build.sh**
   - Added `CGO_ENABLED=0` flag for portable compilation

4. **build-and-deploy.sh**
   - Added `CGO_ENABLED=0` flag for portable compilation

5. **README.md**
   - Updated feature list to mention terminal UI
   - Added visual example of terminal interface
   - Added instructions for screen/tmux usage
   - Added cross-compilation notes

## Benefits

✅ **Zero CGO dependencies** - Pure Go compilation
✅ **Cross-compiles easily** - Works for ARM64 without custom toolchains
✅ **Lightweight binary** - ~7.5MB (vs ~23MB with Fyne)
✅ **SSH-friendly** - Works over SSH without display
✅ **Screen/tmux compatible** - Can run in background sessions
✅ **No display required** - Perfect for headless servers
✅ **Faster compilation** - No C compilation step

## Verification Steps

1. ✅ Compilation test: `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build`
2. ✅ Binary size: 7.5MB (down from 23MB)
3. ✅ Binary type: Statically linked ELF (no dynamic dependencies)
4. ✅ Dependencies cleaned: `go mod tidy` removed Fyne deps

## Build Commands

### For Raspberry Pi ARM64:
```bash
./build-pi.sh
# or manually:
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o insectius-monitor main.go
```

### For Linux AMD64:
```bash
./build.sh
# or manually:
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o insectius-monitor main.go
```

## Visual Comparison

### Before (Fyne GUI)
- Required CGO + OpenGL
- ~23MB binary with dynamic dependencies
- Needed X11/Wayland display
- Full-screen graphical interface
- Cross-compilation required complex toolchains

### After (Terminal UI)
- Pure Go stdlib + ANSI codes
- ~7.5MB static binary
- Works over SSH
- ASCII box-drawing interface
- Cross-compiles with single command

## Terminal UI Layout

```
╔════════════════════════════════════════════════╗
║ Insectius Monitor             Sensores: 2/2 ✓ ║
╠════════════════════════════════════════════════╣
║                                                ║
║                     ✓                          ║  <- Green/Red background
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
╚════════════════════════════════════════════════╝
```

## Testing

To test the compilation:
```bash
# Test ARM64 build
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o test-arm64

# Verify it's static
file test-arm64
# Output: ELF 64-bit LSB executable, ARM aarch64, statically linked

# Check size
ls -lh test-arm64
# Output: 7.5M
```

## Deployment

The program now works seamlessly on:
- Raspberry Pi (ARM64)
- Any Linux server (AMD64/ARM64)
- Via SSH (no display needed)
- In screen/tmux sessions
- As systemd service (logs visible via journalctl)

No changes needed to deployment scripts or systemd service files.
