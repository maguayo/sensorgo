#!/bin/bash
# Installation script for Sensor Monitor GUI on Raspberry Pi

set -e

echo "🔧 Installing Sensor Monitor GUI..."

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Check if Python 3 is installed
if ! command -v python3 &> /dev/null; then
    echo "❌ Python 3 is not installed. Please install Python 3 first."
    exit 1
fi

echo -e "${GREEN}✓${NC} Python 3 found"

# Check if tkinter is available
if ! python3 -c "import tkinter" &> /dev/null; then
    echo -e "${YELLOW}⚠${NC} tkinter not found. Installing..."
    sudo apt-get update
    sudo apt-get install -y python3-tk
    echo -e "${GREEN}✓${NC} tkinter installed"
else
    echo -e "${GREEN}✓${NC} tkinter found"
fi

# Install Python dependencies
echo "📦 Installing Python dependencies..."
pip3 install -r "$SCRIPT_DIR/requirements-gui.txt"
echo -e "${GREEN}✓${NC} Dependencies installed"

# Make scripts executable
chmod +x "$SCRIPT_DIR/monitor_gui.py"
chmod +x "$SCRIPT_DIR/launch_monitor.sh"
echo -e "${GREEN}✓${NC} Scripts made executable"

# Check for API key
if [ ! -f ~/.insectius-monitor ]; then
    echo -e "${YELLOW}⚠${NC} API key file not found at ~/.insectius-monitor"
    echo "   You'll need to create this file before running the monitor."
fi

# Check for sensor configuration
if [ ! -f "$SCRIPT_DIR/authorized_sensors.json" ]; then
    echo -e "${YELLOW}⚠${NC} Sensor configuration not found"
    echo "   Run the main program first to register sensors."
fi

# Offer to create desktop shortcut
echo ""
read -p "Do you want to create a desktop shortcut? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    mkdir -p ~/Desktop

    # Update the Exec path in the desktop file
    sed "s|Exec=.*|Exec=$SCRIPT_DIR/launch_monitor.sh|g" "$SCRIPT_DIR/sensor-monitor.desktop" > ~/Desktop/sensor-monitor.desktop
    chmod +x ~/Desktop/sensor-monitor.desktop

    echo -e "${GREEN}✓${NC} Desktop shortcut created"
fi

# Offer to add to autostart
echo ""
read -p "Do you want the monitor to start automatically on boot? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    mkdir -p ~/.config/autostart

    # Update the Exec path in the desktop file
    sed "s|Exec=.*|Exec=$SCRIPT_DIR/launch_monitor.sh|g" "$SCRIPT_DIR/sensor-monitor.desktop" > ~/.config/autostart/sensor-monitor.desktop

    echo -e "${GREEN}✓${NC} Autostart configured"
fi

echo ""
echo -e "${GREEN}✅ Installation complete!${NC}"
echo ""
echo "To run the monitor:"
echo "  • Double-click the desktop icon (if created)"
echo "  • Or run: $SCRIPT_DIR/launch_monitor.sh"
echo "  • Or run: python3 $SCRIPT_DIR/monitor_gui.py"
echo ""
echo "Keyboard shortcuts:"
echo "  • ESC - Toggle fullscreen"
echo "  • Q - Quit"
