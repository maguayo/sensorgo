#!/usr/bin/env python3
"""
RuuviTag BLE Scanner for Raspberry Pi
Uses bleak library which works reliably with Raspberry Pi's Bluetooth

This script scans for RuuviTags and outputs JSON to stdout or a file.
The Go program can read this data.

Usage:
  python3 ruuvi_scanner.py                    # Output to stdout
  python3 ruuvi_scanner.py --output data.json # Output to file
  python3 ruuvi_scanner.py --daemon           # Run continuously, output to /tmp/ruuvi_data.json
"""

import asyncio
import json
import struct
import sys
import os
import argparse
from datetime import datetime

try:
    from bleak import BleakScanner
except ImportError:
    print("Error: bleak library not installed. Run: pip3 install bleak", file=sys.stderr)
    sys.exit(1)

# Ruuvi manufacturer ID
RUUVI_COMPANY_ID = 0x0499

def parse_ruuvi_rawv2(data):
    """Parse RuuviTag RAWv2 format data"""
    if len(data) < 24 or data[0] != 0x05:
        return None

    try:
        # Temperature (bytes 1-2): signed int16, scale 0.005
        temp_raw = struct.unpack('>h', data[1:3])[0]
        temperature = temp_raw * 0.005

        # Humidity (bytes 3-4): uint16, scale 0.0025
        hum_raw = struct.unpack('>H', data[3:5])[0]
        humidity = hum_raw * 0.0025

        # Pressure (bytes 5-6): uint16, offset 50000 Pa
        press_raw = struct.unpack('>H', data[5:7])[0]
        pressure = (press_raw + 50000) / 100.0

        # Battery (bytes 11-12): first 11 bits in mV
        batt_raw = struct.unpack('>H', data[11:13])[0]
        battery = (batt_raw >> 5) + 1600

        # MAC address (bytes 18-23)
        mac = ':'.join(f'{b:02X}' for b in data[18:24])

        return {
            'temperature': round(temperature, 2),
            'humidity': round(humidity, 2),
            'pressure': round(pressure, 2),
            'battery': battery,
            'mac_from_data': mac
        }
    except Exception as e:
        return None

def detection_callback(device, advertisement_data):
    """Callback for when a BLE device is detected"""
    # Check if this is a RuuviTag
    if advertisement_data.manufacturer_data:
        for company_id, data in advertisement_data.manufacturer_data.items():
            if company_id == RUUVI_COMPANY_ID:
                parsed = parse_ruuvi_rawv2(bytes(data))
                if parsed:
                    result = {
                        'mac': device.address,
                        'name': device.name or 'Unknown',
                        'rssi': advertisement_data.rssi,
                        'timestamp': datetime.now().isoformat(),
                        **parsed
                    }
                    return result
    return None

async def scan_once(duration=10):
    """Scan for RuuviTags for a specified duration"""
    found_devices = {}

    def callback(device, advertisement_data):
        result = detection_callback(device, advertisement_data)
        if result:
            found_devices[device.address] = result
            print(f"Found: {result['name']} ({result['mac']}) - {result['temperature']}°C", file=sys.stderr)

    scanner = BleakScanner(callback)

    print(f"Scanning for {duration} seconds...", file=sys.stderr)
    await scanner.start()
    await asyncio.sleep(duration)
    await scanner.stop()

    return list(found_devices.values())

async def scan_continuous(output_file, interval=30):
    """Continuously scan and update output file"""
    print(f"Running in daemon mode. Output: {output_file}", file=sys.stderr)
    print(f"Scan interval: {interval} seconds", file=sys.stderr)

    while True:
        try:
            devices = await scan_once(duration=10)

            # Write to file
            data = {
                'timestamp': datetime.now().isoformat(),
                'devices': devices
            }

            with open(output_file, 'w') as f:
                json.dump(data, f, indent=2)

            print(f"[{datetime.now().strftime('%H:%M:%S')}] Found {len(devices)} RuuviTag(s)", file=sys.stderr)

            # Wait before next scan
            await asyncio.sleep(interval)

        except Exception as e:
            print(f"Error during scan: {e}", file=sys.stderr)
            await asyncio.sleep(5)

async def main():
    parser = argparse.ArgumentParser(description='RuuviTag BLE Scanner')
    parser.add_argument('--output', '-o', help='Output file (default: stdout)')
    parser.add_argument('--daemon', '-d', action='store_true', help='Run continuously')
    parser.add_argument('--duration', '-t', type=int, default=10, help='Scan duration in seconds')
    parser.add_argument('--interval', '-i', type=int, default=30, help='Interval between scans in daemon mode')
    args = parser.parse_args()

    if args.daemon:
        output_file = args.output or '/tmp/ruuvi_data.json'
        await scan_continuous(output_file, args.interval)
    else:
        devices = await scan_once(args.duration)
        output = json.dumps(devices, indent=2)

        if args.output:
            with open(args.output, 'w') as f:
                f.write(output)
            print(f"Output written to {args.output}", file=sys.stderr)
        else:
            print(output)

if __name__ == '__main__':
    asyncio.run(main())
