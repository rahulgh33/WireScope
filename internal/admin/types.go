package admin

import "time"

// ProbeConfig represents probe agent configuration
type ProbeConfig struct {
	ID          string     `json:"id"`
	ClientID    string     `json:"client_id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Targets     []string   `json:"targets"`
	Interval    int        `json:"interval"` // seconds
	Enabled     bool       `json:"enabled"`
	APIEndpoint string     `json:"api_endpoint"`
	APIToken    string     `json:"api_token,omitempty"` // Masked in responses
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastSeen    *time.Time `json:"last_seen,omitempty"`
}

// APIToken represents an ingest API token
type APIToken struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Token       string     `json:"token,omitempty"` // Full token only on creation
	TokenPrefix string     `json:"token_prefix"`    // e.g., "tok_abc..."
	Enabled     bool       `json:"enabled"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	CreatedBy   string     `json:"created_by"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	UsageCount  int64      `json:"usage_count"`
}

// User represents a system user
type User struct {
	ID        string     `json:"id"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	FullName  string     `json:"full_name,omitempty"`
	Role      string     `json:"role"` // admin, viewer, operator
	Enabled   bool       `json:"enabled"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	LastLogin *time.Time `json:"last_login,omitempty"`
}

// SystemSettings represents global system configuration
type SystemSettings struct {
	EventsRetentionDays     int       `json:"events_retention_days"`
	AggregatesRetentionDays int       `json:"aggregates_retention_days"`
	LatencyThresholdMs      float64   `json:"latency_threshold_ms"`
	ThroughputThresholdMb   float64   `json:"throughput_threshold_mb"`
	ErrorRateThreshold      float64   `json:"error_rate_threshold"`
	DNSBoundThreshold       float64   `json:"dns_bound_threshold"`
	HandshakeBoundSigma     float64   `json:"handshake_bound_sigma"`
	ServerBoundSigma        float64   `json:"server_bound_sigma"`
	ThroughputDropThreshold float64   `json:"throughput_drop_threshold"`
	MaxProbesPerClient      int       `json:"max_probes_per_client"`
	MaxTargetsPerProbe      int       `json:"max_targets_per_probe"`
	MaxSamplesPerWindow     int       `json:"max_samples_per_window"`
	RateLimitPerClientRPS   int       `json:"rate_limit_per_client_rps"`
	UpdatedAt               time.Time `json:"updated_at"`
	UpdatedBy               string    `json:"updated_by"`
}

// DatabaseStats represents database health metrics
type DatabaseStats struct {
	ActiveConnections   int        `json:"active_connections"`
	IdleConnections     int        `json:"idle_connections"`
	MaxConnections      int        `json:"max_connections"`
	EventsTableSize     int64      `json:"events_table_size_bytes"`
	AggregatesTableSize int64      `json:"aggregates_table_size_bytes"`
	TotalDatabaseSize   int64      `json:"total_database_size_bytes"`
	EventsRowCount      int64      `json:"events_row_count"`
	AggregatesRowCount  int64      `json:"aggregates_row_count"`
	AvgQueryTimeMs      float64    `json:"avg_query_time_ms"`
	SlowQueriesCount    int64      `json:"slow_queries_count"`
	LastVacuum          *time.Time `json:"last_vacuum,omitempty"`
	LastAnalyze         *time.Time `json:"last_analyze,omitempty"`
	NextCleanup         *time.Time `json:"next_cleanup,omitempty"`
	Timestamp           time.Time  `json:"timestamp"`
}

// SystemHealth represents overall system health
type SystemHealth struct {
	Status            string          `json:"status"`
	Timestamp         time.Time       `json:"timestamp"`
	Database          ComponentHealth `json:"database"`
	NATS              ComponentHealth `json:"nats"`
	Aggregator        ComponentHealth `json:"aggregator"`
	IngestAPI         ComponentHealth `json:"ingest_api"`
	WebSocket         ComponentHealth `json:"websocket"`
	ActiveProbes      int             `json:"active_probes"`
	ActiveClients     int             `json:"active_clients"`
	EventsPerSecond   float64         `json:"events_per_second"`
	QueueLag          int64           `json:"queue_lag"`
	ErrorRate         float64         `json:"error_rate"`
	AvgProcessingTime float64         `json:"avg_processing_time_ms"`
}

// ComponentHealth represents health of a single component
type ComponentHealth struct {
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Latency   float64   `json:"latency_ms,omitempty"`
	ErrorRate float64   `json:"error_rate,omitempty"`
	LastCheck time.Time `json:"last_check"`
}

// MaintenanceTask represents a database maintenance operation
type MaintenanceTask struct {
	ID           string     `json:"id"`
	Type         string     `json:"type"`
	Description  string     `json:"description"`
	Status       string     `json:"status"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Error        string     `json:"error,omitempty"`
	RowsAffected int64      `json:"rows_affected,omitempty"`
	DurationMs   int64      `json:"duration_ms,omitempty"`
}

// Request types
type CreateProbeRequest struct {
	ClientID    string   `json:"client_id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Targets     []string `json:"targets"`
	Interval    int      `json:"interval"`
	Enabled     bool     `json:"enabled"`
}

type UpdateProbeRequest struct {
	Name        *string   `json:"name,omitempty"`
	Description *string   `json:"description,omitempty"`
	Targets     *[]string `json:"targets,omitempty"`
	Interval    *int      `json:"interval,omitempty"`
	Enabled     *bool     `json:"enabled,omitempty"`
}

type CreateAPITokenRequest struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	FullName string `json:"full_name,omitempty"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type UpdateUserRequest struct {
	Email    *string `json:"email,omitempty"`
	FullName *string `json:"full_name,omitempty"`
	Role     *string `json:"role,omitempty"`
	Enabled  *bool   `json:"enabled,omitempty"`
}

type UpdateSettingsRequest struct {
	EventsRetentionDays     *int     `json:"events_retention_days,omitempty"`
	AggregatesRetentionDays *int     `json:"aggregates_retention_days,omitempty"`
	LatencyThresholdMs      *float64 `json:"latency_threshold_ms,omitempty"`
	ThroughputThresholdMb   *float64 `json:"throughput_threshold_mb,omitempty"`
	ErrorRateThreshold      *float64 `json:"error_rate_threshold,omitempty"`
	DNSBoundThreshold       *float64 `json:"dns_bound_threshold,omitempty"`
	HandshakeBoundSigma     *float64 `json:"handshake_bound_sigma,omitempty"`
	ServerBoundSigma        *float64 `json:"server_bound_sigma,omitempty"`
	ThroughputDropThreshold *float64 `json:"throughput_drop_threshold,omitempty"`
	MaxProbesPerClient      *int     `json:"max_probes_per_client,omitempty"`
	MaxTargetsPerProbe      *int     `json:"max_targets_per_probe,omitempty"`
	MaxSamplesPerWindow     *int     `json:"max_samples_per_window,omitempty"`
	RateLimitPerClientRPS   *int     `json:"rate_limit_per_client_rps,omitempty"`
}
