# Requirements Document

## Introduction

The Network QoE Telemetry + Diagnosis Platform is a distributed system that measures network quality of experience metrics (DNS/TCP/TLS/HTTP timings and throughput) from local agents, processes events through a cloud-like pipeline, aggregates data into time windows, applies diagnostic labeling, and provides observability through dashboards and alerts. The system is designed with production-grade reliability patterns including idempotency, retries, backpressure handling, schema evolution, and comprehensive observability.

## Glossary

- **Probe**: CLI agent that measures network performance metrics and sends telemetry events
- **Ingest_API**: HTTP API service that receives telemetry events from probes
- **Aggregator**: Consumer/worker service that processes events and creates time-windowed aggregates
- **Diagnoser**: Rule-based service that analyzes aggregated data and labels performance bottlenecks
- **TelemetryEvent**: Structured data containing network performance measurements
- **Network_QoE_Platform**: The complete distributed telemetry system
- **JetStream**: NATS messaging system providing at-least-once delivery guarantees
- **Dedup_Table**: Database table ensuring exactly-once processing of events
- **Time_Window**: One-minute aggregation period for telemetry data
- **Diagnosis_Label**: Classification of network performance bottleneck type

## Requirements

### Requirement 1

**User Story:** As a network operations engineer, I want to deploy local measurement agents, so that I can collect real-time network performance data from distributed locations.

#### Acceptance Criteria

1. WHEN the probe CLI is executed with target configuration, THE Network_QoE_Platform SHALL measure DNS resolution timing in milliseconds
2. WHEN the probe performs network measurements, THE Network_QoE_Platform SHALL capture TCP connection establishment timing in milliseconds
3. WHEN the probe establishes secure connections, THE Network_QoE_Platform SHALL measure TLS handshake timing in milliseconds
4. WHEN the probe sends HTTP requests, THE Network_QoE_Platform SHALL measure time-to-first-byte in milliseconds
5. WHEN the probe transfers data, THE Network_QoE_Platform SHALL calculate throughput in kilobits per second

### Requirement 2

**User Story:** As a system architect, I want structured telemetry events with schema versioning, so that the platform can evolve while maintaining backward compatibility.

#### Acceptance Criteria

1. WHEN telemetry events are created, THE Network_QoE_Platform SHALL include event_id as UUID for unique identification
2. WHEN events are generated, THE Network_QoE_Platform SHALL include client_id as stable identifier stored locally by probe
3. WHEN measurements are taken, THE Network_QoE_Platform SHALL include timestamp in milliseconds since epoch and recv_ts_ms set by ingest service
4. WHEN events are structured, THE Network_QoE_Platform SHALL include schema_version for evolution support
5. WHEN network context is captured, THE Network_QoE_Platform SHALL include interface_type, vpn_enabled, and optional user labels

### Requirement 3

**User Story:** As a platform operator, I want reliable event delivery with at-least-once delivery and exactly-once aggregate effects, so that aggregated metrics remain accurate despite network failures.

#### Acceptance Criteria

1. WHEN events are published to the message queue, THE Network_QoE_Platform SHALL provide at-least-once delivery guarantees
2. WHEN duplicate events are received by aggregator, THE Network_QoE_Platform SHALL detect duplicates using events_seen dedup table
3. WHEN processing events, THE Network_QoE_Platform SHALL perform dedup insert and aggregate upsert in a single database transaction
4. WHEN events arrive late, THE Network_QoE_Platform SHALL allow lateness of 2 minutes based on aggregator processing time and drop events older than now() minus lateness
5. WHEN assigning events to windows, THE Network_QoE_Platform SHALL use event ts_ms for window assignment and keep windows mutable until window_end plus lateness tolerance

### Requirement 4

**User Story:** As a performance analyst, I want time-windowed aggregations with percentile calculations, so that I can analyze network performance trends over time.

#### Acceptance Criteria

1. WHEN events are processed, THE Network_QoE_Platform SHALL aggregate data into one-minute time windows
2. WHEN calculating percentiles for MVP, THE Network_QoE_Platform SHALL compute exact p50 and p95 from full sample set held in memory per window
3. WHEN measuring throughput, THE Network_QoE_Platform SHALL download 1MB fixed-size objects over HTTPS with 30-second timeout and force fresh connections to avoid caching
4. WHEN computing statistics, THE Network_QoE_Platform SHALL calculate p50 and p95 percentiles for DNS, TCP, TLS timing and throughput measurements plus count_total, count_success, count_error, and per-stage error counts
5. WHEN scaling beyond MVP, THE Network_QoE_Platform SHALL replace exact computation with HDRHistogram or t-digest sketches

### Requirement 5

**User Story:** As a network troubleshooter, I want automated diagnosis of performance bottlenecks, so that I can quickly identify root causes of network issues.

#### Acceptance Criteria

1. WHEN DNS timing p95 exceeds 60% of total latency p95 and exceeds baseline by 50%, THE Network_QoE_Platform SHALL label the window as DNS-bound
2. WHEN TCP or TLS timing p95 exceeds baseline by 2 standard deviations or 100%, THE Network_QoE_Platform SHALL label the window as Handshake-bound
3. WHEN TTFB p95 exceeds baseline by 2 standard deviations while connection timings remain normal, THE Network_QoE_Platform SHALL label the window as Server-bound
4. WHEN throughput p50 drops below baseline by 30% while latencies remain normal, THE Network_QoE_Platform SHALL label the window as Throughput-bound
5. WHEN comparing against baseline, THE Network_QoE_Platform SHALL use simple moving average over last 10 windows

### Requirement 6

**User Story:** As a DevOps engineer, I want comprehensive observability and monitoring, so that I can operate the platform reliably in production.

#### Acceptance Criteria

1. WHEN processing requests, THE Network_QoE_Platform SHALL expose Prometheus metrics for ingest request rate, error rate, and latency with low-cardinality labels
2. WHEN consuming from queues, THE Network_QoE_Platform SHALL expose metrics for queue depth and consumer lag
3. WHEN processing events, THE Network_QoE_Platform SHALL expose metrics for processed events per second, dedup rate, and end-to-end processing delay
4. WHEN tracing requests, THE Network_QoE_Platform SHALL generate OpenTelemetry spans for ingest requests
5. WHEN processing events, THE Network_QoE_Platform SHALL generate OpenTelemetry spans for aggregator operations

### Requirement 7

**User Story:** As a platform operator, I want local development environment with production-like components, so that I can test and develop the system effectively.

#### Acceptance Criteria

1. WHEN setting up development environment, THE Network_QoE_Platform SHALL provide Docker Compose configuration
2. WHEN running locally, THE Network_QoE_Platform SHALL include PostgreSQL for persistent storage
3. WHEN running locally, THE Network_QoE_Platform SHALL include NATS JetStream for message queuing
4. WHEN running locally, THE Network_QoE_Platform SHALL include Prometheus for metrics collection
5. WHEN running locally, THE Network_QoE_Platform SHALL include Grafana for visualization and Jaeger for distributed tracing

### Requirement 8

**User Story:** As a system reliability engineer, I want failure-mode testing capabilities with defined backpressure handling, so that I can validate the platform's resilience under adverse conditions.

#### Acceptance Criteria

1. WHEN duplicate events are published, THE Network_QoE_Platform SHALL maintain unchanged aggregates
2. WHEN probe experiences backpressure, THE Network_QoE_Platform SHALL use bounded local queue with exponential backoff
3. WHEN ingest receives high traffic, THE Network_QoE_Platform SHALL implement rate limiting per client_id
4. WHEN aggregator processes events, THE Network_QoE_Platform SHALL limit in-flight messages and ack only after durable write
5. WHEN burst traffic occurs, THE Network_QoE_Platform SHALL handle increased lag without crashing and recover to normal performance

### Requirement 9

**User Story:** As a developer, I want clear documentation and development tools, so that I can contribute to and maintain the platform effectively.

#### Acceptance Criteria

1. WHEN setting up the project, THE Network_QoE_Platform SHALL provide Makefile with standard targets
2. WHEN documenting the system, THE Network_QoE_Platform SHALL include architecture diagram in README
3. WHEN explaining design decisions, THE Network_QoE_Platform SHALL provide design documentation covering reliability tradeoffs including delivery semantics, idempotency, backpressure, and failure handling
4. WHEN demonstrating the system, THE Network_QoE_Platform SHALL include step-by-step demo instructions
5. WHEN defining service quality, THE Network_QoE_Platform SHALL specify SLO such as 99% of events processed into aggregates within 10 seconds

### Requirement 10

**User Story:** As a security engineer, I want API authentication and schema validation, so that the platform prevents unauthorized access and handles schema evolution safely.

#### Acceptance Criteria

1. WHEN ingest receives requests, THE Network_QoE_Platform SHALL require API token per client_id via header-based authentication
2. WHEN authentication fails, THE Network_QoE_Platform SHALL return 401 status and emit authentication failure metrics
3. WHEN validating events, THE Network_QoE_Platform SHALL reject events with unknown schema_version or missing required fields for that version
4. WHEN processing events with extra fields, THE Network_QoE_Platform SHALL allow unknown fields and optionally log them for forward compatibility
5. WHEN handling schema evolution, THE Network_QoE_Platform SHALL maintain backward compatibility for known schema versions

### Requirement 11

**User Story:** As a data engineer, I want schema evolution and migration capabilities, so that the platform can adapt to changing requirements over time.

#### Acceptance Criteria

1. WHEN database schema changes, THE Network_QoE_Platform SHALL provide migration scripts
2. WHEN event schema evolves, THE Network_QoE_Platform SHALL maintain backward compatibility
3. WHEN storing events, THE Network_QoE_Platform SHALL use events_seen table with event_id primary key
4. WHEN storing aggregates, THE Network_QoE_Platform SHALL use agg_1m table with composite primary key
5. WHEN visualizing data, THE Network_QoE_Platform SHALL provide starter Grafana dashboard configuration