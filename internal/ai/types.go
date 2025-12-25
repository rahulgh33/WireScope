package ai

import "time"

// TimeRange represents a time range for queries
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// QueryRequest represents an AI query request
type QueryRequest struct {
	Query     string                 `json:"query"`
	SessionID string                 `json:"session_id,omitempty"`
	TimeRange TimeRange              `json:"time_range,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// QueryResponse represents an AI query response
type QueryResponse struct {
	SessionID string           `json:"session_id"`
	Response  ResponseData     `json:"response"`
	Metadata  ResponseMetadata `json:"metadata"`
}

// ResponseData contains the actual response content
type ResponseData struct {
	Text string                 `json:"text"`
	Data map[string]interface{} `json:"data,omitempty"`
}

// ResponseMetadata contains query execution metadata
type ResponseMetadata struct {
	QueryTimeMs        int64 `json:"query_time_ms"`
	DataPointsAnalyzed int64 `json:"data_points_analyzed,omitempty"`
	TokensUsed         int   `json:"tokens_used,omitempty"`
}

// AgentConfig configures the AI agent
type AgentConfig struct {
	Provider         string
	Model            string
	APIKey           string
	Temperature      float64
	MaxTokens        int
	MaxContextTokens int
	EnableCaching    bool
}

// TimeSeriesPoint represents a single data point in a time series
type TimeSeriesPoint struct {
	Timestamp       time.Time
	ClientID        string
	Target          string
	DNSP95          float64
	TCPP95          float64
	TLSP95          float64
	TTFBP95         float64
	TotalLatencyP95 float64
	ThroughputP50   float64
	CountTotal      int64
	CountError      int64
	ErrorRate       float64
	DiagnosisLabel  *string
}

// ClientPerformance represents aggregated performance metrics for a client
type ClientPerformance struct {
	ClientID          string
	Target            string
	AvgLatencyP95     float64
	AvgThroughputP50  float64
	ErrorRate         float64
	TotalMeasurements int64
	PrimaryIssue      *string
}

// OverallMetrics represents summary statistics across all data
type OverallMetrics struct {
	TimeRange         TimeRange
	TotalClients      int64
	ActiveClients     int64
	TotalTargets      int64
	AvgLatencyP95     float64
	AvgThroughputP50  float64
	TotalMeasurements int64
	SuccessRate       float64
	ErrorRate         float64
}
