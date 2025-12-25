#!/bin/bash

# AI Agent Test Script
# Tests the AI agent implementation

set -e

echo "================================"
echo "AI Agent Implementation Test"
echo "================================"
echo ""

# Check binaries exist
echo "1. Checking binaries..."
if [ ! -f bin/ai-agent ]; then
    echo "❌ ai-agent binary not found"
    exit 1
fi
if [ ! -f bin/telemetry-ai ]; then
    echo "❌ telemetry-ai binary not found"
    exit 1
fi
echo "✅ Binaries found"
echo ""

# Check Go files
echo "2. Checking implementation files..."
FILES=(
    "internal/ai/types.go"
    "internal/ai/data_access.go"
    "internal/ai/llm_mock.go"
    "internal/ai/llm_openai.go"
    "internal/ai/agent.go"
    "internal/ai/session.go"
    "cmd/ai-agent/main.go"
    "cmd/telemetry-ai/main.go"
)

for file in "${FILES[@]}"; do
    if [ ! -f "$file" ]; then
        echo "❌ Missing file: $file"
        exit 1
    fi
done
echo "✅ All implementation files present"
echo ""

# Test compilation
echo "3. Testing compilation..."
if make build > /dev/null 2>&1; then
    echo "✅ Build successful"
else
    echo "❌ Build failed"
    exit 1
fi
echo ""

# Check binary sizes
echo "4. Binary sizes:"
ls -lh bin/ai-agent | awk '{print "   ai-agent:     " $5}'
ls -lh bin/telemetry-ai | awk '{print "   telemetry-ai: " $5}'
echo ""

# Test CLI help
echo "5. Testing CLI..."
if ./bin/telemetry-ai --help > /dev/null 2>&1; then
    echo "✅ CLI help works"
else
    echo "❌ CLI help failed"
    exit 1
fi
echo ""

# Summary
echo "================================"
echo "✅ All tests passed!"
echo "================================"
echo ""
echo "To start the AI agent server:"
echo "  export AI_PROVIDER=mock  # Use mock provider (no API key needed)"
echo "  ./bin/ai-agent"
echo ""
echo "To use the CLI:"
echo "  ./bin/telemetry-ai"
echo ""
echo "To use with OpenAI:"
echo "  export AI_PROVIDER=openai"
echo "  export OPENAI_API_KEY=your-key"
echo "  ./bin/ai-agent"
echo ""
