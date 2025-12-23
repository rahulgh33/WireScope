#!/bin/bash

# Test script for the ingest API
# This script tests the ingest API by sending a sample telemetry event

set -e

# Configuration
INGEST_URL="${INGEST_URL:-http://localhost:8090/events}"
API_TOKEN="${API_TOKEN:-test-token-123}"

# Sample telemetry event (matching the TelemetryEvent schema)
EVENT_DATA=$(cat <<EOF
{
  "event_id": "$(uuidgen | tr '[:upper:]' '[:lower:]')",
  "client_id": "probe-test-client-123",
  "ts_ms": $(date +%s)000,
  "schema_version": "1.0",
  "target": "https://example.com",
  "network_context": {
    "interface_type": "ethernet",
    "vpn_enabled": false
  },
  "timings": {
    "dns_ms": 5.2,
    "tcp_ms": 10.5,
    "tls_ms": 15.8,
    "http_ttfb_ms": 50.3
  },
  "throughput_kbps": 12500.5
}
EOF
)

echo "Testing Ingest API at: $INGEST_URL"
echo "Event data:"
echo "$EVENT_DATA" | jq '.'

echo ""
echo "Sending request..."

# Send the request
RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
  -X POST "$INGEST_URL" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $API_TOKEN" \
  -d "$EVENT_DATA")

# Extract HTTP status
HTTP_STATUS=$(echo "$RESPONSE" | grep "HTTP_STATUS:" | cut -d: -f2)
BODY=$(echo "$RESPONSE" | sed '/HTTP_STATUS:/d')

echo "HTTP Status: $HTTP_STATUS"
echo "Response:"
echo "$BODY" | jq '.' 2>/dev/null || echo "$BODY"

if [ "$HTTP_STATUS" -eq 202 ]; then
  echo ""
  echo "✓ Success! Event was accepted by the ingest API"
  exit 0
else
  echo ""
  echo "✗ Failed! Expected HTTP 202, got $HTTP_STATUS"
  exit 1
fi
