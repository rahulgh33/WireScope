package ai

import (
	"context"
	"fmt"
	"strings"
)

// LLMProvider is the interface for LLM integrations
type LLMProvider interface {
	Complete(ctx context.Context, messages []Message, config AgentConfig) (string, error)
}

// Message represents a chat message
type Message struct {
	Role    string
	Content string
}

// MockLLMProvider provides mock responses for testing
type MockLLMProvider struct{}

// NewMockLLMProvider creates a new mock LLM provider
func NewMockLLMProvider() *MockLLMProvider {
	return &MockLLMProvider{}
}

// Complete generates a mock response
func (m *MockLLMProvider) Complete(ctx context.Context, messages []Message, config AgentConfig) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("no messages provided")
	}

	lastMessage := messages[len(messages)-1]
	query := strings.ToLower(lastMessage.Content)

	// Pattern matching for common queries
	if strings.Contains(query, "worst") && strings.Contains(query, "performance") {
		return m.mockWorstPerformersResponse(), nil
	}
	if strings.Contains(query, "trend") {
		return m.mockTrendResponse(), nil
	}
	if strings.Contains(query, "anomal") {
		return m.mockAnomalyResponse(), nil
	}
	if strings.Contains(query, "compare") {
		return m.mockComparisonResponse(), nil
	}

	// Default response
	return "I understand you're asking about network telemetry data. I can help you analyze performance metrics, identify issues, and compare clients. Please provide more specific details about what you'd like to know.", nil
}

func (m *MockLLMProvider) mockWorstPerformersResponse() string {
	return `Based on the available data, here are the clients with the worst performance:

1. **client-001**: Average latency P95 of 450ms, primarily DNS-bound issues
2. **client-002**: Average latency P95 of 380ms, server-bound issues detected
3. **client-003**: Average latency P95 of 350ms, mixed DNS and handshake issues

The main contributing factors are:
- DNS resolution delays (40% of issues)
- Server response times (35% of issues)
- TLS handshake overhead (25% of issues)

Recommendation: Focus on DNS infrastructure optimization for the top performers.`
}

func (m *MockLLMProvider) mockTrendResponse() string {
	return `Analyzing recent trends:

**Latency Trends:**
- Overall latency has increased by 15% over the past week
- DNS latency is the primary contributor (+25% increase)
- TLS and TTFB remain relatively stable

**Throughput Trends:**
- Average throughput decreased by 8%
- Peak hours (9am-5pm) show more significant degradation

**Pattern Analysis:**
- Daily patterns show performance degradation during business hours
- Weekend performance is 20% better than weekdays

Recommendation: Consider capacity upgrades for peak hour handling.`
}

func (m *MockLLMProvider) mockAnomalyResponse() string {
	return `Anomaly Detection Results:

Found 12 significant anomalies in the specified time range:

**High Severity:**
- 3 incidents with latency >3σ above baseline
- All occurred between 2:00 PM - 4:00 PM
- Affected targets: api.example.com, cdn.example.com

**Medium Severity:**
- 7 incidents with latency 2-3σ above baseline
- Distributed across multiple targets
- Primarily DNS-bound issues

**Root Cause:**
The anomalies correlate with a DNS resolver issue that was resolved at 4:15 PM.

Recommendation: Monitor DNS resolver stability and consider backup DNS configuration.`
}

func (m *MockLLMProvider) mockComparisonResponse() string {
	return `Client Comparison Analysis:

**Top Performers:**
- client-010: 95ms average latency, 0.1% error rate
- client-015: 105ms average latency, 0.2% error rate
- client-020: 110ms average latency, 0.15% error rate

**Bottom Performers:**
- client-001: 450ms average latency, 2.5% error rate
- client-002: 380ms average latency, 1.8% error rate
- client-003: 350ms average latency, 1.5% error rate

**Key Differences:**
- Top performers use closer geographic locations to targets
- Bottom performers show consistent DNS resolution issues
- Network path quality varies significantly

Recommendation: Investigate network paths and DNS configuration for underperforming clients.`
}
