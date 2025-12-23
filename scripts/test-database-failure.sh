#!/bin/bash
# Test database failure simulation with transaction retry
# Requirement: 8.5 - Database failure handling

set -euo pipefail

echo "=== Database Failure Test ==="

# Start infrastructure
docker compose up -d postgres nats
sleep 5

# Start aggregator
go run cmd/aggregator/main.go \
  -nats-url nats://localhost:4222 \
  -db-host localhost -db-port 5432 -db-name telemetry \
  -db-user telemetry -db-password telemetry \
  -window-size 60s -flush-delay 5s -metrics-port 9091 &
AGGREGATOR_PID=$!

# Start ingest API
go run cmd/ingest/main.go \
  -port 8080 -nats-url nats://localhost:4222 \
  -api-tokens "test-token-123" &
INGEST_PID=$!

cleanup() {
  echo "Cleaning up..."
  kill $AGGREGATOR_PID $INGEST_PID 2>/dev/null || true
  docker compose down
}
trap cleanup EXIT

sleep 5

echo "Sending events before database failure..."

# Send some events
for i in {1..3}; do
  curl -s -X POST http://localhost:8080/events \
    -H "Authorization: Bearer test-token-123" \
    -H "Content-Type: application/json" \
    -d "{
      \"event_id\": \"pre-failure-$i\",
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