#!/bin/bash
# Test database failure simulation with transaction retry  
# Tests recovery after PostgreSQL outage

set -e  # Exit on error

echo "=== Starting database failure test ==="

# Start infrastructure
echo "Starting infrastructure..."
docker compose down -v > /dev/null 2>&1 || true
docker compose up -d postgres nats prometheus grafana > /dev/null 2>&1

# Wait for services to be ready
echo "Waiting for services to start..."
sleep 10

# Start ingest service
echo "Starting ingest service..."
go run cmd/ingest/main.go -api-tokens "test-token-probe-1" -tracing-enabled=false > /tmp/ingest-db.log 2>&1 &
INGEST_PID=$!
sleep 5

# Start aggregator service
echo "Starting aggregator service..."
go run cmd/aggregator/main.go -db-user telemetry -db-password telemetry -metrics-port 9091 -tracing-enabled=false > /tmp/aggregator-db.log 2>&1 &
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
    cat /tmp/ingest-db.log
    exit 1
fi

if ! curl -s http://localhost:9091/metrics > /dev/null; then
    echo "Error: Aggregator service not responding"
    cat /tmp/aggregator-db.log
    exit 1
fi

echo "Services started successfully"

# Function to send event
send_event() {
    local event_id=$1
    local phase=$2
    curl -s -X POST http://localhost:8080/events \
        -H "Authorization: Bearer test-token-probe-1" \
        -H "Content-Type: application/json" \
        -d '{
            "event_id": "'$event_id'",
            "client_id": "test-client-db-'$phase'",
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
        }' -w "%{http_code}\n"
}

# Send events before database failure
echo "Sending events before database failure..."
pre_failure_success=0
for i in {1..3}; do
    event_id=$(python3 -c "import uuid; print(str(uuid.uuid4()))")
    response=$(send_event "$event_id" "pre")
    if [[ "$response" == *"202" ]]; then
        pre_failure_success=$((pre_failure_success + 1))
    fi
    sleep 1
done

echo "Pre-failure events accepted: $pre_failure_success/3"

# Wait for processing
sleep 5

# Stop PostgreSQL
echo "Stopping PostgreSQL to simulate database failure..."
docker compose stop postgres
sleep 3

# Try to send events during failure
echo "Sending events during database failure..."
failure_success=0
for i in {1..3}; do
    event_id=$(python3 -c "import uuid; print(str(uuid.uuid4()))")
    response=$(send_event "$event_id" "during" 2>/dev/null || echo "failed")
    if [[ "$response" == *"202" ]]; then
        failure_success=$((failure_success + 1))
    fi
    sleep 1
done

echo "During-failure events accepted: $failure_success/3"

# Restart PostgreSQL
echo "Restarting PostgreSQL..."
docker compose up -d postgres
sleep 10

# Send events after recovery
echo "Sending events after database recovery..."
post_failure_success=0
for i in {1..3}; do
    event_id=$(python3 -c "import uuid; print(str(uuid.uuid4()))")
    response=$(send_event "$event_id" "post")
    if [[ "$response" == *"202" ]]; then
        post_failure_success=$((post_failure_success + 1))
    fi
    sleep 1
done

echo "Post-failure events accepted: $post_failure_success/3"

# Check aggregator health after recovery
echo "Checking aggregator health after recovery..."
sleep 5
if curl -s http://localhost:9091/metrics > /dev/null; then
    echo "✅ Aggregator recovered successfully"
    aggregator_recovered=true
else
    echo "❌ Aggregator did not recover"
    aggregator_recovered=false
fi

# Summary
echo ""
echo "=== Database Failure Test Results ==="
echo "Pre-failure events: $pre_failure_success/3"
echo "During-failure events: $failure_success/3" 
echo "Post-failure events: $post_failure_success/3"
echo "Aggregator recovery: $aggregator_recovered"

# Verify results
if [ $pre_failure_success -eq 3 ] && [ $post_failure_success -eq 3 ] && [ "$aggregator_recovered" = true ]; then
    echo "✅ PASS: Database failure handling working correctly"
    exit 0
else
    echo "❌ FAIL: Database failure handling not working properly"
    exit 1
fi
      \"throughput_kbps\": 1000
    }"
done

sleep 5

echo "Simulating database failure by stopping PostgreSQL..."
docker compose stop postgres

sleep 2

echo "Sending events during database failure..."

# Send events during failure (these should be queued)
for i in {1..3}; do
  EVENT_UUID=$(python3 -c "import uuid; print(uuid.uuid4())")
  curl -s -X POST http://localhost:8080/events \
    -H "Authorization: Bearer test-token-123" \
    -H "Content-Type: application/json" \
    -d "{
      \"event_id\": \"$EVENT_UUID\",
      \"client_id\": \"db-test-client\",
      \"ts_ms\": $(date +%s000),
      \"schema_version\": \"1.0\",
      \"target\": \"http://test.com\",
      \"network_context\": {\"interface_type\": \"wifi\", \"vpn_enabled\": false},
      \"timings\": {\"dns_ms\": 5, \"tcp_ms\": 10, \"tls_ms\": 15, \"http_ttfb_ms\": 50},
      \"throughput_kbps\": 1000
    }"
done

sleep 5

echo "Restarting PostgreSQL..."
docker compose start postgres

sleep 10

echo "Sending events after database recovery..."

# Send events after recovery
for i in {1..3}; do
  EVENT_UUID=$(python3 -c "import uuid; print(uuid.uuid4())")
  curl -s -X POST http://localhost:8080/events \
    -H "Authorization: Bearer test-token-123" \
    -H "Content-Type: application/json" \
    -d "{
      \"event_id\": \"$EVENT_UUID\",
      \"client_id\": \"db-test-client\",
      \"ts_ms\": $(date +%s000),
      \"schema_version\": \"1.0\",
      \"target\": \"http://test.com\",
      \"network_context\": {\"interface_type\": \"wifi\", \"vpn_enabled\": false},
      \"timings\": {\"dns_ms\": 5, \"tcp_ms\": 10, \"tls_ms\": 15, \"http_ttfb_ms\": 50},
      \"throughput_kbps\": 1000
    }"
done

sleep 10

echo "Checking final aggregates..."
FINAL_COUNT=$(psql "postgresql://telemetry:telemetry@localhost:5432/telemetry" -t -c "
SELECT count(*) FROM agg_1m WHERE client_id = 'db-test-client';
" | xargs)

echo "Final aggregate count: $FINAL_COUNT"

if [ "$FINAL_COUNT" -gt 0 ]; then
  echo "✅ Database failure test passed - aggregates processed after recovery"
else
  echo "❌ No aggregates found after recovery"
  exit 1
fi

echo "✅ Database failure test completed"