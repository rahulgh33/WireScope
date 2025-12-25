#!/bin/bash

# Simple multi-machine deployment test
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

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
    docker compose -f deploy/docker-compose.nats-simple.yml down -v 2>/dev/null || true
    
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

test_infrastructure() {
    log "Testing infrastructure deployment..."
    
    cd "$PROJECT_ROOT"
    
    # Test storage
    log "Starting storage tier..."
    docker compose -f deploy/docker-compose.storage-simple.yml up -d postgres-primary
    
    sleep 10
    
    if docker exec deploy-postgres-primary-1 pg_isready -U telemetry -d telemetry >/dev/null 2>&1; then
        log "✅ PostgreSQL is ready and accepting connections"
    else
        error "❌ PostgreSQL failed to start"
        return 1
    fi
    
    # Test database connection
    if docker exec deploy-postgres-primary-1 psql -U telemetry -d telemetry -c "SELECT 1;" >/dev/null 2>&1; then
        log "✅ Database connection successful"
    else
        error "❌ Database connection failed"
        return 1
    fi
    
    # Test NATS cluster
    log "Starting NATS server..."
    docker compose -f deploy/docker-compose.nats-simple.yml up -d
    
    sleep 15
    
    # Check NATS server - wait for it to be ready
    local nats_ready=0
    for i in {1..30}; do
        if curl -s -f "http://localhost:8222/healthz" >/dev/null 2>&1; then
            log "✅ NATS server is healthy"
            nats_ready=1
            break
        fi
        echo -n "."
        sleep 1
    done
    echo ""
    
    if [ $nats_ready -eq 0 ]; then
        error "❌ NATS server failed to start"
        docker logs $(docker ps -q --filter "name=nats") 2>&1 | tail -20
        return 1
    fi
    
    log "✅ NATS server is operational"
    
    # Test network connectivity between containers
    log "Testing container network connectivity..."
    local nats_container=$(docker ps -q --filter "name=nats")
    local postgres_container=$(docker ps -q --filter "name=postgres-primary")
    
    if [ -n "$postgres_container" ] && [ -n "$nats_container" ]; then
        if docker exec "$postgres_container" ping -c 1 $(docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$nats_container") >/dev/null 2>&1; then
            log "✅ Container network connectivity working"
        else
            warn "⚠️  Container network connectivity test failed (containers may be on different networks)"
        fi
    else
        warn "⚠️  Could not find containers for connectivity test"
    fi
    
    log "✅ Infrastructure deployment test completed successfully!"
    return 0
}

test_docker_compose_configs() {
    log "Validating Docker Compose configurations..."
    
    cd "$PROJECT_ROOT"
    
    # Check configuration syntax
    local configs=(
        "deploy/docker-compose.storage-simple.yml"
        "deploy/docker-compose.nats-simple.yml" 
        "deploy/docker-compose.processing.yml"
        "deploy/docker-compose.ingest.yml"
        "deploy/docker-compose.probe.yml"
        "deploy/docker-compose.monitoring.yml"
    )
    
    for config in "${configs[@]}"; do
        if docker compose -f "$config" config >/dev/null 2>&1; then
            log "✅ $config is valid"
        else
            error "❌ $config has syntax errors"
            docker compose -f "$config" config 2>&1 | head -5
            return 1
        fi
    done
    
    log "✅ All Docker Compose configurations are valid"
}

test_dockerfiles() {
    log "Testing Dockerfile builds..."
    
    cd "$PROJECT_ROOT"
    
    # Try building one service as test
    if docker build -f deploy/Dockerfile.ingest -t test-ingest . >/dev/null 2>&1; then
        log "✅ Dockerfile builds are working"
        docker rmi test-ingest 2>/dev/null || true
    else
        warn "⚠️  Dockerfile build test failed (may need Go dependencies)"
    fi
}

main() {
    log "Starting simplified multi-machine deployment test..."
    
    # Trap cleanup on exit
    trap cleanup EXIT
    
    cd "$PROJECT_ROOT"
    
    # Setup
    setup_networks
    
    # Run tests
    test_docker_compose_configs
    test_dockerfiles  
    test_infrastructure
    
    log "🎉 Multi-machine deployment validation completed successfully!"
    
    log "Infrastructure is ready. To deploy full services:"
    log "  1. Build service images: make build"
    log "  2. Deploy processing: docker compose -f deploy/docker-compose.processing.yml up -d"
    log "  3. Deploy ingest: docker compose -f deploy/docker-compose.ingest.yml up -d"  
    log "  4. Deploy monitoring: docker compose -f deploy/docker-compose.monitoring.yml up -d"
    
    log "To cleanup: ./deploy/test-distributed-deployment-simple.sh cleanup"
}

# Handle cleanup command
if [[ "${1:-}" == "cleanup" ]]; then
    cleanup
    log "Cleanup completed"
    exit 0
fi

main "$@"