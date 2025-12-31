#!/bin/bash
# Universal Cloud Deployment Script for WireScope
# Works on any Ubuntu/Debian VM with a public IP
# Usage: ./deploy-cloud.sh

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}╔══════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   WireScope Cloud Deployment                ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════╝${NC}"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}Please run as root (use sudo)${NC}"
    exit 1
fi

# Get public IP
PUBLIC_IP=$(curl -s ifconfig.me || curl -s icanhazip.com || echo "unknown")
echo -e "${BLUE}📍 Detected Public IP: ${GREEN}${PUBLIC_IP}${NC}"
echo ""

# Install dependencies
echo -e "${YELLOW}[1/6] Installing dependencies...${NC}"
apt-get update -qq
apt-get install -y -qq docker.io docker-compose git curl make golang-go > /dev/null 2>&1
systemctl enable docker
systemctl start docker
echo -e "${GREEN}✓ Dependencies installed${NC}"

# Clone or update repo
echo -e "${YELLOW}[2/6] Setting up WireScope...${NC}"
if [ -d "/opt/wirescope" ]; then
    cd /opt/wirescope
    git pull
else
    git clone https://github.com/rahulgh33/WireScope.git /opt/wirescope
    cd /opt/wirescope
fi
echo -e "${GREEN}✓ Repository ready${NC}"

# Build binaries
echo -e "${YELLOW}[3/6] Building services...${NC}"
make build
echo -e "${GREEN}✓ Services built${NC}"

# Configure environment
echo -e "${YELLOW}[4/6] Configuring environment...${NC}"
cat > .env << EOF
# Database
DB_USER=telemetry
DB_PASSWORD=telemetry
DB_HOST=postgres
DB_NAME=telemetry
DB_PORT=5432

# NATS
NATS_URL=nats://nats:4222

# API
API_TOKENS=demo-token,prod-token-$(openssl rand -hex 16)
INGEST_PORT=8081

# Monitoring
TRACING_ENABLED=true
OTLP_ENDPOINT=jaeger:4318

# External
EXTERNAL_HOST=${PUBLIC_IP}
EOF
echo -e "${GREEN}✓ Environment configured${NC}"

# Start Docker services
echo -e "${YELLOW}[5/6] Starting Docker services...${NC}"
docker-compose up -d
sleep 10
echo -e "${GREEN}✓ Docker services started${NC}"

# Start Go services
echo -e "${YELLOW}[6/6] Starting WireScope services...${NC}"
./scripts/start-services.sh
sleep 5
echo -e "${GREEN}✓ WireScope services started${NC}"

# Setup firewall
echo -e "${YELLOW}Configuring firewall...${NC}"
ufw --force enable
ufw allow 22/tcp      # SSH
ufw allow 8081/tcp    # Ingest API
ufw allow 3000/tcp    # Grafana
ufw allow 9090/tcp    # Prometheus
echo -e "${GREEN}✓ Firewall configured${NC}"

# Health check
echo ""
echo -e "${BLUE}Running health checks...${NC}"
sleep 5

if curl -s http://localhost:8081/health > /dev/null; then
    echo -e "${GREEN}✓ Ingest API is healthy${NC}"
else
    echo -e "${RED}✗ Ingest API is not responding${NC}"
fi

if docker ps | grep -q postgres; then
    echo -e "${GREEN}✓ PostgreSQL is running${NC}"
else
    echo -e "${RED}✗ PostgreSQL is not running${NC}"
fi

if docker ps | grep -q nats; then
    echo -e "${GREEN}✓ NATS is running${NC}"
else
    echo -e "${RED}✗ NATS is not running${NC}"
fi

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║         Deployment Complete! 🎉              ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}📊 Access URLs:${NC}"
echo -e "  • Grafana:    http://${PUBLIC_IP}:3000 (admin/admin)"
echo -e "  • Prometheus: http://${PUBLIC_IP}:9090"
echo -e "  • Ingest API: http://${PUBLIC_IP}:8081"
echo ""
echo -e "${BLUE}🔌 Connect Probes:${NC}"
echo -e "  Run this on any machine to start monitoring:"
echo ""
echo -e "  ${YELLOW}curl -sSL https://raw.githubusercontent.com/rahulgh33/WireScope/main/quick-probe.sh | bash -s -- ${PUBLIC_IP}${NC}"
echo ""
echo -e "${BLUE}💡 Tips:${NC}"
echo -e "  • View logs:    tail -f /opt/wirescope/logs/*.log"
echo -e "  • Stop all:     cd /opt/wirescope && ./scripts/stop-services.sh && docker-compose down"
echo -e "  • Restart all:  cd /opt/wirescope && ./scripts/start-services.sh"
echo ""
