-- Initial schema for Network QoE Telemetry Platform

-- Events deduplication table
CREATE TABLE IF NOT EXISTS events_seen (
    event_id UUID PRIMARY KEY,
    client_id VARCHAR(255) NOT NULL,
    ts_ms BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Index for efficient client/timestamp queries (idempotent)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_events_seen_client_ts') THEN
        CREATE INDEX idx_events_seen_client_ts ON events_seen(client_id, ts_ms);
    END IF;
END $$;

-- Index for cleanup operations (idempotent)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_events_seen_created_at') THEN
        CREATE INDEX idx_events_seen_created_at ON events_seen(created_at);
    END IF;
END $$;

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

-- Indexes for efficient queries (idempotent)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_agg_1m_window') THEN
        CREATE INDEX idx_agg_1m_window ON agg_1m(window_start_ts);
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_agg_1m_diagnosis') THEN
        CREATE INDEX idx_agg_1m_diagnosis ON agg_1m(diagnosis_label) WHERE diagnosis_label IS NOT NULL;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_agg_1m_client_target_window') THEN
        CREATE INDEX idx_agg_1m_client_target_window ON agg_1m(client_id, target, window_start_ts DESC);
    END IF;
END $$;

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

-- Indexes for alerts (idempotent)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_alerts_client_target') THEN
        CREATE INDEX idx_alerts_client_target ON alerts(client_id, target);
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_alerts_created') THEN
        CREATE INDEX idx_alerts_created ON alerts(created_at);
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_alerts_unresolved') THEN
        CREATE INDEX idx_alerts_unresolved ON alerts(created_at) WHERE resolved_at IS NULL;
    END IF;
END $$;