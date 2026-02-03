#!/bin/bash
# Launch the Sensor Monitor GUI

# Get the directory where this script is located
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Change to that directory
cd "$DIR"

# Run the Python GUI
python3 monitor_gui.py
