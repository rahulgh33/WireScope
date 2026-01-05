#!/bin/bash

# Check status of all services for WireScope

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== WireScope Status ===${NC}\n"

# Check Docker services
echo -e "${BLUE}Docker Services:${NC}"
echo "---"
docker ps --filter "name=wirescope" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || echo -e "${RED}✗ Docker not running or no containers${NC}"

# Check Go services
echo -e "\n${BLUE}Go Services:${NC}"
echo "---"

check_go_service() {
    local service=$1
    local port=$2
    local pid_file="logs/$service.pid"
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if ps -p $pid > /dev/null 2>&1; then
            # Check if service is responding
            if [ -n "$port" ] && curl -s "http://localhost:$port/health" > /dev/null 2>&1; then
                echo -e "${GREEN}✓ $service${NC} (PID: $pid, Port: $port) - ${GREEN}Healthy${NC}"
            else
                echo -e "${YELLOW}✓ $service${NC} (PID: $pid) - ${YELLOW}Running${NC}"
            fi
        else
            echo -e "${RED}✗ $service${NC} - ${RED}Not running${NC} (stale PID: $pid)"
            rm -f "$pid_file"
        fi
    else
        echo -e "${RED}✗ $service${NC} - ${RED}Not running${NC}"
    fi
}

check_go_service "ingest" "3001"
check_go_service "aggregator" ""
check_go_service "diagnoser" ""
check_go_service "ai-agent" ""

# Check Web UI
echo -e "\n${BLUE}Web UI:${NC}"
echo "---"
if lsof -i :5173 > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Web UI${NC} - Running on http://localhost:5173"
elif lsof -i :3000 | grep node > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Web UI${NC} - Running on http://localhost:3000"
else
    echo -e "${YELLOW}✗ Web UI${NC} - Not running (run: cd web && npm run dev)"
fi

# Check service endpoints
echo -e "\n${BLUE}Service Health Checks:${NC}"
echo "---"

check_endpoint() {
    local name=$1
    local url=$2
    
    if curl -s "$url" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ $name${NC} - $url"
    else
        echo -e "${RED}✗ $name${NC} - $url"
    fi
}

check_endpoint "Grafana" "http://localhost:3000"
check_endpoint "Prometheus" "http://localhost:9090"
check_endpoint "Jaeger" "http://localhost:16686"
check_endpoint "NATS Monitoring" "http://localhost:8222"
check_endpoint "Test Target" "http://localhost:8080/health"
check_endpoint "PostgreSQL" "http://localhost:5432" || echo -e "${YELLOW}Note: PostgreSQL check via HTTP not available (use psql)${NC}"

# Resource usage
echo -e "\n${BLUE}Resource Usage:${NC}"
echo "---"
if command -v docker &> /dev/null; then
    docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" \
        --filter "name=wirescope" 2>/dev/null | head -n 10
fi

# Recent logs summary
echo -e "\n${BLUE}Recent Log Summary:${NC}"
echo "---"
if [ -d "logs" ]; then
    for logfile in logs/*.log; do
        if [ -f "$logfile" ]; then
            service=$(basename "$logfile" .log)
            lines=$(wc -l < "$logfile" 2>/dev/null || echo 0)
            echo -e "$service: $lines lines (tail -f $logfile to view)"
        fi
    done
else
    echo -e "${YELLOW}No logs directory found${NC}"
fi

echo -e "\n${BLUE}Quick Commands:${NC}"
echo "  • Start all:    ./scripts/start-services.sh"
echo "  • Stop all:     ./scripts/stop-services.sh"
echo "  • View logs:    tail -f logs/*.log"
echo "  • Run probe:    ./bin/probe --target http://localhost:8080 --interval 10s"
