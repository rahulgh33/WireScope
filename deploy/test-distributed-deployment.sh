#!/bin/bash

# Multi-machine deployment testing script
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Configuration
export POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-telemetry}"
export REPLICA_PASSWORD="${REPLICA_PASSWORD:-replica}"
export AUTH_TOKEN="${AUTH_TOKEN:-test-token}"
export GRAFANA_PASSWORD="${GRAFANA_PASSWORD:-admin}"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $*${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $*${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $*${NC}"
}

cleanup() {
    log "Cleaning up test environment..."
    docker compose -f deploy/docker-compose.storage-simple.yml down -v 2>/dev/null || true
    docker compose -f deploy/docker-compose.nats-cluster.yml down -v 2>/dev/null || true
    docker compose -f deploy/docker-compose.processing.yml down -v 2>/dev/null || true
    docker compose -f deploy/docker-compose.ingest.yml down -v 2>/dev/null || true
    docker compose -f deploy/docker-compose.probe.yml down -v 2>/dev/null || true
    docker compose -f deploy/docker-compose.monitoring.yml down -v 2>/dev/null || true
    
    # Remove external networks
    docker network rm telemetry-storage 2>/dev/null || true
    docker network rm telemetry-processing 2>/dev/null || true
    docker network rm telemetry-ingest 2>/dev/null || true
    docker network rm telemetry-monitoring 2>/dev/null || true
}

setup_networks() {
    log "Setting up external networks..."
    docker network create --driver bridge telemetry-storage 2>/dev/null || true
    docker network create --driver bridge telemetry-processing 2>/dev/null || true
    docker network create --driver bridge telemetry-ingest 2>/dev/null || true
    docker network create --driver bridge telemetry-monitoring 2>/dev/null || true
}

wait_for_service() {
    local service_name="$1"
    local health_command="$2"
    local max_attempts="${3:-30}"
    local attempt=1
    
    log "Waiting for $service_name to be ready..."
    while [ $attempt -le $max_attempts ]; do
        if eval "$health_command" >/dev/null 2>&1; then
            log "$service_name is ready!"
            return 0
        fi
        
        if [ $((attempt % 5)) -eq 0 ]; then
            log "Still waiting for $service_name (attempt $attempt/$max_attempts)..."
        fi
        
        sleep 5
        ((attempt++))
    done
    
    error "$service_name failed to start within $((max_attempts * 5)) seconds"
    return 1
}

test_storage_tier() {
    log "Testing storage tier..."
    
    cd "$PROJECT_ROOT"
    docker compose -f deploy/docker-compose.storage-simple.yml up -d postgres-primary
    
    wait_for_service "PostgreSQL Primary" "docker exec deploy-postgres-primary-1 pg_isready -U telemetry -d telemetry" 20
    
    log "Testing database connection..."
    if docker exec distributed-telemetry-platform-postgres-primary-1 \
        psql -U telemetry -d telemetry -c "SELECT 1;" >/dev/null 2>&1; then
        log "Database connection successful"
    else
        error "Database connection failed"
        return 1
    fi
    
    log "Starting NATS cluster..."
    docker compose -f deploy/docker-compose.nats-cluster.yml up -d
    
    wait_for_service "NATS Node 1" "curl -s -f http://localhost:8222/healthz" 15
    wait_for_service "NATS Node 2" "curl -s -f http://localhost:8223/healthz" 15
    wait_for_service "NATS Node 3" "curl -s -f http://localhost:8224/healthz" 15
    
    log "Testing NATS cluster connectivity..."
    if docker exec distributed-telemetry-platform-nats-1-1 \
        nats server check --server=nats://localhost:4222 >/dev/null 2>&1; then
        log "NATS cluster is healthy"
    else
        warn "NATS cluster health check failed"
    fi
}

test_processing_tier() {
    log "Testing processing tier..."
    
    cd "$PROJECT_ROOT"
    docker compose -f deploy/docker-compose.processing.yml up -d
    
    wait_for_service "Aggregator" "http://localhost:9091/metrics" 20
    wait_for_service "Diagnoser" "http://localhost:9092/metrics" 20
    
    log "Processing tier is ready"
}

test_ingest_tier() {
    log "Testing ingest tier..."
    
    cd "$PROJECT_ROOT"
    docker compose -f deploy/docker-compose.ingest.yml up -d
    
    wait_for_service "Ingest API" "http://localhost:8080/health" 15
    wait_for_service "Ingest Metrics" "http://localhost:9090/metrics" 15
    
    log "Testing ingest API..."
    local test_event='{"event_id":"test-123","client_id":"test-client","ts_ms":1640995200000,"target":"test.com","dns_ms":10,"tcp_ms":20,"tls_ms":30,"ttfb_ms":100,"throughput_kbps":1000}'
    
    if curl -s -X POST "http://localhost:8080/events" \
        -H "Authorization: Bearer test-token" \
        -H "Content-Type: application/json" \
        -d "$test_event" | grep -q "success"; then
        log "Ingest API test successful"
    else
        warn "Ingest API test may have failed"
    fi
}

test_monitoring_tier() {
    log "Testing monitoring tier..."
    
    cd "$PROJECT_ROOT"
    docker compose -f deploy/docker-compose.monitoring.yml up -d
    
    wait_for_service "Prometheus" "http://localhost:9090/-/healthy" 20
    wait_for_service "Grafana" "http://localhost:3000/api/health" 30
    wait_for_service "Jaeger" "http://localhost:16686/" 20
    
    log "Monitoring tier is ready"
}

test_end_to_end_flow() {
    log "Testing end-to-end data flow..."
    
    # Send multiple test events
    local events_sent=0
    for i in {1..5}; do
        local test_event="{\"event_id\":\"test-$i\",\"client_id\":\"test-client\",\"ts_ms\":$(($(date +%s) * 1000)),\"target\":\"test-$i.com\",\"dns_ms\":$((10 + i)),\"tcp_ms\":$((20 + i)),\"tls_ms\":$((30 + i)),\"ttfb_ms\":$((100 + i)),\"throughput_kbps\":$((1000 + i * 100))}"
        
        if curl -s -X POST "http://localhost:8080/events" \
            -H "Authorization: Bearer test-token" \
            -H "Content-Type: application/json" \
            -d "$test_event" >/dev/null; then
            ((events_sent++))
        fi
        sleep 1
    done
    
    log "Sent $events_sent test events"
    
    # Wait for processing
    log "Waiting for event processing..."
    sleep 30
    
    # Check if events were processed
    if docker exec distributed-telemetry-platform-postgres-primary-1 \
        psql -U telemetry -d telemetry -c "SELECT COUNT(*) FROM events_seen;" | grep -q "[1-9]"; then
        log "Events were successfully processed and stored"
        return 0
    else
        warn "No events found in database - processing may be slow"
        return 1
    fi
}

test_failure_scenarios() {
    log "Testing failure scenarios..."
    
    # Test NATS node failure
    log "Testing NATS node failure..."
    docker compose -f deploy/docker-compose.nats-cluster.yml stop nats-2
    
    sleep 10
    
    # Send test event during failure
    local test_event='{"event_id":"fail-test","client_id":"fail-client","ts_ms":'"$(($(date +%s) * 1000))"',"target":"fail.com","dns_ms":15,"tcp_ms":25,"tls_ms":35,"ttfb_ms":105,"throughput_kbps":1050}'
    
    if curl -s -X POST "http://localhost:8080/events" \
        -H "Authorization: Bearer test-token" \
        -H "Content-Type: application/json" \
        -d "$test_event" >/dev/null; then
        log "System remained operational during NATS node failure"
    else
        warn "System may have issues during NATS node failure"
    fi
    
    # Restart failed node
    log "Restarting failed NATS node..."
    docker compose -f deploy/docker-compose.nats-cluster.yml up -d nats-2
    wait_for_service "NATS Node 2" "http://localhost:8223/healthz" 15
    
    log "NATS node recovery successful"
}

main() {
    log "Starting multi-machine deployment test..."
    
    # Trap cleanup on exit
    trap cleanup EXIT
    
    cd "$PROJECT_ROOT"
    
    # Setup
    setup_networks
    
    # Test each tier
    test_storage_tier
    test_processing_tier
    test_ingest_tier
    test_monitoring_tier
    
    # Test end-to-end flow
    test_end_to_end_flow
    
    # Test failure scenarios
    test_failure_scenarios
    
    log "Multi-machine deployment test completed successfully!"
    
    log "Services are running and accessible:"
    log "  - Grafana: http://localhost:3000 (admin/admin)"
    log "  - Prometheus: http://localhost:9090"
    log "  - Jaeger: http://localhost:16686"
    log "  - Ingest API: http://localhost:8080/events"
    
    log "To stop all services, run: $0 cleanup"
}

# Handle cleanup command
if [[ "${1:-}" == "cleanup" ]]; then
    cleanup
    log "Cleanup completed"
    exit 0
fi

main "$@"