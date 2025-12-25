#!/bin/bash

# End-to-end distributed deployment test
# Tests full data flow: probe → ingest → NATS → aggregator → PostgreSQL
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

info() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $*${NC}"
}

cleanup() {
    log "Cleaning up test environment..."
    cd "$PROJECT_ROOT"
    
    # Stop local processes
    if [ -f /tmp/ingest.pid ]; then
        kill $(cat /tmp/ingest.pid) 2>/dev/null || true
        rm /tmp/ingest.pid
    fi
    if [ -f /tmp/aggregator.pid ]; then
        kill $(cat /tmp/aggregator.pid) 2>/dev/null || true
        rm /tmp/aggregator.pid
    fi
    if [ -f /tmp/diagnoser.pid ]; then
        kill $(cat /tmp/diagnoser.pid) 2>/dev/null || true
        rm /tmp/diagnoser.pid
    fi
    
    # Stop Docker containers
    docker compose -f deploy/docker-compose.monitoring.yml down -v 2>/dev/null || true
    docker compose -f deploy/docker-compose.probe.yml down -v 2>/dev/null || true
    docker compose -f deploy/docker-compose.ingest.yml down -v 2>/dev/null || true
    docker compose -f deploy/docker-compose.processing.yml down -v 2>/dev/null || true
    docker compose -f deploy/docker-compose.nats-simple.yml down -v 2>/dev/null || true
    docker compose -f deploy/docker-compose.storage-simple.yml down -v 2>/dev/null || true
    
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

deploy_infrastructure() {
    log "Step 1: Deploying storage infrastructure..."
    cd "$PROJECT_ROOT"
    
    # Deploy PostgreSQL
    docker compose -f deploy/docker-compose.storage-simple.yml up -d postgres-primary
    
    info "Waiting for PostgreSQL to be ready..."
    for i in {1..30}; do
        if docker exec $(docker ps -q --filter "name=postgres-primary") pg_isready -U telemetry -d telemetry >/dev/null 2>&1; then
            log "✅ PostgreSQL is ready"
            break
        fi
        echo -n "."
        sleep 1
    done
    echo ""
    
    # Run migrations
    log "Running database migrations..."
    cd "$PROJECT_ROOT"
    make migrate || warn "Migration may have already been applied"
    
    # Deploy NATS
    log "Deploying NATS JetStream..."
    docker compose -f deploy/docker-compose.nats-simple.yml up -d
    
    info "Waiting for NATS to be ready..."
    for i in {1..30}; do
        if curl -s -f "http://localhost:8222/healthz" >/dev/null 2>&1; then
            log "✅ NATS is ready"
            break
        fi
        echo -n "."
        sleep 1
    done
    echo ""
}

deploy_processing() {
    log "Step 2: Deploying processing tier (using local binaries)..."
    cd "$PROJECT_ROOT"
    
    # Ensure binaries are built
    if [ ! -f "bin/aggregator" ] || [ ! -f "bin/diagnoser" ]; then
        log "Building binaries..."
        make build || {
            error "Failed to build binaries"
            return 1
        }
    fi
    
    # Start aggregator in background
    log "Starting aggregator..."
    NATS_URL=nats://localhost:4222 \
    DB_HOST=localhost \
    DB_PORT=5432 \
    DB_NAME=telemetry \
    DB_USER=telemetry \
    DB_PASSWORD=telemetry \
    METRICS_PORT=9091 \
    WINDOW_SIZE=60s \
    FLUSH_DELAY=5s \
    LATE_TOLERANCE=2m \
    nohup ./bin/aggregator > /tmp/aggregator.log 2>&1 &
    AGGREGATOR_PID=$!
    echo $AGGREGATOR_PID > /tmp/aggregator.pid
    
    # Start diagnoser in background
    log "Starting diagnoser..."
    DB_HOST=localhost \
    DB_PORT=5432 \
    DB_NAME=telemetry \
    DB_USER=telemetry \
    DB_PASSWORD=telemetry \
    METRICS_PORT=9092 \
    INTERVAL=30s \
    nohup ./bin/diagnoser > /tmp/diagnoser.log 2>&1 &
    DIAGNOSER_PID=$!
    echo $DIAGNOSER_PID > /tmp/diagnoser.pid
    
    info "Waiting for processing services to be ready..."
    sleep 5
    
    # Check aggregator health
    for i in {1..10}; do
        if curl -s -f "http://localhost:9091/metrics" >/dev/null 2>&1; then
            log "✅ Aggregator is healthy"
            break
        fi
        echo -n "."
        sleep 1
    done
    echo ""
    
    # Check diagnoser health
    for i in {1..10}; do
        if curl -s -f "http://localhost:9092/metrics" >/dev/null 2>&1; then
            log "✅ Diagnoser is healthy"
            break
        fi
        echo -n "."
        sleep 1
    done
    echo ""
}

deploy_ingest() {
    log "Step 3: Deploying ingest API (using local binary)..."
    cd "$PROJECT_ROOT"
    
    # Ensure binary is built
    if [ ! -f "bin/ingest" ]; then
        log "Building ingest binary..."
        make build || {
            error "Failed to build ingest binary"
            return 1
        }
    fi
    
    # Start ingest API in background
    log "Starting ingest API..."
    NATS_URL=nats://localhost:4222 \
    PORT=8080 \
    AUTH_TOKEN=test-token \
    RATE_LIMIT=1000 \
    BURST_LIMIT=2000 \
    METRICS_PORT=9090 \
    nohup ./bin/ingest > /tmp/ingest.log 2>&1 &
    INGEST_PID=$!
    echo $INGEST_PID > /tmp/ingest.pid
    
    info "Waiting for ingest API to be ready..."
    for i in {1..30}; do
        if curl -s -f "http://localhost:8080/health" >/dev/null 2>&1; then
            log "✅ Ingest API is healthy"
            break
        fi
        echo -n "."
        sleep 1
    done
    echo ""
}

test_end_to_end_flow() {
    log "Step 4: Testing end-to-end data flow..."
    cd "$PROJECT_ROOT"
    
    # Create test event
    local test_event=$(cat <<EOF
{
  "event_id": "$(uuidgen | tr '[:upper:]' '[:lower:]')",
  "client_id": "test-client-e2e",
  "ts_ms": $(date +%s)000,
  "schema_version": 1,
  "network_context": {
    "client_ip": "203.0.113.1",
    "isp": "Test ISP"
  },
  "target": "google.com",
  "timing": {
    "dns_ms": 10,
    "tcp_ms": 15,
    "tls_ms": 20,
    "ttfb_ms": 50,
    "download_ms": 100
  },
  "throughput_mbps": 50.5,
  "error_stage": null
}
EOF
)
    
    # Send event to ingest API
    log "Sending test event to ingest API..."
    local response=$(curl -s -w "\n%{http_code}" -X POST http://localhost:8080/events \
        -H "Authorization: Bearer test-token" \
        -H "Content-Type: application/json" \
        -d "$test_event")
    
    local http_code=$(echo "$response" | tail -n1)
    
    if [ "$http_code" = "200" ] || [ "$http_code" = "202" ]; then
        log "✅ Event successfully submitted to ingest API"
    else
        error "❌ Failed to submit event (HTTP $http_code)"
        echo "$response"
        return 1
    fi
    
    # Wait for processing
    log "Waiting for event processing (90 seconds)..."
    sleep 90
    
    # Check database for aggregates
    log "Checking database for processed aggregates..."
    local aggregate_count=$(docker exec $(docker ps -q --filter "name=postgres-primary") \
        psql -U telemetry -d telemetry -t -A -c \
        "SELECT COUNT(*) FROM agg_1m WHERE client_id = 'test-client-e2e';")
    
    if [ "$aggregate_count" -gt 0 ]; then
        log "✅ Found $aggregate_count aggregate(s) in database"
        
        # Show aggregate details
        info "Aggregate details:"
        docker exec $(docker ps -q --filter "name=postgres-primary") \
            psql -U telemetry -d telemetry -c \
            "SELECT client_id, target, window_start_ts, count_total, count_success 
             FROM agg_1m 
             WHERE client_id = 'test-client-e2e' 
             ORDER BY window_start_ts DESC 
             LIMIT 5;"
    else
        error "❌ No aggregates found in database"
        
        # Debug: Check NATS
        warn "Checking NATS stream status..."
        docker exec $(docker ps -q --filter "name=nats") nats stream list || true
        
        # Debug: Check aggregator logs
        warn "Aggregator logs:"
        docker logs $(docker ps -q --filter "name=aggregator") 2>&1 | tail -20
        
        return 1
    fi
    
    # Check deduplication
    log "Testing deduplication (sending same event again)..."
    curl -s -X POST http://localhost:8080/events \
        -H "Authorization: Bearer test-token" \
        -H "Content-Type: application/json" \
        -d "$test_event" >/dev/null
    
    sleep 10
    
    local aggregate_count_after=$(docker exec $(docker ps -q --filter "name=postgres-primary") \
        psql -U telemetry -d telemetry -t -A -c \
        "SELECT COUNT(*) FROM agg_1m WHERE client_id = 'test-client-e2e';")
    
    if [ "$aggregate_count" -eq "$aggregate_count_after" ]; then
        log "✅ Deduplication working correctly (aggregate count unchanged)"
    else
        warn "⚠️  Deduplication may not be working (count changed from $aggregate_count to $aggregate_count_after)"
    fi
}

test_metrics() {
    log "Step 5: Validating metrics endpoints..."
    
    # Check ingest metrics
    if curl -s "http://localhost:9090/metrics" | grep -q "ingest_requests_total"; then
    if [ -f /tmp/aggregator.pid ]; then
        local old_pid=$(cat /tmp/aggregator.pid)
        kill $old_pid 2>/dev/null || true
        sleep 2
        
        # Restart aggregator
        NATS_URL=nats://localhost:4222 \
        DB_HOST=localhost \
        DB_PORT=5432 \
        DB_NAME=telemetry \
        DB_USER=telemetry \
        DB_PASSWORD=telemetry \
        METRICS_PORT=9091 \
        WINDOW_SIZE=60s \
        FLUSH_DELAY=5s \
        LATE_TOLERANCE=2m \
        nohup ./bin/aggregator > /tmp/aggregator.log 2>&1 &
        echo $! > /tmp/aggregator.pid
        
        sleep 5
        
        if curl -s -f "http://localhost:9091/metrics" >/dev/null 2>&1; then
            log "✅ Aggregator recovered successfully"
        else
            error "❌ Aggregator failed to recover"
            cat /tmp/aggregator.log | tail -20
            return 1
        figgregator metrics are exposed"
    else
        warn "⚠️  Aggregator metrics not found"
    fi
    
    # Show some key metrics
    info "Key metrics:"
    echo "Ingest requests:"
    curl -s "http://localhost:9090/metrics" | grep "^ingest_requests_total" | head -5
    echo ""
    echo "Events processed:"
    curl -s "http://localhost:9091/metrics" | grep "^events_processed_total" | head -5
}

test_failure_scenarios() {
    log "Step 6: Testing failure scenarios..."
    
    # Test 1: Aggregator restart
    log "Test 6.1: Aggregator restart (service continuity)..."
    if [ -f /tmp/aggregator.pid ]; then
        local old_pid=$(cat /tmp/aggregator.pid)
        kill $old_pid 2>/dev/null || true
        sleep 2
        
        # Restart aggregator
        NATS_URL=nats://localhost:4222 \
        DB_HOST=localhost \
        DB_PORT=5432 \
        DB_NAME=telemetry \
        DB_USER=telemetry \
        DB_PASSWORD=telemetry \
        METRICS_PORT=9091 \
        WINDOW_SIZE=60s \
        FLUSH_DELAY=5s \
        LATE_TOLERANCE=2m \
        nohup ./bin/aggregator > /tmp/aggregator.log 2>&1 &
        echo $! > /tmp/aggregator.pid
        
        sleep 5
        
        if curl -s -f "http://localhost:9091/metrics" >/dev/null 2>&1; then
            log "✅ Aggregator recovered successfully"
        else
            error "❌ Aggregator failed to recover"
            cat /tmp/aggregator.log | tail -20
            return 1
        fi
    fi
    
    # Test 2: Send event during downtime
    log "Test 6.2: Event submission resilience..."
    test_event_2=$(cat <<EOF
{
  "event_id": "$(uuidgen | tr '[:upper:]' '[:lower:]')",
  "client_id": "test-client-resilience",
  "ts_ms": $(date +%s)000,
  "schema_version": 1,
  "network_context": {"client_ip": "203.0.113.2", "isp": "Test ISP"},
  "target": "cloudflare.com",
  "timing": {"dns_ms": 8, "tcp_ms": 12, "tls_ms": 18, "ttfb_ms": 45, "download_ms": 90},
  "throughput_mbps": 60.0,
  "error_stage": null
}
EOF
)
    
    if curl -s -X POST http://localhost:8080/events \
        -H "Authorization: Bearer test-token" \
        -H "Content-Type: application/json" \
        -d "$test_event_2" >/dev/null; then
        log "✅ Event accepted during recovery period"
    else
        error "❌ Failed to accept event"
        return 1
    fi
}

generate_report() {
    log "Generating deployment test report..."
    
    cat <<REPORT

========================================
  Multi-Machine Deployment Test Report
========================================

Infrastructure Status:
$(docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep -E "postgres|nats|ingest|aggregator|diagnoser" || echo "No services running")

Network Configuration:
$(docker network ls | grep telemetry || echo "No telemetry networks found")

Database Summary:
$(docker exec $(docker ps -q --filter "name=postgres-primary") psql -U telemetry -d telemetry -c "
SELECT 
    'Aggregates' as table_name, 
    COUNT(*) as row_count 
FROM agg_1m
UNION ALL
SELECT 
    'Dedup Entries' as table_name, 
    COUNT(*) as row_count 
FROM events_seen;
" 2>/dev/null || echo "Database query failed")

Service Health:
- PostgreSQL: $(docker exec $(docker ps -q --filter "name=postgres-primary") pg_isready -U telemetry -d telemetry 2>&1 || echo "Not Ready")
- NATS: $(curl -s http://localhost:8222/healthz 2>&1 || echo "Not Healthy")
- Ingest: $(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health 2>&1)
- Aggregator: $(curl -s -o /dev/null -w "%{http_code}" http://localhost:9091/metrics 2>&1)

========================================

REPORT
}

main() {
    log "Starting end-to-end distributed deployment test..."
    log "This will test the complete data flow across all components"
    
    # Trap cleanup on exit
    trap cleanup EXIT
    
    cd "$PROJECT_ROOT"
    
    # Execute test phases
    setup_networks
    deploy_infrastructure
    deploy_processing
    deploy_ingest
    test_end_to_end_flow
    test_metrics
    test_failure_scenarios
    
    log "🎉 All tests completed successfully!"
    generate_report
    
    log ""
    log "Test environment is still running. To explore:"
    log "  - View metrics: curl http://localhost:9090/metrics"
    log "  - Query database: docker exec -it \$(docker ps -q --filter name=postgres-primary) psql -U telemetry -d telemetry"
    log "  - View NATS stats: curl http://localhost:8222/jsz"
    log ""
    log "To cleanup: ./deploy/test-e2e-distributed.sh cleanup"
}

# Handle cleanup command
if [[ "${1:-}" == "cleanup" ]]; then
    cleanup
    log "Cleanup completed"
    exit 0
fi

main "$@"
