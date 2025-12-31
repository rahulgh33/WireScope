#!/bin/bash
# Quick start script for WireScope probe
# Usage: curl -sSL https://raw.githubusercontent.com/rahulgh33/WireScope/main/quick-probe.sh | bash -s -- SERVER_IP

set -e

SERVER_IP="${1:-192.168.1.39}"
API_TOKEN="${2:-demo-token}"
CLIENT_ID="${3:-$(hostname)-probe}"

echo "🚀 Starting WireScope Probe"
echo "   Server: ${SERVER_IP}"
echo "   Client: ${CLIENT_ID}"
echo ""

# Download if not exists
if [ ! -f "./probe" ]; then
    echo "📥 Downloading probe..."
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        wget -q https://github.com/rahulgh33/WireScope/releases/latest/download/probe-linux-amd64 -O probe
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        curl -sL https://github.com/rahulgh33/WireScope/releases/latest/download/probe-darwin-amd64 -o probe
    fi
    chmod +x probe
    echo "✅ Downloaded"
fi

# Run probe
./probe \
  --ingest-url "http://${SERVER_IP}:8081/events" \
  --api-token "${API_TOKEN}" \
  --target https://google.com \
  --target https://github.com \
  --target https://cloudflare.com \
  --client-id "${CLIENT_ID}" \
  --interval 60s
