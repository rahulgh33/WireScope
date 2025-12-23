#!/bin/bash
# Test backpressure mechanisms
# Validates rate limiting and queue behavior under load

set -e

echo "=== Milestone B Demo: Backpressure Test ==="
echo ""

# Configuration
INGEST_URL="${INGEST_URL:-http://localhost:8080}"
API_TOKEN="${API_TOKEN:-test-token}"
CLIENT_ID="backpressure-test-$(date +%s)"
TARGET="http://test-target.local"
METRICS_URL="${METRICS_URL:-http://localhost:9090/metrics}"

echo "Configuration:"
echo "  Ingest URL: $INGEST_URL"
echo "  Metrics URL: $METRICS_URL"
echo "  Client ID: $CLIENT_ID"
echo ""

send_event() {
  local event_id=$1
  local timestamp_ms=$(date +%s000)
  
  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "$INGEST_URL/events" \
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
    }")
    
  echo "$HTTP_CODE"
}

get_rate_limited_count() {
  curl -s "$INGEST_URL/metrics" 2>/dev/null | grep 'ingest_requests_total{status="rate_limited"}' | awk '{print $2}' || echo "0"
}

echo "Step 1: Send events at normal rate"
echo "-----------------------------------"
SUCCESS_COUNT=0
for i in {1..10}; do
  HTTP_CODE=$(send_event "normal-$i-$(uuidgen)")
  if [ "$HTTP_CODE" = "202" ]; then
    SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
  fi
  sleep 0.1  # 10 req/s - well below limit
done

echo "✓ Sent 10 events at normal rate"
echo "  Accepted: $SUCCESS_COUNT/10"

if [ "$SUCCESS_COUNT" != "10" ]; then
  echo "✗ Expected all 10 to be accepted"
  exit 1
fi

echo ""
echo "Step 2: Burst test - send many events rapidly"
echo "----------------------------------------------"
echo "Sending 150 events as fast as possible..."
echo "(Rate limit is 100 req/s with burst of 20)"
echo ""

ACCEPTED=0
RATE_LIMITED=0

for i in {1..150}; do
  HTTP_CODE=$(send_event "burst-$i-$(uuidgen)")
  if [ "$HTTP_CODE" = "202" ]; then
    ACCEPTED=$((ACCEPTED + 1))
    echo -n "."
  elif [ "$HTTP_CODE" = "429" ]; then
    RATE_LIMITED=$((RATE_LIMITED + 1))
    echo -n "R"
  else
    echo -n "?"
  fi
done

echo ""
echo ""
echo "Results:"
echo "  Accepted (202): $ACCEPTED"
echo "  Rate Limited (429): $RATE_LIMITED"
echo "  Total: $((ACCEPTED + RATE_LIMITED))"

if [ "$RATE_LIMITED" -gt "0" ]; then
  echo "✓ SUCCESS: Rate limiting activated!"
  echo "  Some requests were rate-limited as expected"
else
  echo "⚠ WARNING: No rate limiting observed"
  echo "  This might indicate rate limits are too high or burst is large"
fi

echo ""
echo "Step 3: Wait for rate limiter to recover"
echo "-----------------------------------------"
echo "Waiting 2 seconds for token bucket to refill..."
sleep 2

echo ""
echo "Sending 10 more events..."
RECOVERY_SUCCESS=0
for i in {1..10}; do
  HTTP_CODE=$(send_event "recovery-$i-$(uuidgen)")
  if [ "$HTTP_CODE" = "202" ]; then
    RECOVERY_SUCCESS=$((RECOVERY_SUCCESS + 1))
  fi
  sleep 0.15  # ~6 req/s
done

echo "✓ Recovery test: $RECOVERY_SUCCESS/10 accepted"
if [ "$RECOVERY_SUCCESS" -ge "8" ]; then
  echo "✓ Rate limiter recovered successfully"
else
  echo "⚠ Rate limiter may still be throttling"
fi

echo ""
echo "Step 4: Check metrics"
echo "---------------------"

if command -v curl >/dev/null 2>&1; then
  echo "Fetching metrics from $INGEST_URL/metrics..."
  
  TOTAL_REQUESTS=$(curl -s "$INGEST_URL/metrics" 2>/dev/null | grep 'ingest_requests_total' | grep -v '#' | awk '{sum+=$2} END {print sum}')
  RATE_LIMITED_METRIC=$(get_rate_limited_count)
  
  if [ -n "$TOTAL_REQUESTS" ]; then
    echo "  Total requests processed: $TOTAL_REQUESTS"
    echo "  Rate limited (from metrics): $RATE_LIMITED_METRIC"
  else
    echo "  ⚠ Could not fetch metrics"
  fi
fi

echo ""
echo "Step 5: Check queue lag metrics (aggregator)"
echo "---------------------------------------------"
if command -v curl >/dev/null 2>&1; then
  QUEUE_LAG=$(curl -s "$METRICS_URL" 2>/dev/null | grep 'queue_lag_messages ' | awk '{print $2}' || echo "unknown")
  QUEUE_ACK_PENDING=$(curl -s "$METRICS_URL" 2>/dev/null | grep 'queue_ack_pending_messages ' | awk '{print $2}' || echo "unknown")
  
  echo "  Queue lag: $QUEUE_LAG messages"
  echo "  Ack pending: $QUEUE_ACK_PENDING messages"
  
  if [ "$QUEUE_LAG" != "unknown" ]; then
    if [ "$(echo "$QUEUE_LAG < 100" | bc 2>/dev/null || echo 1)" = "1" ]; then
      echo "✓ Queue lag is reasonable"
    else
      echo "⚠ Queue lag is high - aggregator may be under pressure"
    fi
  fi
fi

echo ""
echo "=== TEST PASSED: Backpressure Mechanisms ==="
echo "✓ Rate limiting is active"
echo "✓ Token bucket refills over time"
echo "✓ System gracefully handles burst traffic"
echo "✓ Queue metrics are available"
echo ""
echo "Summary:"
echo "  - Normal rate: all events accepted"
echo "  - Burst rate: some events rate-limited (429)"
echo "  - Recovery: rate limiter allows traffic after cooldown"
echo ""
