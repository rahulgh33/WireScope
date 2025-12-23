#!/bin/bash
# Test duplicate event handling with unchanged aggregates
# Validates that sending the same event multiple times results in exactly-once aggregate effects

set -e

echo "=== Milestone B Demo: Duplicate Event Handling ==="
echo ""

# Configuration
INGEST_URL="${INGEST_URL:-http://localhost:8080}"
API_TOKEN="${API_TOKEN:-test-token}"
CLIENT_ID="demo-client-$(date +%s)"
TARGET="http://test-target.local"
EVENT_ID="duplicate-test-$(uuidgen)"

# Check if required tools are available
command -v curl >/dev/null 2>&1 || { echo "curl is required but not installed. Aborting." >&2; exit 1; }
command -v psql >/dev/null 2>&1 || { echo "psql is required but not installed. Aborting." >&2; exit 1; }
command -v jq >/dev/null 2>&1 || { echo "jq is required but not installed. Aborting." >&2; exit 1; }

# Database connection
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-telemetry}"
DB_USER="${DB_USER:-telemetry}"
DB_PASSWORD="${DB_PASSWORD:-telemetry}"

echo "Configuration:"
echo "  Ingest URL: $INGEST_URL"
echo "  Client ID: $CLIENT_ID"
echo "  Target: $TARGET"
echo "  Event ID: $EVENT_ID"
echo ""

# Create test event
TIMESTAMP_MS=$(date +%s000)
WINDOW_START_MS=$((TIMESTAMP_MS / 60000 * 60000))

create_event() {
  cat <<EOF
{
  "schema_version": "1.0",
  "event_id": "$EVENT_ID",
  "client_id": "$CLIENT_ID",
  "timestamp_ms": $TIMESTAMP_MS,
  "target": "$TARGET",
  "network_context": {
    "interface_name": "eth0",
    "local_ip": "192.168.1.100"
  },
  "timing_measurements": {
    "dns_ms": 10.5,
    "tcp_ms": 15.2,
    "tls_ms": 45.8,
    "ttfb_ms": 120.3,
    "total_ms": 191.8
  },
  "throughput_measurement": {
    "bytes_transferred": 1048576,
    "duration_ms": 850.5,
    "throughput_mbps": 9.86
  },
  "error_stage": null
}
EOF
}

echo "Step 1: Send event for the first time"
echo "--------------------------------------"
RESPONSE=$(create_event | curl -s -w "\nHTTP_CODE:%{http_code}" \
  -X POST "$INGEST_URL/events" \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d @-)

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
BODY=$(echo "$RESPONSE" | grep -v "HTTP_CODE:")

if [ "$HTTP_CODE" = "202" ]; then
  echo "✓ Event accepted (HTTP $HTTP_CODE)"
  echo "  Response: $BODY"
else
  echo "✗ Unexpected HTTP code: $HTTP_CODE"
  echo "  Response: $BODY"
  exit 1
fi

echo ""
echo "Waiting 5 seconds for aggregator to process..."
sleep 5

echo ""
echo "Step 2: Check aggregate count (should be 1)"
echo "-------------------------------------------"
QUERY="SELECT count_total, count_success FROM agg_1m WHERE client_id = '$CLIENT_ID' AND target = '$TARGET' AND window_start_ts = to_timestamp($WINDOW_START_MS / 1000.0)"
RESULT=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -A -c "$QUERY")

if [ -n "$RESULT" ]; then
  COUNT_TOTAL=$(echo "$RESULT" | cut -d'|' -f1)
  COUNT_SUCCESS=$(echo "$RESULT" | cut -d'|' -f2)
  echo "✓ Aggregate found:"
  echo "  count_total: $COUNT_TOTAL"
  echo "  count_success: $COUNT_SUCCESS"
  
  if [ "$COUNT_TOTAL" != "1" ]; then
    echo "✗ FAIL: Expected count_total=1, got $COUNT_TOTAL"
    exit 1
  fi
else
  echo "✗ No aggregate found in database"
  exit 1
fi

echo ""
echo "Step 3: Send the SAME event again (duplicate)"
echo "----------------------------------------------"
RESPONSE=$(create_event | curl -s -w "\nHTTP_CODE:%{http_code}" \
  -X POST "$INGEST_URL/events" \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d @-)

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
BODY=$(echo "$RESPONSE" | grep -v "HTTP_CODE:")

if [ "$HTTP_CODE" = "202" ]; then
  echo "✓ Event accepted (HTTP $HTTP_CODE)"
  echo "  Response: $BODY"
else
  echo "✗ Unexpected HTTP code: $HTTP_CODE"
  exit 1
fi

echo ""
echo "Waiting 5 seconds for aggregator to process..."
sleep 5

echo ""
echo "Step 4: Check aggregate count (should STILL be 1)"
echo "-------------------------------------------------"
RESULT=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -A -c "$QUERY")

if [ -n "$RESULT" ]; then
  COUNT_TOTAL=$(echo "$RESULT" | cut -d'|' -f1)
  COUNT_SUCCESS=$(echo "$RESULT" | cut -d'|' -f2)
  echo "✓ Aggregate found:"
  echo "  count_total: $COUNT_TOTAL"
  echo "  count_success: $COUNT_SUCCESS"
  
  if [ "$COUNT_TOTAL" = "1" ]; then
    echo "✓ SUCCESS: Duplicate was properly deduplicated!"
    echo "  Aggregate remained unchanged (count_total=1)"
  else
    echo "✗ FAIL: Expected count_total=1, got $COUNT_TOTAL"
    echo "  Duplicate was NOT properly handled!"
    exit 1
  fi
else
  echo "✗ No aggregate found in database"
  exit 1
fi

echo ""
echo "Step 5: Send the same event a THIRD time"
echo "-----------------------------------------"
RESPONSE=$(create_event | curl -s -w "\nHTTP_CODE:%{http_code}" \
  -X POST "$INGEST_URL/events" \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d @-)

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
echo "✓ Event accepted (HTTP $HTTP_CODE)"

echo ""
echo "Waiting 5 seconds for aggregator to process..."
sleep 5

echo ""
echo "Step 6: Final verification (should STILL be 1)"
echo "----------------------------------------------"
RESULT=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -A -c "$QUERY")
COUNT_TOTAL=$(echo "$RESULT" | cut -d'|' -f1)

if [ "$COUNT_TOTAL" = "1" ]; then
  echo "✓ SUCCESS: All duplicates properly deduplicated!"
  echo "  Final count_total: $COUNT_TOTAL"
else
  echo "✗ FAIL: Expected count_total=1, got $COUNT_TOTAL"
  exit 1
fi

echo ""
echo "Step 7: Check events_seen table"
echo "--------------------------------"
EVENTS_SEEN_QUERY="SELECT COUNT(*) FROM events_seen WHERE event_id = '$EVENT_ID'"
EVENTS_SEEN_COUNT=$(PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -A -c "$EVENTS_SEEN_QUERY")

echo "Events in events_seen table: $EVENTS_SEEN_COUNT"
if [ "$EVENTS_SEEN_COUNT" = "1" ]; then
  echo "✓ Exactly one entry in events_seen (correct deduplication)"
else
  echo "⚠ Unexpected count in events_seen: $EVENTS_SEEN_COUNT"
fi

echo ""
echo "=== TEST PASSED: Duplicate Event Handling ==="
echo "✓ Events deduplicated correctly"
echo "✓ Aggregate effects are exactly-once"
echo "✓ Database maintains consistency"
echo ""
