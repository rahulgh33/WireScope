#!/bin/bash
# Test poison message handling with DLQ routing
# Requirement: 8.1 - Poison message handling

set -euo pipefail

echo "=== Poison Message Test ==="

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

cleanup() {
  echo "Cleaning up..."
  kill $AGGREGATOR_PID 2>/dev/null || true
  docker compose down
}
trap cleanup EXIT

sleep 5

echo "Publishing poison message directly to NATS..."

# Use nats CLI to publish invalid JSON
docker exec distributed-telemetry-platform-nats-1 \
  nats pub telemetry.events '{"invalid": "json", missing required fields}'

sleep 10

echo "Checking DLQ for poison message..."

# Check if DLQ has messages
DLQ_MSGS=$(docker exec distributed-telemetry-platform-nats-1 \
  nats stream info telemetry-events-dlq --json | jq '.state.messages // 0')

echo "DLQ messages: $DLQ_MSGS"

if [ "$DLQ_MSGS" -gt 0 ]; then
  echo "✅ Poison message routed to DLQ"
else
  echo "❌ No messages found in DLQ"
  exit 1
fi

echo "✅ Poison message test completed"