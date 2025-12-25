package ai

import (
	"context"
	"fmt"
)

// Agent is the main AI agent
type Agent struct {
	llm    LLMProvider
	dal    *DataAccessLayer
	config AgentConfig
}

// NewAgent creates a new AI agent
func NewAgent(llm LLMProvider, dal *DataAccessLayer, config AgentConfig) *Agent {
	return &Agent{
		llm:    llm,
		dal:    dal,
		config: config,
	}
}

// Query processes a natural language query
func (a *Agent) Query(ctx context.Context, req QueryRequest) (*QueryResponse, error) {
	// Build context from data
	dataContext, err := a.buildDataContext(ctx, req.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to build data context: %w", err)
	}

	// Build system prompt
	systemPrompt := a.buildSystemPrompt(dataContext)

	// Prepare messages for LLM
	messages := []Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: req.Query,
		},
	}

	// Get LLM response
	response, err := a.llm.Complete(ctx, messages, a.config)
	if err != nil {
		return nil, fmt.Errorf("LLM completion failed: %w", err)
	}

	// Build response
	return &QueryResponse{
		SessionID: req.SessionID,
		Response: ResponseData{
			Text: response,
			Data: dataContext,
		},
		Metadata: ResponseMetadata{
			DataPointsAnalyzed: dataContext["total_measurements"].(int64),
		},
	}, nil
}

func (a *Agent) buildDataContext(ctx context.Context, timeRange TimeRange) (map[string]interface{}, error) {
	// Get overall metrics
	metrics, err := a.dal.GetOverallMetrics(ctx, timeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get overall metrics: %w", err)
	}

	// Get worst performers
	worstPerformers, err := a.dal.CompareClientPerformance(ctx, timeRange, nil, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to get worst performers: %w", err)
	}

	// Build context map
	context := map[string]interface{}{
		"total_clients":      metrics.TotalClients,
		"active_clients":     metrics.ActiveClients,
		"total_targets":      metrics.TotalTargets,
		"avg_latency_p95":    metrics.AvgLatencyP95,
		"avg_throughput_p50": metrics.AvgThroughputP50,
		"total_measurements": metrics.TotalMeasurements,
		"success_rate":       metrics.SuccessRate,
		"error_rate":         metrics.ErrorRate,
		"worst_performers":   worstPerformers,
	}

	return context, nil
}

func (a *Agent) buildSystemPrompt(dataContext map[string]interface{}) string {
	return fmt.Sprintf(`You are an AI assistant specialized in network telemetry analysis.

Current System State:
- Total Clients: %v
- Active Clients: %v
- Total Targets: %v
- Average Latency (P95): %.2f ms
- Average Throughput (P50): %.2f bytes/sec
- Total Measurements: %v
- Success Rate: %.2f%%
- Error Rate: %.2f%%

Data Schema:
- Metrics: DNS latency, TCP latency, TLS latency, TTFB, throughput
- Diagnosis Labels: DNS-bound, Handshake-bound, Server-bound, Throughput-bound
- Aggregation: 1-minute windows with P50, P95 percentiles

Your task is to analyze network performance data and provide clear, actionable insights.
Be specific, cite data points, and provide recommendations when appropriate.`,
		dataContext["total_clients"],
		dataContext["active_clients"],
		dataContext["total_targets"],
		dataContext["avg_latency_p95"],
		dataContext["avg_throughput_p50"],
		dataContext["total_measurements"],
		dataContext["success_rate"].(float64)*100,
		dataContext["error_rate"].(float64)*100,
	)
}
