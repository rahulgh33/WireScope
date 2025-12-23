#!/bin/bash

# Network QoE Platform Endpoint Testing Script

set -e

echo "🧪 Testing Network QoE Platform Endpoints..."

# Test Test Target Server
echo "🔍 Testing Test Target Server..."
health_response=$(curl -s http://localhost:8080/health)
if [[ "$health_response" == "OK" ]]; then
    echo "✅ Test target health endpoint working"
else
    echo "❌ Test target health endpoint failed"
    exit 1
fi

# Test slow endpoint
echo "🔍 Testing slow endpoint..."
slow_response=$(curl -s "http://localhost:8080/slow?ms=100")
if [[ "$slow_response" == *"Delayed response (100ms)"* ]]; then
    echo "✅ Test target slow endpoint working"
else
    echo "❌ Test target slow endpoint failed"
    exit 1
fi

# Test 1MB file
echo "🔍 Testing 1MB file endpoint..."
file_size=$(curl -s http://localhost:8080/fixed/1mb.bin | wc -c | tr -d ' ')
if [[ "$file_size" == "1048576" ]]; then
    echo "✅ Test target 1MB file endpoint working (size: $file_size bytes)"
else
    echo "❌ Test target 1MB file endpoint failed (size: $file_size bytes)"
    exit 1
fi

# Test Prometheus
echo "🔍 Testing Prometheus..."
prom_response=$(curl -s http://localhost:9090/-/healthy)
if [[ "$prom_response" == *"Prometheus Server is Healthy"* ]]; then
    echo "✅ Prometheus health endpoint working"
else
    echo "❌ Prometheus health endpoint failed"
    exit 1
fi

# Test Grafana
echo "🔍 Testing Grafana..."
grafana_response=$(curl -s http://localhost:3000/api/health)
if [[ "$grafana_response" == *"\"database\": \"ok\""* ]]; then
    echo "✅ Grafana health endpoint working"
else
    echo "❌ Grafana health endpoint failed"
    exit 1
fi

# Test Jaeger
echo "🔍 Testing Jaeger..."
jaeger_response=$(curl -s http://localhost:16686/api/services)
if [[ "$jaeger_response" == *"\"data\""* ]]; then
    echo "✅ Jaeger API endpoint working"
else
    echo "❌ Jaeger API endpoint failed"
    exit 1
fi

# Test Database
echo "🔍 Testing Database..."
db_tables=$(docker compose exec -T postgres psql -U telemetry -d telemetry -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';" -t | tr -d ' ')
if [[ "$db_tables" == "3" ]]; then
    echo "✅ Database schema created successfully (3 tables)"
else
    echo "❌ Database schema creation failed (found $db_tables tables)"
    exit 1
fi

echo ""
echo "🎉 All endpoint tests passed successfully!"
echo ""
echo "✅ Test Target Server: http://localhost:8080"
echo "✅ Prometheus: http://localhost:9090"
echo "✅ Grafana: http://localhost:3000 (admin/admin)"
echo "✅ Jaeger: http://localhost:16686"
echo "✅ Database: PostgreSQL with 3 tables created"
echo ""
echo "The Network QoE Telemetry Platform development environment is ready!"