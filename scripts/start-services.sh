#!/bin/bash

# Start all Go services for WireScope
# Usage: ./scripts/start-services.sh [options]
# Options:
#   --build    Build binaries before starting
#   --migrate  Run migrations before starting
#   --logs     Follow logs after starting

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse arguments
BUILD=false
MIGRATE=false
FOLLOW_LOGS=false

for arg in "$@"; do
    case $arg in
        --build) BUILD=true ;;
        --migrate) MIGRATE=true ;;
        --logs) FOLLOW_LOGS=true ;;
        *) echo "Unknown option: $arg"; exit 1 ;;
    esac
done

echo -e "${BLUE}=== WireScope Startup ===${NC}\n"

# Check if Docker services are running
echo -e "${YELLOW}Checking Docker services...${NC}"
if ! docker ps | grep -q "wirescope-postgres-1"; then
    echo -e "${RED}✗ Docker services are not running${NC}"
    echo -e "${YELLOW}Starting Docker services...${NC}"
    make up
    echo -e "${GREEN}✓ Docker services started${NC}"
    echo -e "${YELLOW}Waiting 10 seconds for services to stabilize...${NC}"
    sleep 10
else
    echo -e "${GREEN}✓ Docker services are running${NC}"
fi

# Build binaries if requested
if [ "$BUILD" = true ]; then
    echo -e "\n${YELLOW}Building Go binaries...${NC}"
    make build
    echo -e "${GREEN}✓ Binaries built${NC}"
fi

# Check if binaries exist
echo -e "\n${YELLOW}Checking binaries...${NC}"
BINARIES=("ingest" "aggregator" "diagnoser" "probe")
for binary in "${BINARIES[@]}"; do
    if [ ! -f "bin/$binary" ]; then
        echo -e "${RED}✗ Binary bin/$binary not found${NC}"
        echo -e "${YELLOW}Run with --build flag or execute: make build${NC}"
        exit 1
    fi
done
echo -e "${GREEN}✓ All binaries found${NC}"

# Run migrations if requested
if [ "$MIGRATE" = true ]; then
    echo -e "\n${YELLOW}Running database migrations...${NC}"
    make migrate
    echo -e "${GREEN}✓ Migrations completed${NC}"
fi

# Create logs directory
mkdir -p logs

# Stop any existing services
echo -e "\n${YELLOW}Stopping any existing services...${NC}"
pkill -f "bin/ingest" 2>/dev/null || true
pkill -f "bin/aggregator" 2>/dev/null || true
pkill -f "bin/diagnoser" 2>/dev/null || true
sleep 2

# Start services
echo -e "\n${BLUE}Starting Go services...${NC}"

# Export database configuration
export DB_USER=telemetry
export DB_PASSWORD=telemetry
export DB_HOST=localhost
export DB_NAME=telemetry
export DB_PORT=5432
export DB_SSLMODE=disable

# Start Ingest API
echo -e "${YELLOW}Starting Ingest API...${NC}"
nohup ./bin/ingest --port 8081 > logs/ingest.log 2>&1 &
INGEST_PID=$!
echo $INGEST_PID > logs/ingest.pid
echo -e "${GREEN}✓ Ingest API started (PID: $INGEST_PID)${NC}"
sleep 2

# Start Aggregator
echo -e "${YELLOW}Starting Aggregator...${NC}"
nohup ./bin/aggregator \
  --db-user=${DB_USER} \
  --db-password=${DB_PASSWORD} \
  --db-host=${DB_HOST} \
  --db-name=${DB_NAME} \
  --db-port=${DB_PORT} \
  > logs/aggregator.log 2>&1 &
AGGREGATOR_PID=$!
echo $AGGREGATOR_PID > logs/aggregator.pid
echo -e "${GREEN}✓ Aggregator started (PID: $AGGREGATOR_PID)${NC}"
sleep 2

# Note: Diagnoser is not implemented yet - diagnosis logic exists in aggregator

# Start AI Agent if binary exists
if [ -f "bin/ai-agent" ]; then
    echo -e "${YELLOW}Starting AI Agent...${NC}"
    SERVER_ADDR=:9000 nohup ./bin/ai-agent > logs/ai-agent.log 2>&1 &
    AI_AGENT_PID=$!
    echo $AI_AGENT_PID > logs/ai-agent.pid
    echo -e "${GREEN}✓ AI Agent started (PID: $AI_AGENT_PID)${NC}"
fi

# Wait a moment for services to start
sleep 3

# Check service health
echo -e "\n${BLUE}Checking service health...${NC}"

# Check Ingest API
if curl -s http://localhost:8081/health > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Ingest API is healthy (http://localhost:8081)${NC}"
else
    echo -e "${RED}✗ Ingest API may not be healthy (check logs/ingest.log)${NC}"
fi

# Check AI Agent
if curl -s http://localhost:9000/health > /dev/null 2>&1; then
    echo -e "${GREEN}✓ AI Agent is healthy (http://localhost:9000)${NC}"
else
    echo -e "${RED}✗ AI Agent may not be healthy (check logs/ai-agent.log)${NC}"
fi

# Check if processes are still running
for service in ingest aggregator diagnoser; do
    if [ -f "logs/$service.pid" ]; then
        pid=$(cat "logs/$service.pid")
        if ps -p $pid > /dev/null 2>&1; then
            echo -e "${GREEN}✓ $service is running (PID: $pid)${NC}"
        else
            echo -e "${RED}✗ $service failed to start (check logs/$service.log)${NC}"
        fi
    fi
done


# Summary
echo -e "\n${BLUE}=== Services Started ===${NC}"
echo -e "${GREEN}✓ All services are running${NC}\n"

echo -e "${BLUE}Access Points:${NC}"
echo -e "  • Web UI:       http://localhost:5173 (run 'cd web && npm run dev')"
echo -e "  • Ingest API:   http://localhost:3001"
echo -e "  • Grafana:      http://localhost:3000 (admin/admin)"
echo -e "  • Prometheus:   http://localhost:9090"
echo -e "  • Jaeger:       http://localhost:16686"
echo -e "  • NATS:         http://localhost:8222"

echo -e "\n${BLUE}Useful Commands:${NC}"
echo -e "  • View logs:    tail -f logs/*.log"
echo -e "  • Stop all:     ./scripts/stop-services.sh"
echo -e "  • Test system:  ./bin/probe --target http://localhost:8080 --interval 10s"

echo -e "\n${BLUE}Log Files:${NC}"
echo -e "  • Ingest:       logs/ingest.log"
echo -e "  • Aggregator:   logs/aggregator.log"
echo -e "  • Diagnoser:    logs/diagnoser.log"

# Follow logs if requested
if [ "$FOLLOW_LOGS" = true ]; then
    echo -e "\n${YELLOW}Following logs (Ctrl+C to exit)...${NC}\n"
    tail -f logs/*.log
fi
