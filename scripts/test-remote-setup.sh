#!/bin/bash
# Quick test script to verify your Mac server is ready for remote clients

set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}╔══════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   WireScope Remote Setup Test               ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════╝${NC}"
echo ""

# Get Mac IP
MAC_IP=$(ifconfig | grep "inet " | grep -v 127.0.0.1 | awk '{print $2}' | head -1)
echo -e "${BLUE}📍 Your Mac IP: ${GREEN}${MAC_IP}${NC}"
echo ""

# Test 1: Docker services
echo -e "${BLUE}[1/5] Checking Docker services...${NC}"
if docker-compose ps | grep -q "Up"; then
    echo -e "${GREEN}✓ Docker services are running${NC}"
else
    echo -e "${RED}✗ Docker services not running. Run: docker-compose up -d${NC}"
    exit 1
fi

# Test 2: Postgres
echo -e "${BLUE}[2/5] Checking PostgreSQL...${NC}"
if docker exec wirescope-postgres-1 pg_isready -U telemetry -d telemetry > /dev/null 2>&1; then
    echo -e "${GREEN}✓ PostgreSQL is healthy${NC}"
else
    echo -e "${RED}✗ PostgreSQL is not ready${NC}"
    exit 1
fi

# Test 3: NATS
echo -e "${BLUE}[3/5] Checking NATS...${NC}"
if curl -s http://localhost:8222/varz > /dev/null 2>&1; then
    echo -e "${GREEN}✓ NATS is running${NC}"
else
    echo -e "${YELLOW}⚠  NATS monitoring endpoint not responding${NC}"
fi

# Test 4: Ingest API
echo -e "${BLUE}[4/5] Checking Ingest API...${NC}"
if curl -s http://localhost:8081/health > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Ingest API is running on port 8081${NC}"
    echo -e "${GREEN}  Response: $(curl -s http://localhost:8081/health)${NC}"
else
    echo -e "${RED}✗ Ingest API is not running${NC}"
    echo -e "${YELLOW}  Start it with: ./bin/ingest --port 8081 --nats-url nats://localhost:4222 --api-token demo-token${NC}"
    exit 1
fi

# Test 5: External connectivity
echo -e "${BLUE}[5/5] Testing external connectivity...${NC}"
echo -e "${YELLOW}  Attempting to reach ${MAC_IP}:8081 from localhost...${NC}"
if curl -s http://${MAC_IP}:8081/health > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Ingest API is accessible via local IP${NC}"
else
    echo -e "${YELLOW}⚠  Cannot reach via local IP (might be firewall)${NC}"
fi

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║          Server Ready! 🎉                    ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}📋 Remote Desktop Setup Instructions:${NC}"
echo ""
echo -e "1. On your remote desktop, download the probe:"
echo -e "   ${YELLOW}wget https://github.com/rahulgh33/WireScope/releases/latest/download/probe-linux-amd64 -O probe${NC}"
echo -e "   ${YELLOW}chmod +x probe${NC}"
echo ""
echo -e "2. Run the probe pointing to your Mac:"
echo -e "   ${GREEN}./probe \\${NC}"
echo -e "   ${GREEN}  --ingest-url http://${MAC_IP}:8081/events \\${NC}"
echo -e "   ${GREEN}  --api-token demo-token \\${NC}"
echo -e "   ${GREEN}  --target https://google.com \\${NC}"
echo -e "   ${GREEN}  --client-id remote-desktop-1 \\${NC}"
echo -e "   ${GREEN}  --interval 60${NC}"
echo ""
echo -e "3. View results:"
echo -e "   ${BLUE}http://localhost:3000${NC} (Grafana dashboard)"
echo -e "   ${BLUE}http://localhost:9090${NC} (Prometheus)"
echo ""
echo -e "${BLUE}💡 Tip: If remote connection fails, check firewall:${NC}"
echo -e "   ${YELLOW}sudo /usr/libexec/ApplicationFirewall/socketfilterfw --add ./bin/ingest${NC}"
echo -e "   ${YELLOW}sudo /usr/libexec/ApplicationFirewall/socketfilterfw --unblockapp ./bin/ingest${NC}"
echo ""
