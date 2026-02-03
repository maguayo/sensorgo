# Sensor Status Monitor GUI

A full-screen Python GUI that monitors sensor readings and displays a visual status indicator.

## Features

- **Full-screen status display**: Shows a large green ✓ (check) or red ✗ (cross)
- **Green check**: All sensors have reported within the last 6 minutes
- **Red cross**: One or more sensors haven't reported in over 6 minutes, or there's an API error
- **Sensor information**: Displays sensor count and IDs in the bottom right corner
- **Last check timestamp**: Shows when the last API check occurred (bottom left)
- **Auto-refresh**: Checks sensor status every 30 seconds

## Requirements

- Python 3.7+
- `tkinter` (usually pre-installed on Raspberry Pi OS)
- `requests` library

## Installation on Raspberry Pi

1. **Install Python dependencies:**
   ```bash
   pip3 install -r requirements-gui.txt
   ```

2. **Ensure you have the required files:**
   - `~/.insectius-monitor` - API key file (created by main program)
   - `authorized_sensors.json` - Sensor configuration (created by running main program first)

3. **Make scripts executable:**
   ```bash
   chmod +x monitor_gui.py
   chmod +x launch_monitor.sh
   ```

## Running the Monitor

### Option 1: Command Line
```bash
python3 monitor_gui.py
```

Or use the launch script:
```bash
./launch_monitor.sh
```

### Option 2: Desktop Icon (Raspberry Pi)

1. **Copy the desktop file to your desktop or applications folder:**
   ```bash
   # For desktop shortcut:
   cp sensor-monitor.desktop ~/Desktop/
   chmod +x ~/Desktop/sensor-monitor.desktop

   # For application menu:
   cp sensor-monitor.desktop ~/.local/share/applications/
   ```

2. **Edit the Exec path in sensor-monitor.desktop if needed:**
   Update the path to match where you installed the project:
   ```
   Exec=/path/to/sensorsgo/launch_monitor.sh
   ```

3. **Double-click the icon to launch**

### Option 3: Auto-start on Boot

To start the monitor automatically when the Raspberry Pi boots:

1. **Create autostart directory if it doesn't exist:**
   ```bash
   mkdir -p ~/.config/autostart
   ```

2. **Copy the desktop file:**
   ```bash
   cp sensor-monitor.desktop ~/.config/autostart/
   ```

3. **Edit the path in the autostart file if needed**

## Keyboard Controls

- `ESC` - Toggle full-screen mode
- `q` - Quit the application

## Configuration

Edit these constants in `monitor_gui.py` if needed:

- `API_BASE_URL` - API endpoint (default: `https://go.larvai.com/api/v1`)
- `CHECK_INTERVAL` - How often to check sensors in milliseconds (default: 30000 = 30 seconds)
- `TIMEOUT_MINUTES` - Alert threshold in minutes (default: 6 minutes)

## Troubleshooting

### "No API key found" error
- Ensure `~/.insectius-monitor` file exists and contains your API key
- This file is created when you run the main sensor monitoring program

### "No sensors configured" error
- Run the main Go program first to register sensors: `go run main.go`
- This creates the `authorized_sensors.json` file

### GUI doesn't display correctly
- Ensure tkinter is installed: `sudo apt-get install python3-tk`
- Try running in windowed mode first by commenting out the fullscreen line

### Red cross always showing
- Check that sensors are actually sending data
- Verify the API endpoint is correct
- Check your network connection
- Look at the bottom left corner for error messages

## API Endpoint

The monitor checks this endpoint for each sensor:
```
GET /api/v1/sensor-readings/:sensor_id/last
```

It expects a JSON response with a `created_at` or `timestamp` field in ISO format.

## Development

To test the monitor without full screen:
```python
# Comment out this line in monitor_gui.py:
# self.root.attributes('-fullscreen', True)
```

To change check frequency (e.g., every 10 seconds):
```python
CHECK_INTERVAL = 10000  # 10 seconds in milliseconds
```
