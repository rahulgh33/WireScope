#!/bin/bash

# WireScope Setup Validation Script

set -e

echo "🔍 Validating WireScope Setup..."

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "❌ Docker is not installed or not in PATH"
    exit 1
fi

# Check if Docker Compose is available
if ! docker compose version &> /dev/null; then
    echo "❌ Docker Compose is not available"
    exit 1
fi

echo "✅ Docker and Docker Compose are available"

# Validate Docker Compose configuration
echo "🔍 Validating Docker Compose configuration..."
if docker compose config --quiet; then
    echo "✅ Docker Compose configuration is valid"
else
    echo "❌ Docker Compose configuration has errors"
    exit 1
fi

# Check if required files exist
echo "🔍 Checking required files..."

required_files=(
    "docker-compose.yml"
    "Makefile"
    "go.mod"
    "config/prometheus.yml"
    "config/otel-collector.yml"
    "config/init.sql"
    "config/test-server.py"
    "config/Dockerfile.test-server"
    "migrations/001_initial_schema.up.sql"
    "migrations/001_initial_schema.down.sql"
)

for file in "${required_files[@]}"; do
    if [[ -f "$file" ]]; then
        echo "✅ $file exists"
    else
        echo "❌ $file is missing"
        exit 1
    fi
done

# Check directory structure
echo "🔍 Checking directory structure..."

required_dirs=(
    "cmd/probe"
    "cmd/ingest"
    "cmd/aggregator"
    "cmd/diagnoser"
    "internal"
    "pkg"
    "config"
    "migrations"
)

for dir in "${required_dirs[@]}"; do
    if [[ -d "$dir" ]]; then
        echo "✅ $dir/ directory exists"
    else
        echo "❌ $dir/ directory is missing"
        exit 1
    fi
done

# Test Docker Compose services can be parsed
echo "🔍 Testing Docker Compose services..."
services=$(docker compose config --services)
expected_services=("postgres" "nats" "prometheus" "grafana" "jaeger" "otel-collector" "test-target")

for service in "${expected_services[@]}"; do
    if echo "$services" | grep -q "^$service$"; then
        echo "✅ Service '$service' is configured"
    else
        echo "❌ Service '$service' is missing from configuration"
        exit 1
    fi
done

echo ""
echo "🎉 Setup validation completed successfully!"
echo ""
echo "Next steps:"
echo "1. Run 'make dev' to start the development environment"
echo "2. Run 'make build' to compile the Go binaries"
echo "3. Check service health at:"
echo "   - Grafana: http://localhost:3000 (admin/admin)"
echo "   - Prometheus: http://localhost:9090"
echo "   - Jaeger: http://localhost:16686"
echo "   - Test Target: http://localhost:8080"