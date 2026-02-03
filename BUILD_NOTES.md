# Build Notes - Terminal UI

## Summary of Implementation

Successfully replaced Fyne GUI with a terminal-based UI using pure ANSI escape codes. The program now:
- ✅ Cross-compiles to ARM64 without CGO for the target platform
- ✅ Produces a 7.5MB statically-linked binary
- ✅ Works over SSH without display requirements
- ✅ Compatible with screen/tmux

## Important Note about CGO

### On macOS (Development)
The bluetooth library uses CoreBluetooth which requires CGO on macOS. This is **expected and correct** - you cannot compile with `CGO_ENABLED=0` on macOS itself.

### On Linux (Deployment Target - Raspberry Pi)
The bluetooth library uses native Linux Bluetooth APIs via D-Bus, which **does NOT require CGO**. Cross-compiling from macOS to Linux ARM64 works perfectly with `CGO_ENABLED=0`.

## Build Commands

### Cross-compile for Raspberry Pi (from macOS)
```bash
# This works because target is Linux (no CGO needed)
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o insectius-monitor

# Or use the provided script
./build-pi.sh
```

### Build on Raspberry Pi itself
```bash
# On the Pi, you can also build with CGO disabled
CGO_ENABLED=0 go build -o insectius-monitor
```

## Verification

Cross-compilation test:
```bash
$ CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o test
$ file test
test: ELF 64-bit LSB executable, ARM aarch64, statically linked

$ ls -lh test
-rwxr-xr-x  1 user  staff   7.5M  test
```

This confirms:
1. ✅ Static linking (no dynamic dependencies)
2. ✅ ARM64 architecture (Raspberry Pi compatible)
3. ✅ Reasonable size (7.5MB vs 23MB with Fyne)

## Dependencies

After removing Fyne, only one main dependency remains:
- `tinygo.org/x/bluetooth` - Bluetooth Low Energy library
  - On Linux: Uses D-Bus (no CGO)
  - On macOS: Uses CoreBluetooth (requires CGO)

The other dependencies are transitive and minimal (dbus, logrus, etc).

## Testing on Raspberry Pi

After transferring the binary to Raspberry Pi:
```bash
# Transfer
scp insectius-monitor pi@raspberry:/home/pi/

# On the Pi
sudo ./install.sh

# Run interactively to see terminal UI
./insectius-monitor

# Or run via systemd and view with journalctl
sudo systemctl start insectius-monitor
journalctl -u insectius-monitor -f
```

## Terminal UI Features

The terminal UI renders using ANSI escape codes:
- Box-drawing characters (╔══╗ ║ ╚══╝)
- Colored backgrounds (\033[42m for green, \033[41m for red)
- Real-time updates with clear screen (\033[2J\033[H)
- Thread-safe rendering with mutexes

No external terminal UI library needed - just stdlib!
