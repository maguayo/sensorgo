#!/usr/bin/env python3
"""
Sensor Status Monitor
Displays a full-screen status indicator showing whether sensors are active
"""

import tkinter as tk
from tkinter import font
import json
import os
import sys
import requests
from datetime import datetime, timedelta, timezone
from pathlib import Path

# Configuration
API_BASE_URL = "https://go.larvai.com/api/v1"
CONFIG_FILE = "authorized_sensors.json"
API_KEY_FILE = os.path.expanduser("~/.insectius-monitor")
CHECK_INTERVAL = 30000  # Check every 30 seconds (in milliseconds)
TIMEOUT_MINUTES = 6  # Alert if no reading in last 6 minutes

class SensorMonitorGUI:
    def __init__(self, root):
        self.root = root
        self.root.title("Sensor Status Monitor")

        # Make window full screen
        self.root.attributes('-fullscreen', True)
        self.root.configure(bg='black')

        # Bind ESC key to exit fullscreen
        self.root.bind('<Escape>', self.toggle_fullscreen)
        self.root.bind('q', lambda e: self.root.quit())

        # Load API key
        self.api_key = self.load_api_key()
        if not self.api_key:
            self.show_error("No API key found. Create ~/.insectius-monitor file")
            return

        # Load sensor configuration
        self.sensors = self.load_sensors()
        if not self.sensors:
            self.show_error("No sensors configured. Run main program first.")
            return

        # Create main status display
        self.status_label = tk.Label(
            root,
            text="?",
            font=font.Font(size=400, weight='bold'),
            bg='black',
            fg='gray'
        )
        self.status_label.place(relx=0.5, rely=0.5, anchor='center')

        # Create sensor info display (bottom right)
        self.info_frame = tk.Frame(root, bg='black')
        self.info_frame.place(relx=0.98, rely=0.98, anchor='se')

        self.info_label = tk.Label(
            self.info_frame,
            text="",
            font=font.Font(size=12),
            bg='black',
            fg='white',
            justify='right'
        )
        self.info_label.pack()

        # Create timestamp display (bottom left)
        self.timestamp_label = tk.Label(
            root,
            text="",
            font=font.Font(size=10),
            bg='black',
            fg='gray',
            justify='left'
        )
        self.timestamp_label.place(relx=0.02, rely=0.98, anchor='sw')

        # Update sensor info display
        self.update_sensor_info()

        # Start monitoring
        self.check_sensor_status()

    def load_api_key(self):
        """Load API key from file"""
        try:
            with open(API_KEY_FILE, 'r') as f:
                return f.read().strip()
        except FileNotFoundError:
            return None

    def load_sensors(self):
        """Load sensor configuration from JSON file"""
        try:
            with open(CONFIG_FILE, 'r') as f:
                config = json.load(f)
                return config.get('authorized_sensors', [])
        except FileNotFoundError:
            return []

    def update_sensor_info(self):
        """Update the sensor information display in bottom right"""
        sensor_count = len(self.sensors)
        sensor_list = "\n".join([
            f"{s['name']} ({s['mac'][:8]}...)"
            for s in self.sensors
        ])

        info_text = f"Sensors: {sensor_count}\n{sensor_list}"
        self.info_label.config(text=info_text)

    def check_sensor_status(self):
        """Check all sensors and update display"""
        all_ok = True
        oldest_reading = None
        error_message = None

        try:
            for sensor in self.sensors:
                sensor_id = sensor['mac']

                # Call the API endpoint
                url = f"{API_BASE_URL}/sensor-readings/{sensor_id}/last"
                headers = {"Authorization": f"Bearer {self.api_key}"}

                try:
                    response = requests.get(url, headers=headers, timeout=10)

                    if response.status_code == 200:
                        data = response.json()

                        # Parse the timestamp (format: 2026-02-03T17:07:21.963143179Z)
                        reading_time_str = data.get('timestamp')
                        if reading_time_str:
                            # Parse ISO format timestamp with Z timezone
                            try:
                                reading_time = datetime.fromisoformat(reading_time_str.replace('Z', '+00:00'))
                            except Exception as e:
                                all_ok = False
                                error_message = f"Timestamp parse error: {str(e)[:30]}"
                                continue

                            # Calculate time difference (use UTC for comparison)
                            now = datetime.now(timezone.utc)
                            time_diff = now - reading_time

                            # Track oldest reading
                            if oldest_reading is None or reading_time < oldest_reading:
                                oldest_reading = reading_time

                            # Check if reading is too old (more than 6 minutes)
                            if time_diff > timedelta(minutes=TIMEOUT_MINUTES):
                                all_ok = False
                                error_message = f"{sensor['name']}: Last reading {int(time_diff.total_seconds() / 60)} min ago"
                        else:
                            all_ok = False
                            error_message = "No timestamp in response"
                    elif response.status_code == 404:
                        all_ok = False
                        error_message = f"No readings found for {sensor['name']}"
                    else:
                        all_ok = False
                        error_message = f"API error {response.status_code}"

                except requests.RequestException as e:
                    all_ok = False
                    error_message = f"Connection error: {str(e)[:50]}"

        except Exception as e:
            all_ok = False
            error_message = f"Error: {str(e)[:50]}"

        # Update display
        if all_ok:
            self.status_label.config(text="✓", fg='#00ff00')  # Green check
        else:
            self.status_label.config(text="✗", fg='#ff0000')  # Red cross

        # Update timestamp display
        timestamp_text = f"Last check: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}"
        if oldest_reading:
            timestamp_text += f"\nOldest reading: {oldest_reading.strftime('%Y-%m-%d %H:%M:%S')}"
        if error_message:
            timestamp_text += f"\n{error_message}"
        self.timestamp_label.config(text=timestamp_text)

        # Schedule next check
        self.root.after(CHECK_INTERVAL, self.check_sensor_status)

    def show_error(self, message):
        """Show error message"""
        error_label = tk.Label(
            self.root,
            text=f"ERROR\n{message}",
            font=font.Font(size=30, weight='bold'),
            bg='black',
            fg='red'
        )
        error_label.place(relx=0.5, rely=0.5, anchor='center')

    def toggle_fullscreen(self, event=None):
        """Toggle fullscreen mode"""
        current = self.root.attributes('-fullscreen')
        self.root.attributes('-fullscreen', not current)

def main():
    root = tk.Tk()
    app = SensorMonitorGUI(root)
    root.mainloop()

if __name__ == "__main__":
    main()
