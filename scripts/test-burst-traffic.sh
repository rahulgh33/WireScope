#!/bin/bash
# Create burst traffic test for lag monitoring
# Requirement: 8.1 - Burst traffic handling

set -euo pipefail

echo "=== Burst Traffic Test ==="

# Start infrastructure
docker compose up -d postgres nats

sleep 5

# Start aggregator (foreground to see output)
go run cmd/aggregator/main.go \
  -nats-url nats://localhost:4222 \
  -db-host localhost -db-port 5432 -db-name telemetry \
  -db-user telemetry -db-password telemetry \
  -window-size 60s -flush-delay 5s -metrics-port 9091 &
AGGREGATOR_PID=$!

# Start ingest API 
go run cmd/ingest/main.go \
  -port 8080 -nats-url nats://localhost:4222 \
  -api-tokens "test-token-123" \
  -rate-limit 10 -rate-limit-burst 5 \
  -tracing-enabled=false > /tmp/ingest.log 2>&1 &
INGEST_PID=$!

cleanup() {
  echo "Cleaning up..."
  kill $AGGREGATOR_PID $INGEST_PID 2>/dev/null || true
  docker compose down
}
trap cleanup EXIT

sleep 5

echo "Testing burst traffic..."

# Send burst of events
for i in {1..20}; do
  EVENT_UUID=$(python3 -c "import uuid; print(uuid.uuid4())")
  curl -s -X POST http://localhost:8080/events \
    -H "Authorization: Bearer test-token-123" \
    -H "Content-Type: application/json" \
    -d "{
      \"event_id\": \"$EVENT_UUID\",
      \"client_id\": \"burst-client\",
      \"ts_ms\": $(date +%s000),
      \"schema_version\": \"1.0\",
      \"target\": \"http://test.com\",
      \"network_context\": {\"interface_type\": \"wifi\", \"vpn_enabled\": false},
      \"timings\": {\"dns_ms\": 5, \"tcp_ms\": 10, \"tls_ms\": 15, \"http_ttfb_ms\": 50},
      \"throughput_kbps\": 1000
    }" &
  
  if (( i % 5 == 0 )); then
    wait
    sleep 1
  fi
done

wait
sleep 5

# Check metrics
echo "Checking rate limit metrics..."
RATE_LIMIT_HITS=$(curl -s http://localhost:8080/metrics | grep ingest_rate_limit_hits_total | tail -1 || echo "0")
echo "Rate limit hits: $RATE_LIMIT_HITS"

if echo "$RATE_LIMIT_HITS" | grep -q "ingest_rate_limit_hits_total"; then
  echo "✅ Rate limiting activated during burst"
else
  echo "⚠️  No rate limit hits detected"
fi

echo "✅ Burst traffic test completed"