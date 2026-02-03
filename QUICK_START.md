# Quick Start Guide - Terminal UI

## What Changed?

Replaced Fyne GUI with a lightweight terminal UI. Your program now:
- Compiles to a 7.5MB static binary (was 23MB)
- Works over SSH without display
- Cross-compiles to Raspberry Pi ARM64 easily

## Build for Raspberry Pi

```bash
./build-pi.sh
```

This creates `insectius-monitor` binary for ARM64.

## Deploy to Raspberry Pi

```bash
# Transfer binary
scp insectius-monitor pi@your-raspberry-pi:/home/pi/

# Transfer install script
scp install.sh pi@your-raspberry-pi:/home/pi/

# SSH into Pi
ssh pi@your-raspberry-pi

# Install
sudo ./install.sh

# Start service
sudo systemctl start insectius-monitor

# View live output
journalctl -u insectius-monitor -f
```

## View Terminal UI Directly

If you want to see the visual terminal UI (instead of journalctl):

```bash
# Stop service first
sudo systemctl stop insectius-monitor

# Run directly
./insectius-monitor
```

You'll see:
```
╔════════════════════════════════════════════════╗
║ Insectius Monitor             Sensores: 2/2 ✓ ║
╠════════════════════════════════════════════════╣
║                     ✓                          ║  <- Green when OK
║                  EXITOSA                       ║
║              17:15:45 - 03/02/2026             ║
╠════════════════════════════════════════════════╣
║              Actividad del Sistema             ║
╠════════════════════════════════════════════════╣
║ [17:15:45] 📡 Ruuvi 39B1 detectado            ║
║ [17:15:45] ✅ Datos enviados exitosamente      ║
╚════════════════════════════════════════════════╝
```

## Run in Background with screen

```bash
# Start screen session
screen -S monitor

# Run program
./insectius-monitor

# Detach (leave running): Ctrl+A then D

# Later, reattach to see UI:
screen -r monitor
```

## Build Commands Reference

| Target | Command |
|--------|---------|
| Raspberry Pi ARM64 | `./build-pi.sh` |
| Linux AMD64 | `./build.sh` |
| Manual (any arch) | `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build` |

## What Didn't Change

- ✅ Bluetooth scanning works the same
- ✅ API integration unchanged
- ✅ Configuration files unchanged
- ✅ systemd service unchanged
- ✅ Install/uninstall scripts unchanged

Only the visual interface changed from GUI to terminal.

## Troubleshooting

### Binary won't run on Raspberry Pi
```bash
# Check architecture
file insectius-monitor
# Should say: ARM aarch64

# Make executable
chmod +x insectius-monitor
```

### Can't see terminal UI colors
Your terminal needs ANSI color support. Most modern terminals (SSH included) support this. If colors don't work, the UI will still be functional, just monochrome.

### systemd service not showing UI
The systemd service logs to journalctl, which shows log text but not the visual UI. To see the visual UI, run the program directly or in screen/tmux.

## Next Steps

1. Build: `./build-pi.sh`
2. Deploy: Transfer binary to Raspberry Pi
3. Install: Run `sudo ./install.sh` on Pi
4. Monitor: Use `journalctl -u insectius-monitor -f`

That's it! Your sensor monitor is now running with a lightweight terminal UI.
