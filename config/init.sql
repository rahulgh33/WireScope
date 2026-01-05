-- Initial schema for WireScope
-- This file is automatically executed by PostgreSQL on first startup

-- Events deduplication table
CREATE TABLE IF NOT EXISTS events_seen (
    event_id UUID PRIMARY KEY,
    client_id VARCHAR(255) NOT NULL,
    ts_ms BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Index for efficient client/timestamp queries
CREATE INDEX IF NOT EXISTS idx_events_seen_client_ts ON events_seen(client_id, ts_ms);

-- Index for cleanup operations
CREATE INDEX IF NOT EXISTS idx_events_seen_created_at ON events_seen(created_at);

-- Aggregates table for 1-minute windows
CREATE TABLE IF NOT EXISTS agg_1m (
    client_id VARCHAR(255) NOT NULL,
    target VARCHAR(255) NOT NULL,
    window_start_ts TIMESTAMP NOT NULL,
    count_total BIGINT NOT NULL DEFAULT 0,
    count_success BIGINT NOT NULL DEFAULT 0,
    count_error BIGINT NOT NULL DEFAULT 0,
    dns_error_count BIGINT NOT NULL DEFAULT 0,
    tcp_error_count BIGINT NOT NULL DEFAULT 0,
    tls_error_count BIGINT NOT NULL DEFAULT 0,
    http_error_count BIGINT NOT NULL DEFAULT 0,
    throughput_error_count BIGINT NOT NULL DEFAULT 0,
    dns_p50 DOUBLE PRECISION,
    dns_p95 DOUBLE PRECISION,
    tcp_p50 DOUBLE PRECISION,
    tcp_p95 DOUBLE PRECISION,
    tls_p50 DOUBLE PRECISION,
    tls_p95 DOUBLE PRECISION,
    ttfb_p50 DOUBLE PRECISION,
    ttfb_p95 DOUBLE PRECISION,
    throughput_p50 DOUBLE PRECISION,
    throughput_p95 DOUBLE PRECISION,
    diagnosis_label VARCHAR(50),
    updated_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (client_id, target, window_start_ts)
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_agg_1m_window ON agg_1m(window_start_ts);
CREATE INDEX IF NOT EXISTS idx_agg_1m_diagnosis ON agg_1m(diagnosis_label) WHERE diagnosis_label IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_agg_1m_client_target_window ON agg_1m(client_id, target, window_start_ts DESC);

-- Optional alerts table for future use
CREATE TABLE IF NOT EXISTS alerts (
    id SERIAL PRIMARY KEY,
    client_id VARCHAR(255) NOT NULL,
    target VARCHAR(255) NOT NULL,
    alert_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    threshold_value DOUBLE PRECISION,
    actual_value DOUBLE PRECISION,
    window_start_ts TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    resolved_at TIMESTAMP
);

-- Indexes for alerts
CREATE INDEX IF NOT EXISTS idx_alerts_client_target ON alerts(client_id, target);
CREATE INDEX IF NOT EXISTS idx_alerts_created ON alerts(created_at);
CREATE INDEX IF NOT EXISTS idx_alerts_unresolved ON alerts(created_at) WHERE resolved_at IS NULL;