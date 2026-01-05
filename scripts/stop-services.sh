#!/bin/bash

# Stop all Go services for WireScope

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Stopping WireScope Services ===${NC}\n"

# Function to stop a service by PID file
stop_service() {
    local service=$1
    local pid_file="logs/$service.pid"
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if ps -p $pid > /dev/null 2>&1; then
            echo -e "${YELLOW}Stopping $service (PID: $pid)...${NC}"
            kill $pid
            sleep 2
            
            # Force kill if still running
            if ps -p $pid > /dev/null 2>&1; then
                echo -e "${YELLOW}Force stopping $service...${NC}"
                kill -9 $pid 2>/dev/null || true
            fi
            
            rm -f "$pid_file"
            echo -e "${GREEN}✓ $service stopped${NC}"
        else
            echo -e "${YELLOW}✗ $service (PID: $pid) not running${NC}"
            rm -f "$pid_file"
        fi
    else
        echo -e "${YELLOW}✗ No PID file for $service${NC}"
    fi
}

# Stop all services
for service in ingest aggregator diagnoser ai-agent; do
    stop_service "$service"
done

# Also try to kill by process name (backup method)
echo -e "\n${YELLOW}Checking for any remaining processes...${NC}"
pkill -f "bin/ingest" 2>/dev/null && echo -e "${GREEN}✓ Killed remaining ingest processes${NC}" || true
pkill -f "bin/aggregator" 2>/dev/null && echo -e "${GREEN}✓ Killed remaining aggregator processes${NC}" || true
pkill -f "bin/diagnoser" 2>/dev/null && echo -e "${GREEN}✓ Killed remaining diagnoser processes${NC}" || true
pkill -f "bin/ai-agent" 2>/dev/null && echo -e "${GREEN}✓ Killed remaining ai-agent processes${NC}" || true

echo -e "\n${GREEN}✓ All services stopped${NC}"
echo -e "\n${BLUE}Note:${NC} Docker services are still running. Use 'make down' to stop them."
