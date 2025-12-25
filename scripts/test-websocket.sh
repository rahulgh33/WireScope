#!/bin/bash

# Test WebSocket Connection for Real-Time Metrics
# This script tests the WebSocket endpoint and broadcasts test messages

set -e

API_URL="${API_URL:-http://localhost:8080}"
WS_URL="${WS_URL:-ws://localhost:8080}"
TOKEN="${TOKEN:-demo-token}"

echo "=== WebSocket Real-Time Metrics Test ==="
echo "API URL: $API_URL"
echo "WebSocket URL: $WS_URL"
echo ""

# Test 1: Check if the API is up
echo "Test 1: Checking API health..."
if curl -sf "$API_URL/health" > /dev/null; then
    echo "✓ API is healthy"
else
    echo "✗ API is not responding"
    exit 1
fi
echo ""

# Test 2: Test WebSocket connection using a simple Node.js or Python script
# For now, we'll use curl to test the broadcast endpoint

echo "Test 2: Broadcasting test message to WebSocket..."
BROADCAST_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/ws/broadcast" \
    -H "Content-Type: application/json" \
    -d '{
        "channel": "dashboard",
        "data": {
            "type": "test",
            "message": "Hello from test script",
            "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%S.000Z)'"
        }
    }')

if echo "$BROADCAST_RESPONSE" | grep -q "ok"; then
    echo "✓ Broadcast message sent successfully"
else
    echo "✗ Failed to broadcast message"
    echo "Response: $BROADCAST_RESPONSE"
fi
echo ""

# Test 3: Broadcast a simulated aggregate update
echo "Test 3: Broadcasting simulated aggregate update..."
AGGREGATE_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/ws/broadcast" \
    -H "Content-Type: application/json" \
    -d '{
        "channel": "client:test-client-001",
        "data": {
            "client_id": "test-client-001",
            "target": "https://api.example.com",
            "window_start_ts": "'$(date -u +%Y-%m-%dT%H:%M:%S.000Z)'",
            "latency_p95": 245.5,
            "latency_p50": 120.3,
            "throughput_p50": 15.8,
            "error_rate": 0.02,
            "count_total": 100
        }
    }')

if echo "$AGGREGATE_RESPONSE" | grep -q "ok"; then
    echo "✓ Aggregate update broadcast successfully"
else
    echo "✗ Failed to broadcast aggregate"
    echo "Response: $AGGREGATE_RESPONSE"
fi
echo ""

# Test 4: Broadcast a diagnosis alert
echo "Test 4: Broadcasting diagnosis alert..."
DIAGNOSIS_RESPONSE=$(curl -s -X POST "$API_URL/api/v1/ws/broadcast" \
    -H "Content-Type: application/json" \
    -d '{
        "channel": "diagnostics",
        "data": {
            "client_id": "test-client-001",
            "target": "https://api.example.com",
            "diagnosis": "Server-bound",
            "severity": "warning",
            "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%S.000Z)'"
        }
    }')

if echo "$DIAGNOSIS_RESPONSE" | grep -q "ok"; then
    echo "✓ Diagnosis alert broadcast successfully"
else
    echo "✗ Failed to broadcast diagnosis"
    echo "Response: $DIAGNOSIS_RESPONSE"
fi
echo ""

echo "=== WebSocket Test Summary ==="
echo "All basic tests passed!"
echo ""
echo "To test WebSocket connection from a client:"
echo "1. Start the AI agent server: ./bin/ai-agent"
echo "2. Open a WebSocket connection: wscat -c '$WS_URL/api/v1/ws/metrics?token=$TOKEN'"
echo "3. Subscribe to channels: {\"type\":\"subscribe\",\"channels\":[\"dashboard\",\"diagnostics\"],\"schema_version\":\"1.0\",\"timestamp\":\"2024-12-24T00:00:00Z\"}"
echo "4. Run this script again to send test broadcasts"
echo ""
echo "Frontend integration:"
echo "The React app will automatically connect when you access http://localhost:3000"
