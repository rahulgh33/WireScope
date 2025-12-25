#!/bin/bash
# Test script for burst traffic handling 
# Tests rate limiting under high load conditions

set -e  # Exit on error

echo "=== Starting burst traffic test ==="

# Start infrastructure
echo "Starting infrastructure..."
docker compose down -v > /dev/null 2>&1 || true
docker compose up -d postgres nats prometheus grafana > /dev/null 2>&1

# Wait for services to be ready
echo "Waiting for services to start..."
sleep 10

# Start ingest service
echo "Starting ingest service..."
go run cmd/ingest/main.go -api-tokens "test-token-probe-1" -rate-limit 5 -rate-limit-burst 3 -tracing-enabled=false > /tmp/ingest-burst.log 2>&1 &
INGEST_PID=$!
sleep 5

# Start aggregator service  
echo "Starting aggregator service..."
go run cmd/aggregator/main.go -db-user telemetry -db-password telemetry -metrics-port 9091 -tracing-enabled=false > /tmp/aggregator-burst.log 2>&1 &
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
    cat /tmp/ingest-burst.log
    exit 1
fi

if ! curl -s http://localhost:9091/metrics > /dev/null; then
    echo "Error: Aggregator service not responding"  
    cat /tmp/aggregator-burst.log
    exit 1
fi

echo "Services started successfully"

# Function to send event
send_event() {
    local event_id=$(python3 -c "import uuid; print(str(uuid.uuid4()))")
    curl -s -X POST http://localhost:8080/events \
        -H "Authorization: Bearer test-token-probe-1" \
        -H "Content-Type: application/json" \
        -d '{
            "event_id": "'$event_id'",
            "client_id": "test-client-burst",
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
            "throughput_kbps": 1024,
            "traceparent": "trace-'$event_id'",
            "tracestate": "span-'$event_id'"
        }' -w "%{http_code}\n"
}

# Send burst of events
echo "Sending burst of 20 events..."
accepted=0
rate_limited=0

for i in {1..20}; do
    response=$(send_event)
    http_code=$(echo "$response" | tail -1)
    if [[ "$http_code" == "202" ]]; then
        accepted=$((accepted + 1))
    elif [[ "$http_code" == "429" ]]; then
        rate_limited=$((rate_limited + 1))
    fi
    sleep 0.05  # Small delay between requests
done

echo "Events sent: 20"
echo "Accepted: $accepted"  
echo "Rate limited: $rate_limited"

# Check rate limiting metrics
echo "Checking rate limiting metrics..."
sleep 2
rate_limit_hits=$(curl -s http://localhost:8080/metrics | grep "ingest_rate_limit_hits_total" | grep -o '[0-9]\+$' | head -1 || echo "0")
echo "Rate limit hits from metrics: $rate_limit_hits"

# Verify test results
if [ $rate_limited -gt 0 ]; then
    echo "✅ PASS: Rate limiting is working (blocked $rate_limited requests)"
    exit 0
else
    echo "❌ FAIL: No requests were rate limited"
    exit 1
fi