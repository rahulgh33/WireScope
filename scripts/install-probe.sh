#!/bin/bash
# WireScope Probe Installer
# Usage: curl -sSL https://raw.githubusercontent.com/rahulgh33/WireScope/main/scripts/install-probe.sh | bash -s -- SERVER_IP

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

SERVER_IP="$1"
if [ -z "$SERVER_IP" ]; then
    echo -e "${YELLOW}Usage: $0 SERVER_IP${NC}"
    echo "Example: curl -sSL https://raw.githubusercontent.com/rahulgh33/WireScope/main/scripts/install-probe.sh | bash -s -- 192.168.1.100"
    exit 1
fi

# Detect OS and architecture
OS="unknown"
ARCH=$(uname -m)

if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OS="linux"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    OS="darwin"
else
    echo -e "${YELLOW}Unsupported OS: $OSTYPE${NC}"
    echo "Please download manually from: https://github.com/rahulgh33/WireScope/releases"
    exit 1
fi

# Map architecture
case "$ARCH" in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${YELLOW}Unsupported architecture: $ARCH${NC}"
        ARCH="amd64"  # Default to amd64
        ;;
esac

BINARY_NAME="probe-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/rahulgh33/WireScope/releases/latest/download/${BINARY_NAME}"

echo -e "${BLUE}╔══════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║      WireScope Probe Installer              ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}📦 Detected: ${OS} ${ARCH}${NC}"
echo -e "${BLUE}🌐 Server: ${SERVER_IP}${NC}"
echo -e "${BLUE}⬇️  Downloading from GitHub releases...${NC}"
echo ""

# Download probe binary
if command -v wget &> /dev/null; then
    wget -q --show-progress -O probe "$DOWNLOAD_URL" || {
        echo -e "${YELLOW}⚠️  Download failed. Trying curl...${NC}"
        curl -L -o probe "$DOWNLOAD_URL"
    }
elif command -v curl &> /dev/null; then
    curl -L -o probe "$DOWNLOAD_URL"
else
    echo -e "${YELLOW}❌ Neither wget nor curl found. Please install one of them.${NC}"
    exit 1
fi

chmod +x probe

# Test connection to server
echo ""
echo -e "${BLUE}🔌 Testing connection to server...${NC}"
if curl -s -f "http://${SERVER_IP}:8081/health" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Server is reachable!${NC}"
else
    echo -e "${YELLOW}⚠️  Warning: Cannot reach server at http://${SERVER_IP}:8081${NC}"
    echo -e "${YELLOW}   Make sure:${NC}"
    echo -e "${YELLOW}   1. Server is running (./quick-start-server.sh)${NC}"
    echo -e "${YELLOW}   2. Firewall allows port 8081${NC}"
    echo -e "${YELLOW}   3. You're using the correct IP address${NC}"
    echo ""
fi

# Get hostname for client ID
CLIENT_ID=$(hostname | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9-]/-/g')-probe

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║          Installation Complete! 🎉           ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}🚀 Start monitoring:${NC}"
echo ""
echo -e "  ./probe \\"
echo -e "    --ingest-url http://${SERVER_IP}:8081/events \\"
echo -e "    --api-token demo-token \\"
echo -e "    --target https://google.com \\"
echo -e "    --client-id ${CLIENT_ID}"
echo ""
echo -e "${BLUE}Or run in background:${NC}"
echo ""
echo -e "  nohup ./probe \\"
echo -e "    --ingest-url http://${SERVER_IP}:8081/events \\"
echo -e "    --api-token demo-token \\"
echo -e "    --target https://google.com \\"
echo -e "    --client-id ${CLIENT_ID} \\"
echo -e "    > probe.log 2>&1 &"
echo ""
echo -e "${BLUE}📊 View data:${NC} Open http://${SERVER_IP}:3000 in your browser"
echo ""
