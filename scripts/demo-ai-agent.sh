#!/bin/bash

# Quick Demo of AI Agent
# This script demonstrates the AI agent capabilities

echo "╔═════════════════════════════════════════════════════╗"
echo "║     AI Agent Quick Demo                             ║"
echo "╚═════════════════════════════════════════════════════╝"
echo ""

# Check if server is running
if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "⚠️  AI Agent server is not running!"
    echo ""
    echo "Start the server first:"
    echo "  export AI_PROVIDER=mock"
    echo "  ./bin/ai-agent"
    echo ""
    exit 1
fi

echo "✅ Server is running"
echo ""

# Test capabilities
echo "📋 Querying AI capabilities..."
curl -s http://localhost:8080/api/v1/ai/capabilities \
    -H "Authorization: Bearer demo-key" | jq '.'
echo ""

# Test query
echo "🤖 Asking: 'Which clients had the worst performance?'"
echo ""

RESPONSE=$(curl -s http://localhost:8080/api/v1/ai/query \
    -H "Authorization: Bearer demo-key" \
    -H "Content-Type: application/json" \
    -d '{
        "query": "Which clients had the worst performance today?",
        "time_range": {
            "start": "2025-12-24T00:00:00Z",
            "end": "2025-12-24T23:59:59Z"
        }
    }')

echo "$RESPONSE" | jq -r '.response.text'
echo ""

QUERY_TIME=$(echo "$RESPONSE" | jq -r '.metadata.query_time_ms')
echo "⏱️  Query completed in ${QUERY_TIME}ms"
echo ""

echo "✅ Demo complete!"
echo ""
echo "Try the interactive CLI:"
echo "  ./bin/telemetry-ai"
echo ""
