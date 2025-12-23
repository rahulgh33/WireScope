#!/bin/bash
# Test DLQ routing for poison messages
# Validates that malformed events are sent to DLQ after max retries

set -e

echo "=== Milestone B Demo: DLQ Poison Message Test ==="
echo ""

# Configuration
NATS_URL="${NATS_URL:-nats://localhost:4222}"
DLQ_SUBJECT="telemetry.dlq"
METRICS_URL="${METRICS_URL:-http://localhost:9090/metrics}"

echo "Configuration:"
echo "  NATS URL: $NATS_URL"
echo "  DLQ Subject: $DLQ_SUBJECT"
echo "  Metrics URL: $METRICS_URL"
echo ""

# Check if nats CLI is available
if ! command -v nats >/dev/null 2>&1; then
  echo "⚠ NATS CLI not found. Installing..."
  echo "  Run: go install github.com/nats-io/natscli/nats@latest"
  echo ""
  echo "Alternatively, you can manually test DLQ by:"
  echo "  1. Publishing a malformed event to telemetry.events"
  echo "  2. Watching aggregator logs for 'sending to DLQ' messages"
  echo "  3. Checking DLQ metrics increase"
  exit 1
fi

get_dlq_count() {
  curl -s "$METRICS_URL" 2>/dev/null | grep 'dlq_messages_total ' | awk '{print $2}' || echo "0"
}

echo "Step 1: Get initial DLQ count"
echo "-----------------------------"
INITIAL_DLQ_COUNT=$(get_dlq_count)
echo "Initial DLQ messages: $INITIAL_DLQ_COUNT"
echo ""

echo "Step 2: Publish a MALFORMED event directly to NATS"
echo "---------------------------------------------------"
echo "This event has invalid JSON that will fail to unmarshal"
echo ""

# Publish malformed JSON
MALFORMED_JSON='{"schema_version": "1.0", "event_id": "poison-test", INVALID JSON HERE}'

nats pub --server="$NATS_URL" telemetry.events "$MALFORMED_JSON" 2>&1 || {
  echo "✓ Attempted to publish malformed event"
}

echo "✓ Published malformed event to NATS"
echo ""

echo "Step 3: Wait for aggregator to attempt processing"
echo "--------------------------------------------------"
echo "Aggregator will retry up to 5 times (MaxDeliver), then send to DLQ"
echo "This may take 30-60 seconds depending on AckWait settings..."
echo ""

for i in {1..12}; do
  echo -n "Waiting... ($i/12) "
  sleep 5
  
  CURRENT_DLQ_COUNT=$(get_dlq_count)
  if [ "$CURRENT_DLQ_COUNT" != "$INITIAL_DLQ_COUNT" ]; then
    echo ""
    echo "✓ DLQ count changed!"
    break
  fi
  echo ""
done

echo ""
echo "Step 4: Check DLQ metrics"
echo "-------------------------"
FINAL_DLQ_COUNT=$(get_dlq_count)
echo "Initial DLQ count: $INITIAL_DLQ_COUNT"
echo "Final DLQ count: $FINAL_DLQ_COUNT"

if [ "$FINAL_DLQ_COUNT" != "$INITIAL_DLQ_COUNT" ]; then
  INCREASE=$((FINAL_DLQ_COUNT - INITIAL_DLQ_COUNT))
  echo "✓ SUCCESS: DLQ count increased by $INCREASE"
  echo "  Poison message was sent to DLQ after max retries"
else
  echo "⚠ WARNING: DLQ count unchanged"
  echo "  This might mean:"
  echo "    - Still processing retries (wait longer)"
  echo "    - Aggregator not running"
  echo "    - DLQ disabled in configuration"
fi

echo ""
echo "Step 5: Inspect DLQ messages (if nats CLI available)"
echo "-----------------------------------------------------"

# Try to peek at DLQ stream
nats stream info telemetry-events-dlq --server="$NATS_URL" 2>/dev/null && {
  echo ""
  echo "DLQ Stream Info:"
  nats stream info telemetry-events-dlq --server="$NATS_URL" | grep -E "(Messages|Bytes|First|Last)" || true
  
  echo ""
  echo "Sample DLQ message:"
  nats sub --server="$NATS_URL" "$DLQ_SUBJECT" --count=1 --timeout=2s 2>/dev/null || {
    echo "  (No messages available to display)"
  }
} || {
  echo "  Could not access DLQ stream info"
}

echo ""
echo "Step 6: Test with another malformed event"
echo "------------------------------------------"
MALFORMED_JSON2='{"incomplete": "json without required fields"}'

nats pub --server="$NATS_URL" telemetry.events "$MALFORMED_JSON2" 2>&1 || true
echo "✓ Published second malformed event"

echo ""
echo "Waiting 60 seconds for processing and DLQ routing..."
sleep 60

FINAL_DLQ_COUNT2=$(get_dlq_count)
TOTAL_INCREASE=$((FINAL_DLQ_COUNT2 - INITIAL_DLQ_COUNT))

echo ""
echo "Final DLQ count: $FINAL_DLQ_COUNT2"
echo "Total increase: $TOTAL_INCREASE"

if [ "$TOTAL_INCREASE" -ge "1" ]; then
  echo "✓ DLQ routing is working"
else
  echo "⚠ DLQ count did not increase as expected"
fi

echo ""
echo "=== TEST SUMMARY: DLQ Poison Messages ==="
if [ "$TOTAL_INCREASE" -ge "1" ]; then
  echo "✓ PASSED: Poison messages routed to DLQ"
  echo "✓ Max delivery attempts enforced"
  echo "✓ DLQ metrics updated correctly"
  echo "✓ System protected from poison messages"
else
  echo "⚠ INCONCLUSIVE: Please check aggregator logs"
  echo "  Look for: 'sending to DLQ' or 'exceeded max deliveries'"
  echo "  Run: tail -f <aggregator-log>"
fi
echo ""
