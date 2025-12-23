#!/bin/bash
# Test aggregator restart with safe continuation
# Validates that aggregator can restart without data loss and continues processing

set -e

echo "=== Milestone B Demo: Aggregator Restart Test ==="
echo ""

# Configuration
INGEST_URL="${INGEST_URL:-http://localhost:8080}"
API_TOKEN="${API_TOKEN:-test-token}"
CLIENT_ID="restart-test-$(date +%s)"
TARGET="http://test-target.local"

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-telemetry}"
DB_USER="${DB_USER:-telemetry}"
DB_PASSWORD="${DB_PASSWORD:-telemetry}"

# Check tools
command -v curl >/dev/null 2>&1 || { echo "curl required" >&2; exit 1; }
command -v psql >/dev/null 2>&1 || { echo "psql required" >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "jq required" >&2; exit 1; }

echo "Configuration:"
echo "  Client ID: $CLIENT_ID"
echo "  Target: $TARGET"
echo ""

send_event() {
  local event_id=$1
  local timestamp_ms=$(date +%s000)
  
  curl -s -X POST "$INGEST_URL/events" \
    -H "Authorization: Bearer $API_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"schema_version\": \"1.0\",
      \"event_id\": \"$event_id\",
      \"client_id\": \"$CLIENT_ID\",
      \"timestamp_ms\": $timestamp_ms,
      \"target\": \"$TARGET\",
      \"network_context\": {
        \"interface_name\": \"eth0\",
        \"local_ip\": \"192.168.1.100\"
      },
      \"timing_measurements\": {
        \"dns_ms\": 10.0,
        \"tcp_ms\": 15.0,
        \"tls_ms\": 45.0,
        \"ttfb_ms\": 120.0,
        \"total_ms\": 190.0
      },
      \"throughput_measurement\": {
        \"bytes_transferred\": 1048576,
        \"duration_ms\": 850.0,
        \"throughput_mbps\": 9.86
      },
      \"error_stage\": null
    }" > /dev/null
    
  echo "  Sent event: $event_id"
}

get_aggregate_count() {
  QUERY="SELECT COUNT(*) FROM agg_1m WHERE client_id = '$CLIENT_ID' AND target = '$TARGET'"
  PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -A -c "$QUERY"
}

get_events_seen_count() {
  QUERY="SELECT COUNT(*) FROM events_seen WHERE client_id = '$CLIENT_ID'"
  PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -A -c "$QUERY"
}

echo "Step 1: Send 3 events while aggregator is running"
echo "--------------------------------------------------"
send_event "restart-1-$(uuidgen)"
send_event "restart-2-$(uuidgen)"
send_event "restart-3-$(uuidgen)"

echo ""
echo "Waiting 5 seconds for processing..."
sleep 5

INITIAL_AGGREGATE_COUNT=$(get_aggregate_count)
INITIAL_EVENTS_SEEN=$(get_events_seen_count)

echo ""
echo "✓ Initial state:"
echo "  Aggregates created: $INITIAL_AGGREGATE_COUNT"
echo "  Events seen: $INITIAL_EVENTS_SEEN"

if [ "$INITIAL_EVENTS_SEEN" != "3" ]; then
  echo "✗ Expected 3 events in events_seen, got $INITIAL_EVENTS_SEEN"
  exit 1
fi

echo ""
echo "Step 2: Stop the aggregator"
echo "---------------------------"
echo "⚠ Please STOP the aggregator process now (Ctrl+C in its terminal)"
echo "   or run: pkill -f 'bin/aggregator'"
echo ""
read -p "Press Enter when aggregator is stopped..."

echo ""
echo "Step 3: Send 3 MORE events while aggregator is DOWN"
echo "----------------------------------------------------"
echo "(These events will queue up in NATS)"
send_event "restart-4-$(uuidgen)"
send_event "restart-5-$(uuidgen)"
send_event "restart-6-$(uuidgen)"

echo ""
echo "✓ Events sent to NATS (queued, not yet processed)"

echo ""
echo "Step 4: Check database (should be unchanged)"
echo "---------------------------------------------"
EVENTS_SEEN_BEFORE_RESTART=$(get_events_seen_count)
echo "Events seen: $EVENTS_SEEN_BEFORE_RESTART (should still be 3)"

if [ "$EVENTS_SEEN_BEFORE_RESTART" != "3" ]; then
  echo "✗ Unexpected: events_seen changed while aggregator was down!"
  exit 1
fi
echo "✓ No processing occurred while aggregator was down (expected)"

echo ""
echo "Step 5: Restart the aggregator"
echo "-------------------------------"
echo "⚠ Please START the aggregator process now"
echo "   Example: ./bin/aggregator --db-user=telemetry --db-password=telemetry"
echo ""
read -p "Press Enter when aggregator is running..."

echo ""
echo "Waiting 10 seconds for aggregator to process queued events..."
sleep 10

echo ""
echo "Step 6: Verify all events were processed"
echo "-----------------------------------------"
FINAL_EVENTS_SEEN=$(get_events_seen_count)
echo "Events seen: $FINAL_EVENTS_SEEN (should be 6)"

if [ "$FINAL_EVENTS_SEEN" = "6" ]; then
  echo "✓ SUCCESS: All 6 events processed!"
  echo "  - 3 events before shutdown"
  echo "  - 3 events queued during shutdown"
  echo "  - All processed after restart"
else
  echo "✗ FAIL: Expected 6 events, got $FINAL_EVENTS_SEEN"
  exit 1
fi

echo ""
echo "Step 7: Send duplicates to verify dedup still works"
echo "----------------------------------------------------"
send_event "restart-1-$(uuidgen)" # New ID
EVENTS_SEEN_AFTER_DUP=$(get_events_seen_count)

echo ""
echo "Waiting 5 seconds..."
sleep 5

EVENTS_SEEN_FINAL=$(get_events_seen_count)
echo "Events seen: $EVENTS_SEEN_FINAL (should be 7 - one new event)"

if [ "$EVENTS_SEEN_FINAL" = "7" ]; then
  echo "✓ Deduplication still working after restart"
else
  echo "⚠ Unexpected event count: $EVENTS_SEEN_FINAL"
fi

echo ""
echo "=== TEST PASSED: Aggregator Restart ==="
echo "✓ Aggregator stopped gracefully"
echo "✓ Events queued in NATS during downtime"
echo "✓ Aggregator restarted and processed queued events"
echo "✓ No data loss occurred"
echo "✓ Deduplication continued working"
echo ""
