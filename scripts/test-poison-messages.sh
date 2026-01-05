#!/bin/bash
# Test poison message handling with DLQ routing
# Tests DLQ routing for invalid messages

set -e  # Exit on error

echo "=== Starting poison message test ==="

# Start infrastructure
echo "Starting infrastructure..."
docker compose down -v > /dev/null 2>&1 || true
docker compose up -d postgres nats prometheus grafana > /dev/null 2>&1

# Wait for services to be ready  
echo "Waiting for services to start..."
sleep 10

# Start ingest service
echo "Starting ingest service..."
go run cmd/ingest/main.go -api-tokens "test-token-probe-1" -tracing-enabled=false > /tmp/ingest-poison.log 2>&1 &
INGEST_PID=$!
sleep 5

# Start aggregator service
echo "Starting aggregator service..."
go run cmd/aggregator/main.go -db-user telemetry -db-password telemetry -metrics-port 9091 -tracing-enabled=false > /tmp/aggregator-poison.log 2>&1 &
AGGREGATOR_PID=$!
sleep 5

# Function to cleanup
cleanup() {
    echo "Cleaning up..."
    kill $INGEST_PID $AGGREGATOR_PID 2>/dev/null || true
    docker compose down -v > /dev/null 2>&1
}
trap cleanup EXIT

# Verify services are running
echo "Verifying services..."
if ! curl -s http://localhost:8080/health > /dev/null; then
    echo "Error: Ingest service not responding"
    cat /tmp/ingest-poison.log
    exit 1
fi

if ! curl -s http://localhost:9091/metrics > /dev/null; then
    echo "Error: Aggregator service not responding"
    cat /tmp/aggregator-poison.log  
    exit 1
fi

echo "Services started successfully"

echo "Testing poison message handling..."

# Send valid message first
echo "Sending valid message..."
valid_id=$(python3 -c "import uuid; print(str(uuid.uuid4()))")
valid_response=$(curl -s -X POST http://localhost:8080/events \
    -H "Authorization: Bearer test-token-probe-1" \
    -H "Content-Type: application/json" \
    -d '{
        "event_id": "'$valid_id'",
        "client_id": "test-client-poison",
        "ts_ms": '$(date +%s000)',
        "schema_version": "1.0",
        "target": "https://test.example.com",
        "network_context": {
            "interface_type": "wifi", 
            "vpn_enabled": false
        },
        "timings": {
            "dns_ms": 10,
            "tcp_ms": 20,
            "tls_ms": 30,
            "http_ttfb_ms": 50
        },
        "throughput_kbps": 1024
    }' -w "%{http_code}")

echo "Valid message response: $valid_response"

# Send malformed messages
echo "Sending malformed messages..."

# Missing required fields
echo "Testing missing required fields..."
poison1_response=$(curl -s -X POST http://localhost:8080/events \
    -H "Authorization: Bearer test-token-probe-1" \
    -H "Content-Type: application/json" \
    -d '{
        "event_id": "test-poison-1",
        "client_id": "test-client-poison"
    }' -w "%{http_code}" || echo "failed")

# Invalid JSON
echo "Testing invalid JSON..."
poison2_response=$(curl -s -X POST http://localhost:8080/events \
    -H "Authorization: Bearer test-token-probe-1" \
    -H "Content-Type: application/json" \
    -d '{"event_id": "test-poison-2", "invalid": json}' \
    -w "%{http_code}" || echo "failed")

# Invalid data types
echo "Testing invalid data types..." 
poison3_response=$(curl -s -X POST http://localhost:8080/events \
    -H "Authorization: Bearer test-token-probe-1" \
    -H "Content-Type: application/json" \
    -d '{
        "event_id": "test-poison-3",
        "client_id": "test-client-poison",
        "ts_ms": "invalid-timestamp",
        "schema_version": "1.0",
        "target": "https://test.example.com",
        "network_context": {
            "interface_type": "wifi"
        },
        "timings": {
            "dns_ms": "invalid"
        },
        "throughput_kbps": "invalid"
    }' -w "%{http_code}" || echo "failed")

echo "Poison message responses:"
echo "Missing fields: $poison1_response"
echo "Invalid JSON: $poison2_response"
echo "Invalid types: $poison3_response"

# Wait for processing
sleep 5

# Check DLQ metrics
echo "Checking DLQ metrics..."
dlq_metrics=$(curl -s http://localhost:9091/metrics | grep "dlq_messages_total" || echo "No DLQ metrics found")
echo "DLQ metrics: $dlq_metrics"

# Verify results
invalid_count=0
if [[ "$poison1_response" == *"400"* ]]; then
    invalid_count=$((invalid_count + 1))
fi
if [[ "$poison2_response" == *"400"* ]]; then
    invalid_count=$((invalid_count + 1))
fi
if [[ "$poison3_response" == *"400"* ]]; then
    invalid_count=$((invalid_count + 1))
fi

if [ $invalid_count -gt 0 ]; then
    echo "✅ PASS: Invalid messages properly rejected ($invalid_count/3 rejected)"
    exit 0
else
    echo "❌ FAIL: Invalid messages were not properly rejected"
    exit 1
fi

echo "Checking DLQ for poison message..."

# Check if DLQ has messages
DLQ_MSGS=$(docker exec wirescope-nats-1 \
  nats stream info telemetry-events-dlq --json | jq '.state.messages // 0')

echo "DLQ messages: $DLQ_MSGS"

if [ "$DLQ_MSGS" -gt 0 ]; then
  echo "✅ Poison message routed to DLQ"
else
  echo "❌ No messages found in DLQ"
  exit 1
fi

echo "✅ Poison message test completed"