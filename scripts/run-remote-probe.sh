#!/bin/bash
# Run this on your remote desktop to connect to your Mac server

# Your Mac Server IP
SERVER_IP="192.168.1.39"

# Download probe if not already downloaded
if [ ! -f "probe" ]; then
    echo "Downloading probe..."
    
    # Detect OS
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        wget https://github.com/rahulgh33/WireScope/releases/latest/download/probe-linux-amd64 -O probe
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        curl -L https://github.com/rahulgh33/WireScope/releases/latest/download/probe-darwin-amd64 -o probe
    else
        echo "Please download probe manually from https://github.com/rahulgh33/WireScope/releases"
        exit 1
    fi
    
    chmod +x probe
    echo "✓ Probe downloaded"
fi

# Run the probe
echo "Starting probe..."
echo "Connecting to server at ${SERVER_IP}:8081"
echo "Press Ctrl+C to stop"
echo ""

./probe \
  --ingest-url http://${SERVER_IP}:8081/events \
  --api-token demo-token \
  --target https://google.com \
  --target https://github.com \
  --target https://cloudflare.com \
  --client-id remote-desktop-1 \
  --interval 60
