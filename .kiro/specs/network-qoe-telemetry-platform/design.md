# Network QoE Telemetry Platform Design Document

## Overview

The Network QoE Telemetry + Diagnosis Platform is a distributed system designed to measure, collect, aggregate, and analyze network quality of experience metrics. The platform follows a microservices architecture with four core components: a probe agent for measurements, an ingest API for event collection, an aggregator for data processing, and a diagnoser for bottleneck identification.

The system is built with production-grade reliability patterns including at-least-once delivery with exactly-once aggregate effects via deduplication, comprehensive observability, schema evolution support, and failure resilience. It uses a local-first architecture with Docker Compose for development, leveraging NATS JetStream for reliable messaging and PostgreSQL for durable storage.

## Architecture

### High-Level Architecture

```
┌─────────────┐    HTTP/JSON    ┌─────────────┐    NATS        ┌─────────────┐
│   Probe     │ ──────────────> │ Ingest API  │ ──────────────> │ Aggregator  │
│   (CLI)     │                 │  (HTTP)     │   JetStream    │ (Consumer)  │
└─────────────┘                 └─────────────┘                 └─────────────┘
                                        │                              │
                                        │                              │
                                        v                              v
                                ┌─────────────┐                ┌─────────────┐
                                │ Prometheus  │                │ PostgreSQL  │
                                │  Metrics    │                │  Storage    │
                                └─────────────┘                └─────────────┘
                                                                      │
                                                                      │
                                                                      v
                                                               ┌─────────────┐
                                                               │ Diagnoser   │
                                                               │ (Rules)     │
                                                               └─────────────┘
                                                                      │
                                                                      │
                                                                      v
                                                               ┌─────────────┐
                                                               │  Grafana    │
                                                               │ Dashboard   │
                                                               └─────────────┘
```

### Component Responsibilities

**Probe Agent**
- Measures DNS, TCP, TLS, HTTP timings and throughput
- Maintains stable client_id for identification
- Implements local queuing with backpressure handling
- Sends structured telemetry events to ingest API

**Ingest API**
- Validates API tokens and schema versions
- Implements rate limiting per client_id
- Publishes events to NATS JetStream
- Exposes Prometheus metrics and OpenTelemetry traces

**Aggregator**
- Consumes events from NATS JetStream
- Implements exactly-once processing with dedup table
- Computes time-windowed aggregates with percentiles
- Stores results in PostgreSQL with transactional guarantees

**Diagnoser**
- Analyzes aggregated data for performance bottlenecks
- Applies rule-based classification (DNS-bound, Server-bound, etc.)
- Compares against baseline using moving averages
- Updates diagnosis labels in aggregate records

## Components and Interfaces

### Probe Agent Interface

```go
type ProbeConfig struct {
    ClientID     string
    Targets      []string
    Interval     time.Duration
    IngestURL    string
    APIToken     string
    QueueSize    int
    BatchSize    int
}

type NetworkMeasurement struct {
    Target       string
    DNSMs        float64
    TCPMs        float64
    TLSMs        float64
    HTTPTTFBMs   float64
    ThroughputKbps float64
    ErrorStage   *string
}
```

### Ingest API Interface

```go
type TelemetryEvent struct {
    EventID        string                 `json:"event_id"`
    ClientID       string                 `json:"client_id"`
    TimestampMs    int64                  `json:"ts_ms"`
    RecvTimestampMs *int64                `json:"recv_ts_ms,omitempty"` // Set by ingest service
    SchemaVersion  string                 `json:"schema_version"`
    Target         string                 `json:"target"`
    NetworkContext NetworkContext         `json:"network_context"`
    Timings        TimingMeasurements     `json:"timings"`
    ThroughputKbps float64               `json:"throughput_kbps"`
    ErrorStage     *string               `json:"error_stage,omitempty"`
}

type NetworkContext struct {
    InterfaceType string  `json:"interface_type"`
    VPNEnabled    bool    `json:"vpn_enabled"`
    UserLabel     *string `json:"user_label,omitempty"`
}

type TimingMeasurements struct {
    DNSMs      float64 `json:"dns_ms"`
    TCPMs      float64 `json:"tcp_ms"`
    TLSMs      float64 `json:"tls_ms"`
    HTTPTTFBMs float64 `json:"http_ttfb_ms"`
}
```

### Aggregator Interface

```go
type WindowedAggregate struct {
    ClientID        string
    Target          string
    WindowStartTs   time.Time
    CountTotal      int64
    CountSuccess    int64
    CountError      int64
    ErrorStageCounts map[string]int64 // DNS, TCP, TLS, HTTP, throughput error counts
    DNSP50          float64
    DNSP95          float64
    TCPP50          float64
    TCPP95          float64
    TLSP50          float64
    TLSP95          float64
    TTFBP50         float64
    TTFBP95         float64
    ThroughputP50   float64
    ThroughputP95   float64
    DiagnosisLabel  *string
    UpdatedAt       time.Time
}
```

### Message Queue Interface

```go
type EventProcessor interface {
    PublishEvent(event *TelemetryEvent) error
    ConsumeEvents(handler func(*TelemetryEvent) error) error
    AckEvent(eventID string) error
}
```

## Data Models

### PostgreSQL Schema

#### Events Deduplication Table
```sql
CREATE TABLE events_seen (
    event_id UUID PRIMARY KEY,
    client_id VARCHAR(255) NOT NULL,
    ts_ms BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_events_seen_client_ts ON events_seen(client_id, ts_ms);
```

### Exactly-Once Aggregate Effects Implementation

**Deduplication Transaction Pattern**
```sql
BEGIN;
  -- Attempt dedup insert, ignore conflicts
  INSERT INTO events_seen (event_id, client_id, ts_ms) 
  VALUES ($1, $2, $3) 
  ON CONFLICT (event_id) DO NOTHING;
  
  -- Check if insert succeeded (new event)
  GET DIAGNOSTICS insert_count = ROW_COUNT;
  
  -- Only update aggregates for new events
  IF insert_count > 0 THEN
    INSERT INTO agg_1m (...) VALUES (...) 
    ON CONFLICT (client_id, target, window_start_ts) 
    DO UPDATE SET 
      count_total = agg_1m.count_total + 1,
      -- ... update percentiles and counters
      updated_at = NOW();
  END IF;
COMMIT;
```

**Race Condition Prevention**
- Primary key constraint on `events_seen.event_id` prevents duplicate processing
- Transaction ensures atomic dedup check + aggregate update
- Multiple aggregator workers safe due to database-level conflict resolution
- Failed transactions automatically retry with exponential backoff
```

#### Aggregates Table
```sql
CREATE TABLE agg_1m (
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

CREATE INDEX idx_agg_1m_window ON agg_1m(window_start_ts);
CREATE INDEX idx_agg_1m_diagnosis ON agg_1m(diagnosis_label) WHERE diagnosis_label IS NOT NULL;
CREATE INDEX idx_agg_1m_client_target_window ON agg_1m(client_id, target, window_start_ts DESC);
```

### Database Scalability and Retention

**Retention Policies**
- `events_seen` table: Retain for 7 days, then delete (dedup only needs recent history)
- `agg_1m` table: Retain for 90 days, consider partitioning by `window_start_ts`
- `alerts` table: Retain for 30 days for resolved alerts, indefinite for active alerts

**Partitioning Strategy**
- Partition `agg_1m` by month on `window_start_ts` for efficient queries and maintenance
- Consider hash partitioning on `client_id` if client count grows significantly
- Automated partition creation and cleanup via scheduled jobs

**Query Optimization**
- Primary queries: Recent data for specific (client_id, target) pairs
- Dashboard queries: Aggregated views across multiple clients/targets
- Index on (client_id, target, window_start_ts DESC) for time-series queries
- Materialized views for common dashboard aggregations

**Maintenance Jobs**
- Daily maintenance job deletes old rows from events_seen (>7 days)
- Weekly job removes old partitions from agg_1m (>90 days)
- Automated partition creation for upcoming months
```

#### Alerts Table (Optional)
```sql
CREATE TABLE alerts (
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

CREATE INDEX idx_alerts_client_target ON alerts(client_id, target);
CREATE INDEX idx_alerts_created ON alerts(created_at);
```

### NATS JetStream Configuration

```yaml
streams:
  - name: telemetry-events
    subjects: ["telemetry.events.*"]
    storage: file
    retention: limits
    max_age: 24h
    max_msgs: 1000000
    max_bytes: 1GB
    replicas: 1
    
  - name: telemetry-events-dlq
    subjects: ["telemetry.events.dlq"]
    storage: file
    retention: limits
    max_age: 168h  # 7 days
    max_msgs: 100000
    max_bytes: 100MB
    replicas: 1
    
consumers:
  - name: aggregator
    stream: telemetry-events
    durable: true
    deliver_policy: all
    ack_policy: explicit
    max_deliver: 3
    ack_wait: 30s
    max_ack_pending: 100
    
  - name: dlq-inspector
    stream: telemetry-events-dlq
    durable: true
    deliver_policy: all
    ack_policy: explicit
```

### Queue Semantics and Consumer Behavior

**Message Delivery**
- At-least-once delivery guarantees via NATS JetStream
- No ordering requirements across different (client_id, target) pairs
- Exponential backoff on redelivery: 1s, 2s, 4s, then DLQ
- Poison message handling: After 3 delivery attempts, republish to DLQ

**Dead Letter Queue Implementation**
- Aggregator checks `msg.NumDelivered >= max_deliver` before processing
- Failed messages republished to `telemetry.events.dlq` subject, then ACK original
- DLQ stream `telemetry-events-dlq` captures poison messages
- Manual inspection and replay capabilities via `dlq-inspector` consumer
- Alerting on DLQ message accumulation

**Consumer Configuration**
- Maximum 100 in-flight messages per consumer
- Acknowledgment only after successful database commit
- Consumer restart resumes from last acknowledged message
- Aggregator workers may be sharded by client_id to reduce database contention

**Horizontal Scaling**
- Multiple aggregator instances supported via JetStream consumer groups
- Deduplication primary key prevents race conditions across workers
- Optional: Partition consumers by client_id hash for reduced hot row contention

### Schema Validation Strategy

**Validation Modes**
- **Strict Mode**: Reject unknown schema_version; reject missing required fields; allow unknown fields
- **Lenient Mode**: Accept older versions and map to latest internal struct; allow unknown fields; log and count validation issues

**Schema Evolution Policy**
- New optional fields: Add to latest schema version, older versions ignore
- New required fields: Increment schema version, maintain backward compatibility
- Field removal: Deprecate in current version, remove in future version
- Type changes: Treat as new field with migration logic

**Observability Metrics**
- `schema_validation_total{version, mode, result}`: Count of validation attempts
- `schema_version_distribution{version}`: Distribution of incoming schema versions
- `validation_errors_total{error_type}`: Count of validation failures by type

**Window Assignment**
- Events assigned to 1-minute windows based on `ts_ms` field
- Window boundaries: `FLOOR(ts_ms / 60000) * 60000`
- Late event tolerance: 2 minutes based on aggregator processing time
- Events older than `now() - 2 minutes` are dropped and logged
- `recv_ts_ms` captured at ingest for clock skew debugging

**Window Lifecycle**
- Windows remain "open" for 2 minutes after their end time
- Open windows accept late-arriving events and update aggregates
- Closed windows are finalized and diagnosis rules applied
- Diagnosis runs when window closes (marked as provisional until final)
- Aggregate records are upserted with final statistics

**Percentile Computation (MVP)**
- Store all samples in memory per window per (client_id, target)
- Compute exact P50/P95 from sorted sample arrays (up to 10,000 samples)
- Beyond 10,000 samples: Downsample uniformly to maintain approximate percentiles
- Future optimization: Replace with HDRHistogram or t-digest sketches
- Memory management: Monitor per-window memory usage and apply backpressure

**Throughput Measurement Specification**
- Download 1MB fixed-size objects via `GET /fixed/1mb.bin?rand=<uuid>`
- Send `Cache-Control: no-cache` header to bypass CDN caching
- Use fresh TCP connection (disable keep-alive) for throughput measurement only
- 30-second timeout for downloads
- On timeout: Store partial throughput_kbps value and set error_stage="throughput_timeout"
- Use same target host as timing measurements for consistency

**Percentile and Error Handling Policy**
- Percentiles (P50/P95) computed over successful samples only
- Failed requests contribute to error counts but not percentile calculations
- Dashboards display error rates alongside percentiles for complete picture
- Quality counters provide context: count_total, count_success, count_error

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property Reflection

After analyzing all acceptance criteria, several properties can be consolidated to eliminate redundancy:

- Properties 1.1-1.5 (individual timing measurements) can be combined into a comprehensive measurement property
- Properties 2.1-2.5 (event structure validation) can be combined into an event format property  
- Properties 6.1-6.3 (metrics exposure) can be combined into a comprehensive observability property
- Properties 10.3 and 11.2 (schema validation and backward compatibility) overlap and can be unified

### Core Properties

**Property 1: Network measurement completeness**
*For any* valid target endpoint, when the probe performs measurements, all timing metrics (DNS, TCP, TLS, TTFB) and throughput should be captured with positive values or appropriate error indicators
**Validates: Requirements 1.1, 1.2, 1.3, 1.4, 1.5**

**Property 2: Event structure validity**
*For any* telemetry event generated, the event should contain valid UUID event_id, stable client_id, epoch timestamp, schema_version, and complete network context fields
**Validates: Requirements 2.1, 2.2, 2.3, 2.4, 2.5**

**Property 3: At-least-once delivery with exactly-once aggregate effects**
*For any* duplicate event published to the system, the aggregated statistics should remain unchanged after the duplicate is processed via deduplication
**Validates: Requirements 3.2, 8.1**

**Property 4: Transactional consistency**
*For any* event processing operation, deduplication insert and aggregate upsert should both succeed or both fail as a single atomic transaction
**Validates: Requirements 3.3**

**Property 5: Late event handling with explicit time reference**
*For any* event with timestamp older than aggregator processing time minus 2 minutes, the event should be dropped and logged rather than processed into aggregates
**Validates: Requirements 3.4**

**Property 6: Window assignment accuracy**
*For any* event with timestamp ts_ms, the event should be assigned to the window with start time FLOOR(ts_ms / 60000) * 60000
**Validates: Requirements 3.5, 4.1**

**Property 7: Percentile calculation correctness**
*For any* set of measurement samples within a time window, the computed P50 and P95 percentiles should match the exact percentiles of the sorted sample array
**Validates: Requirements 4.2, 4.4**

**Property 7a: Aggregate quality counters completeness**
*For any* time window aggregate, the system should maintain count_total, count_success, count_error, and per-stage error counts (DNS, TCP, TLS, HTTP, throughput) for comprehensive error analysis
**Validates: Requirements 4.4**

**Property 8: Throughput measurement consistency with explicit parameters**
*For any* throughput measurement, the calculation should use 1MB fixed-size object download over HTTPS with 30-second timeout, fresh connections, and consistent bytes-per-time computation
**Validates: Requirements 4.3**

**Property 9: DNS-bound diagnosis with explicit thresholds**
*For any* time window where DNS timing p95 represents ≥60% of total latency p95 and exceeds baseline by ≥50%, the diagnosis label should be set to "DNS-bound"
**Validates: Requirements 5.1**

**Property 10: Handshake-bound diagnosis with explicit thresholds**
*For any* time window where TCP or TLS timing p95 exceeds baseline by 2 standard deviations or 100%, the diagnosis label should be set to "Handshake-bound"
**Validates: Requirements 5.2**

**Property 11: Server-bound diagnosis with explicit thresholds**
*For any* time window where TTFB p95 exceeds baseline by 2 standard deviations while connection timings remain normal, the diagnosis label should be set to "Server-bound"
**Validates: Requirements 5.3**

**Property 12: Throughput-bound diagnosis with explicit thresholds**
*For any* time window where throughput p50 drops below baseline by ≥30% while latencies remain normal, the diagnosis label should be set to "Throughput-bound"
**Validates: Requirements 5.4**

**Property 13: Baseline calculation with explicit window count**
*For any* metric baseline calculation, the system should use simple moving average over the last 10 windows with consistent window selection
**Validates: Requirements 5.5**

**Property 14: Observability metrics with cardinality awareness**
*For any* system operation, appropriate Prometheus metrics should be exposed with low-cardinality labels (avoiding high-cardinality target labels), including request rates, error rates, queue depths, processing rates, and end-to-end processing delay
**Validates: Requirements 6.1, 6.2, 6.3**

**Property 15: Distributed tracing completeness**
*For any* request or event processing operation, OpenTelemetry spans should be created with appropriate context propagation
**Validates: Requirements 6.4, 6.5**

**Property 16: Authentication enforcement**
*For any* ingest API request without valid API token, the system should return 401 status and emit authentication failure metrics
**Validates: Requirements 10.1, 10.2**

**Property 17: Schema validation with forward compatibility**
*For any* event with known schema version, the system should process it successfully while allowing unknown fields, and events with unknown schema versions or missing required fields should be rejected with appropriate logging
**Validates: Requirements 10.3, 10.4, 10.5, 11.2**

**Property 18: Backpressure handling**
*For any* system component experiencing load beyond capacity, appropriate backpressure mechanisms should engage (bounded queues, exponential backoff, rate limiting, in-flight limits)
**Validates: Requirements 8.2, 8.3, 8.4**

**Property 19: System resilience under load**
*For any* burst traffic scenario, the system should handle increased lag without crashing and recover to normal performance levels
**Validates: Requirements 8.5**

**Property 20: At-least-once message delivery with proper acknowledgment**
*For any* event published to the message queue, the system should provide at-least-once delivery with acknowledgment only after durable storage commit
**Validates: Requirements 3.1, 8.4**

## Observability and Monitoring

### Prometheus Metrics

**End-to-End Latency Metrics**
- `processing_delay_ms`: Current time minus event ts_ms (end-to-end delay)
- `ingest_delay_ms`: recv_ts_ms minus ts_ms (client-to-ingest delay)
- `aggregate_delay_ms`: Aggregate update time minus recv_ts_ms (processing delay)

**Ingest API Metrics**
- `http_requests_total{method, status, client_id_hash}`: Request count (hash client_id for cardinality)
- `http_request_duration_seconds{method, status}`: Request latency histogram
- `authentication_failures_total{reason}`: Auth failure count by reason
- `rate_limit_exceeded_total{client_id_hash}`: Rate limiting events

**Queue and Consumer Metrics**
- `jetstream_consumer_lag_messages{consumer}`: Messages pending consumption
- `jetstream_consumer_ack_pending{consumer}`: Unacknowledged messages
- `message_processing_duration_seconds{consumer}`: Processing time per message
- `dlq_messages_total`: Count of messages sent to dead letter queue

**Aggregator Metrics**
- `events_processed_total{result}`: Count of processed events (success/duplicate/error)
- `dedup_rate`: Ratio of duplicate to total events
- `window_updates_total{client_id_hash, target_hash}`: Aggregate update count
- `percentile_computation_duration_seconds`: Time to compute percentiles per window

**Database Metrics**
- `database_connections_active`: Active database connections
- `database_query_duration_seconds{operation}`: Query execution time
- `database_transaction_duration_seconds{operation}`: Transaction time
- `events_seen_table_size_bytes`: Size of deduplication table

### Cardinality Management
- Hash high-cardinality labels (client_id, target) to prevent metric explosion
- Use separate logs for raw client_id/target values when needed for debugging
- Monitor metric cardinality and alert on excessive growth

## Error Handling

### Error Classification

**Network Measurement Errors**
- DNS resolution failures: Timeout, NXDOMAIN, server failure
- TCP connection errors: Connection refused, timeout, network unreachable
- TLS handshake errors: Certificate validation, protocol mismatch, timeout
- HTTP errors: 4xx/5xx responses, timeout, connection reset
- Throughput errors: Download timeout, incomplete transfer, network interruption

**System Processing Errors**
- Schema validation errors: Unknown version, missing required fields, invalid format
- Authentication errors: Missing token, invalid token, expired token
- Database errors: Connection failure, constraint violation, timeout
- Queue errors: Publish failure, consumer lag, message corruption
- Resource errors: Memory exhaustion, disk full, CPU overload

### Error Handling Strategies

**Probe Agent Error Handling**
```go
type ErrorHandling struct {
    MaxRetries      int           // 3 attempts per measurement
    RetryBackoff    time.Duration // Exponential backoff starting at 1s
    CircuitBreaker  bool          // Skip failing targets temporarily
    LocalQueue      int           // Buffer events during outages
    FallbackMode    bool          // Continue with partial measurements
}
```

**Ingest API Error Handling**
```go
type IngestErrorHandling struct {
    RequestTimeout  time.Duration // 30s per request
    RateLimit       int           // Per client_id limits
    ValidationMode  string        // "strict" or "lenient"
    ErrorMetrics    bool          // Emit detailed error metrics
    CircuitBreaker  bool          // Protect downstream services
}
```

**Aggregator Error Handling**
```go
type AggregatorErrorHandling struct {
    DeadLetterQueue bool          // Failed events to DLQ
    MaxRetries      int           // 3 attempts per event
    BatchTimeout    time.Duration // Process partial batches
    DatabaseRetry   bool          // Retry transient DB errors
    MemoryLimits    int64         // Prevent OOM on large windows
}
```

### Error Recovery Patterns

**Graceful Degradation**
- Continue processing with partial data when non-critical components fail
- Emit degraded service metrics to alert operators
- Maintain core functionality even with reduced feature set

**Circuit Breaker Pattern**
- Automatically disable failing components temporarily
- Implement health checks for automatic recovery
- Provide manual override capabilities for operators

**Retry with Backoff**
- Exponential backoff for transient failures
- Maximum retry limits to prevent infinite loops
- Jitter to prevent thundering herd problems

## Testing Strategy

### Dual Testing Approach

The platform requires both unit testing and property-based testing to ensure comprehensive coverage:

**Unit Testing Focus**
- Specific examples demonstrating correct behavior
- Integration points between components  
- Error conditions and edge cases
- Configuration validation
- Database schema compliance

**Property-Based Testing Focus**
- Universal properties that should hold across all inputs
- Correctness properties from the design document
- Invariants that must be maintained under all conditions
- System behavior under random input variations

### Property-Based Testing Framework

**Framework Selection**: Use **Testify** with **gopter** for Go-based property testing
- Minimum 100 iterations per property test
- Each property test tagged with design document reference
- Format: `**Feature: network-qoe-telemetry-platform, Property {number}: {property_text}**`

**Test Configuration**
```go
type PropertyTestConfig struct {
    Iterations    int           // Minimum 100
    MaxSize       int           // Generator size limits
    Timeout       time.Duration // Per-test timeout
    Shrinking     bool          // Enable counterexample shrinking
    Parallelism   int           // Concurrent test execution
}
```

### Integration Testing Strategy

**End-to-End Testing**
- Full pipeline tests from probe to dashboard
- Multi-component failure scenarios
- Performance and load testing
- Schema evolution testing

**Component Integration Testing**
- API contract validation between services
- Database transaction behavior
- Message queue delivery semantics
- Observability integration

### Performance Testing

**Load Testing Scenarios**
- High-frequency event ingestion (1000+ events/sec)
- Large batch processing (10,000+ events/batch)
- Concurrent client simulation (100+ probes)
- Extended duration testing (24+ hours)

**Performance Metrics**
- End-to-end latency (event creation to aggregate storage)
- Throughput capacity (events processed per second)
- Resource utilization (CPU, memory, disk, network)
- Queue depth and consumer lag under load

### Failure Mode Testing

**Chaos Engineering Tests**
- Random component failures during processing
- Network partitions between services
- Database connection interruptions
- Message queue unavailability
- Resource exhaustion scenarios

**Recovery Testing**
- Service restart behavior
- Data consistency after failures
- Backpressure activation and recovery
- Circuit breaker functionality

## Deployment Architecture

### Local Development Environment

**Docker Compose Services**
```yaml
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: telemetry
      POSTGRES_USER: telemetry
      POSTGRES_PASSWORD: telemetry
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    ports:
      - "5432:5432"

  nats:
    image: nats:2.10-alpine
    command: ["-js", "-sd", "/data"]
    volumes:
      - nats_data:/data
    ports:
      - "4222:4222"
      - "8222:8222"

  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./config/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    ports:
      - "9090:9090"

  grafana:
    image: grafana/grafana:latest
    environment:
      GF_SECURITY_ADMIN_PASSWORD: admin
    volumes:
      - grafana_data:/var/lib/grafana
      - ./config/grafana:/etc/grafana/provisioning
    ports:
      - "3000:3000"

  jaeger:
    image: jaegertracing/all-in-one:latest
    environment:
      COLLECTOR_OTLP_ENABLED: true
    ports:
      - "16686:16686"
      - "4317:4317"

  otel-collector:
    image: otel/opentelemetry-collector-contrib:latest
    volumes:
      - ./config/otel-collector.yml:/etc/otel-collector.yml
    command: ["--config=/etc/otel-collector.yml"]
    ports:
      - "4318:4318"
    depends_on:
      - jaeger
      - prometheus
```

### Production Considerations

**Scalability Patterns**
- Horizontal scaling of ingest API instances
- Partitioned aggregator workers by client_id
- Database read replicas for dashboard queries
- Message queue clustering for high availability

**Security Hardening**
- TLS encryption for all inter-service communication
- API token rotation and management
- Database connection encryption
- Network segmentation and firewall rules

**Monitoring and Alerting**
- SLO-based alerting (99% of events processed within 10 seconds)
- Resource utilization alerts (CPU, memory, disk)
- Error rate thresholds and anomaly detection
- Business metric monitoring (diagnosis accuracy, data quality)

**Backup and Recovery**
- Automated database backups with point-in-time recovery
- Message queue persistence and replication
- Configuration and dashboard backup
- Disaster recovery procedures and testing