# Performance Tuning Guide - Network QoE Telemetry Platform

This guide provides recommendations for optimizing the Network QoE Telemetry Platform for production workloads and high-throughput scenarios.

## Table of Contents

- [Baseline Performance](#baseline-performance)
- [Component-Specific Tuning](#component-specific-tuning)
- [Database Optimization](#database-optimization)
- [NATS JetStream Configuration](#nats-jetstream-configuration)
- [Scaling Strategies](#scaling-strategies)
- [Monitoring Performance](#monitoring-performance)
- [Load Testing](#load-testing)

## Baseline Performance

### Expected Throughput (Single Instance)

| Component | Events/Second | Latency (P95) | CPU Usage | Memory Usage |
|-----------|---------------|---------------|-----------|--------------|
| Probe Agent | 10-100 | N/A | <5% | <50MB |
| Ingest API | 10,000+ | <10ms | 10-30% | <500MB |
| Aggregator | 5,000+ | <100ms | 20-40% | 200-500MB |
| Diagnoser | N/A (batch) | <1s per run | <10% | <200MB |

### Resource Recommendations

**Development**:
- CPU: 4 cores
- RAM: 8GB
- Disk: 20GB SSD

**Production (per component)**:
- Ingest API: 2-4 cores, 2-4GB RAM
- Aggregator: 2-4 cores, 2-4GB RAM
- PostgreSQL: 4-8 cores, 8-16GB RAM
- NATS: 2-4 cores, 2-4GB RAM

## Component-Specific Tuning

### Probe Agent

#### Measurement Interval

```bash
# High-frequency monitoring (development/debugging)
./bin/probe --interval 5s

# Normal monitoring (production)
./bin/probe --interval 30s

# Low-frequency monitoring (cost-sensitive)
./bin/probe --interval 5m
```

**Trade-offs**:
- Shorter intervals: Higher granularity, more cost, more load
- Longer intervals: Lower cost, less granularity, delayed detection

#### Concurrent Measurements

```go
// internal/probe/measurement.go
const (
    // Increase for faster multi-target measurements
    MaxConcurrentMeasurements = 10  // Default: 5
    
    // Connection timeout
    MeasurementTimeout = 30 * time.Second  // Default
)
```

#### Network Optimizations

```bash
# Use HTTP/2 when possible
# Reduces connection overhead

# Enable connection pooling
# Reuse connections to ingest API

# Set reasonable timeouts
--timeout 30s  # Total measurement timeout
```

### Ingest API

#### Rate Limiting

```go
// cmd/ingest/main.go
rateLimiter := NewRateLimiter(
    100, // Requests per second per client (increase for production)
    60,  // Burst size
)
```

**Tuning**:
- Too low: Legitimate traffic rejected (HTTP 429)
- Too high: Vulnerable to abuse, higher resource usage
- Recommended: 100 RPS/client for production

#### Connection Pool

```go
// Database connection pool
db.SetMaxOpenConns(25)        // Default: 25, Production: 50-100
db.SetMaxIdleConns(25)        // Default: 25, Production: 25-50
db.SetConnMaxLifetime(5 * time.Minute)  // Default: 5min
```

#### HTTP Server Tuning

```go
server := &http.Server{
    Addr:              ":8081",
    ReadTimeout:       10 * time.Second,  // Default
    WriteTimeout:      10 * time.Second,  // Default
    IdleTimeout:       120 * time.Second, // Default
    MaxHeaderBytes:    1 << 20,           // 1MB
    ReadHeaderTimeout: 5 * time.Second,
}
```

**Production recommendations**:
```go
ReadTimeout:       5 * time.Second,   // Faster timeout
WriteTimeout:      10 * time.Second,
IdleTimeout:       60 * time.Second,  // Lower idle
MaxHeaderBytes:    512 * 1024,        // 512KB (smaller headers)
```

#### NATS Publishing

```go
// Batch publishing for high throughput
const (
    BatchSize     = 100                 // Messages per batch
    BatchTimeout  = 100 * time.Millisecond  // Max wait time
)

// Enable async publishing
js.PublishAsync("telemetry.events", data,
    nats.StallWait(5*time.Second),
)
```

### Aggregator

#### Consumer Configuration

```go
// Consumer with tuned parameters
_, err := js.Subscribe("telemetry.events", processEvent,
    nats.Durable("aggregator"),
    nats.ManualAck(),
    nats.AckWait(30*time.Second),          // Default: 30s, Production: 60s
    nats.MaxAckPending(1000),              // Default: 1000, Production: 5000
    nats.MaxDeliver(3),                    // Default: 3
)
```

**Tuning**:
- `MaxAckPending`: Higher = more throughput, higher memory
- `AckWait`: Longer = handles slow processing, risk of duplicates
- `MaxDeliver`: Higher = more retries, could delay DLQ routing

#### Window Flush Strategy

```go
// Flush window when:
// 1. Window closes (time-based)
// 2. Sample limit reached (memory-based)
// 3. Manual flush trigger

const (
    MaxSamplesPerWindow = 10000           // Default: 10k
    WindowDuration      = 1 * time.Minute // Default: 1min
    FlushBufferSize     = 100             // Batch flush multiple windows
)
```

**Optimization**:
```go
// For high-cardinality workloads (many client-target pairs):
MaxSamplesPerWindow = 5000  // Reduce memory per window

// For low-latency requirements:
WindowDuration = 30 * time.Second  // Shorter windows

// For high throughput:
FlushBufferSize = 500  // Larger batches
```

#### Database Write Optimization

```go
// Use prepared statements
stmt, _ := db.Prepare(`
    INSERT INTO agg_1m (client_id, target, window_start_ts, ...)
    VALUES ($1, $2, $3, ...)
    ON CONFLICT (client_id, target, window_start_ts)
    DO UPDATE SET ...
`)

// Batch writes
tx.Begin()
for _, agg := range aggregates {
    stmt.Exec(agg...)
}
tx.Commit()
```

#### Memory Management

```go
// Limit in-memory windows
const MaxActiveWindows = 100  // Per client-target pair

// Implement LRU eviction
type WindowCache struct {
    maxSize int
    cache   *lru.Cache
}

// Periodic cleanup of old windows
ticker := time.NewTicker(5 * time.Minute)
go func() {
    for range ticker.C {
        cleanupOldWindows()
    }
}()
```

### Diagnoser

#### Batch Processing

```go
// Process diagnosis in batches
const (
    DiagnosisBatchSize = 1000         // Aggregates per batch
    DiagnosisInterval  = 60 * time.Second  // Default: 1min
)

// For high-cardinality:
DiagnosisBatchSize = 5000
DiagnosisInterval = 5 * time.Minute  // Less frequent
```

#### Baseline Calculation Optimization

```go
// Cache baseline calculations
type BaselineCache struct {
    mu        sync.RWMutex
    baselines map[string]*Baseline
    ttl       time.Duration
}

// Update baselines incrementally
// Instead of recalculating from scratch every time
func UpdateBaseline(clientID, target string, newWindow *Aggregate) {
    // Add new window to moving average
    // Remove oldest window
    // O(1) update instead of O(n) full recalculation
}
```

## Database Optimization

### PostgreSQL Configuration

Edit `postgresql.conf` or set via environment:

```ini
# Connection Settings
max_connections = 100                  # Default: 100, Production: 200-500
shared_buffers = 256MB                 # Default: 128MB, Production: 4-8GB
effective_cache_size = 1GB             # Default: 4GB, Production: 50-75% of RAM

# Write Performance
wal_buffers = 16MB                     # Default: -1 (auto)
checkpoint_completion_target = 0.9     # Default: 0.5
max_wal_size = 4GB                     # Default: 1GB
min_wal_size = 1GB                     # Default: 80MB

# Query Planning
random_page_cost = 1.1                 # Default: 4.0 (for SSD)
effective_io_concurrency = 200         # Default: 1 (for SSD)

# Autovacuum (Critical for high-write workloads)
autovacuum = on                        # Default: on
autovacuum_max_workers = 4             # Default: 3
autovacuum_naptime = 30s               # Default: 1min (more frequent)
```

### Table-Specific Settings

```sql
-- events_seen: High-churn table
ALTER TABLE events_seen SET (
    autovacuum_vacuum_scale_factor = 0.05,  -- More frequent vacuum
    autovacuum_analyze_scale_factor = 0.02,
    autovacuum_vacuum_cost_delay = 10ms,
    fillfactor = 90                          -- Leave room for updates
);

-- agg_1m: Frequent upserts
ALTER TABLE agg_1m SET (
    autovacuum_vacuum_scale_factor = 0.1,
    autovacuum_analyze_scale_factor = 0.05,
    fillfactor = 90
);
```

### Index Optimization

```sql
-- Check index usage
SELECT 
    schemaname, tablename, indexname, idx_scan, idx_tup_read, idx_tup_fetch
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
ORDER BY idx_scan DESC;

-- Drop unused indexes (idx_scan = 0)

-- Add indexes for common queries
CREATE INDEX CONCURRENTLY idx_agg_1m_client_window 
ON agg_1m(client_id, window_start_ts DESC) 
WHERE diagnosis_label IS NOT NULL;

-- Partial indexes for common filters
CREATE INDEX CONCURRENTLY idx_agg_1m_errors
ON agg_1m(client_id, target, window_start_ts DESC)
WHERE count_error > 0;
```

### Query Optimization

```sql
-- Use EXPLAIN ANALYZE to profile queries
EXPLAIN (ANALYZE, BUFFERS) 
SELECT * FROM agg_1m 
WHERE client_id = 'client-123' 
  AND window_start_ts > NOW() - INTERVAL '1 hour'
ORDER BY window_start_ts DESC;

-- Look for:
-- - Sequential scans (should use indexes)
-- - High buffer reads (needs more cache)
-- - Slow sort operations (add indexes)
```

### Partitioning (For Large Datasets)

```sql
-- Convert agg_1m to partitioned table
CREATE TABLE agg_1m_partitioned (
    LIKE agg_1m INCLUDING ALL
) PARTITION BY RANGE (window_start_ts);

-- Create monthly partitions
CREATE TABLE agg_1m_2025_12 PARTITION OF agg_1m_partitioned
    FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');

CREATE TABLE agg_1m_2026_01 PARTITION OF agg_1m_partitioned
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');

-- Benefits:
-- - Faster queries (partition pruning)
-- - Easier maintenance (drop old partitions)
-- - Better index performance (smaller indexes)
```

### Connection Pooling

```bash
# Use PgBouncer for connection pooling
docker run -d \
  --name pgbouncer \
  -e POSTGRESQL_HOST=postgres \
  -e POSTGRESQL_PORT=5432 \
  -e POSTGRESQL_USERNAME=telemetry \
  -e POSTGRESQL_PASSWORD=telemetry \
  -e POSTGRESQL_DATABASE=telemetry \
  -e PGBOUNCER_POOL_MODE=transaction \
  -e PGBOUNCER_MAX_CLIENT_CONN=1000 \
  -e PGBOUNCER_DEFAULT_POOL_SIZE=25 \
  -p 6432:6432 \
  bitnami/pgbouncer:latest

# Update applications to connect through PgBouncer
# Connection string: postgres://telemetry:telemetry@localhost:6432/telemetry
```

## NATS JetStream Configuration

### Stream Configuration

```bash
# Optimize for high throughput
nats stream add telemetry-events \
  --subjects "telemetry.events" \
  --storage file \
  --retention limits \
  --max-age 7d \
  --max-msgs 10000000 \
  --max-bytes 10GB \
  --max-msg-size 1MB \
  --replicas 1 \
  --discard old

# For high availability (requires NATS cluster)
--replicas 3
```

### Consumer Configuration

```bash
# Optimize consumer for throughput
nats consumer add telemetry-events aggregator \
  --filter "telemetry.events" \
  --ack explicit \
  --replay instant \
  --deliver all \
  --max-deliver 3 \
  --max-ack-pending 5000 \
  --ack-wait 60s

# For lower latency:
--max-ack-pending 1000
--ack-wait 30s

# For higher throughput:
--max-ack-pending 10000
--ack-wait 120s
```

### NATS Server Tuning

Edit `nats.conf`:

```conf
# Resource limits
max_payload = 1MB          # Default: 1MB
max_connections = 10000    # Default: 64k
max_pending = 256MB        # Default: 256MB

# JetStream
jetstream {
    max_memory_store = 4GB   # In-memory limit
    max_file_store = 100GB   # File storage limit
    store_dir = "/data/nats"
}

# Performance
write_deadline = "10s"     # Default: 10s
ping_interval = "2m"       # Default: 2m
```

## Scaling Strategies

### Horizontal Scaling

#### Ingest API (Multiple Instances)

```bash
# Run multiple ingest instances behind a load balancer
./bin/ingest --port 8081 &
./bin/ingest --port 8082 &
./bin/ingest --port 8083 &

# Use nginx as load balancer
# nginx.conf:
upstream ingest_backend {
    least_conn;
    server localhost:8081;
    server localhost:8082;
    server localhost:8083;
}

server {
    listen 80;
    location /api/ {
        proxy_pass http://ingest_backend;
    }
}
```

**Considerations**:
- Rate limiting needs distributed state (Redis)
- Metrics aggregation across instances
- Health checks for load balancer

#### Aggregator (Consumer Groups)

```go
// Multiple aggregator instances with same consumer group
// NATS automatically distributes messages

// Instance 1
./bin/aggregator --consumer-group aggregator-group &

// Instance 2  
./bin/aggregator --consumer-group aggregator-group &

// Instance 3
./bin/aggregator --consumer-group aggregator-group &
```

**Benefits**:
- Automatic load distribution
- Fault tolerance (other instances continue if one fails)
- Increased throughput

**Considerations**:
- Each instance needs database connection
- Coordinate window flushing across instances
- Monitor consumer lag per instance

### Vertical Scaling

#### CPU Optimization

```bash
# Set GOMAXPROCS to match CPU cores
export GOMAXPROCS=8
./bin/aggregator

# Profile CPU usage
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Identify bottlenecks
# Look for: hot functions, lock contention
```

#### Memory Optimization

```bash
# Set GC target percentage (lower = more frequent GC)
export GOGC=75  # Default: 100
./bin/aggregator

# Profile memory
go tool pprof http://localhost:6060/debug/pprof/heap

# Reduce memory allocations
# - Reuse buffers
# - Pool objects
# - Limit in-memory windows
```

### Database Scaling

#### Read Replicas

```yaml
# docker-compose.yml
postgres-primary:
  image: postgres:15
  environment:
    - POSTGRES_USER=telemetry
    - POSTGRES_PASSWORD=telemetry
  command: postgres -c wal_level=replica

postgres-replica:
  image: postgres:15
  environment:
    - POSTGRES_USER=telemetry
    - POSTGRES_PASSWORD=telemetry
  command: |
    bash -c "
    until pg_basebackup -h postgres-primary -U telemetry -D /var/lib/postgresql/data -P; do
      sleep 5
    done
    postgres -c hot_standby=on
    "
```

**Use replicas for**:
- Diagnoser reads
- Grafana queries
- Analytics workloads

**Write to primary only**:
- Aggregator writes
- Deduplication checks

#### Connection Pooling with PgBouncer

See [Database Optimization](#connection-pooling) section above.

## Monitoring Performance

### Key Metrics to Watch

#### Ingest API

```promql
# Request rate
rate(telemetry_ingest_requests_total[1m])

# Error rate
rate(telemetry_ingest_errors_total[1m]) / rate(telemetry_ingest_requests_total[1m])

# Response time
histogram_quantile(0.95, rate(telemetry_ingest_duration_seconds_bucket[1m]))

# Rate limiting
rate(telemetry_ingest_rate_limited_total[1m])
```

#### Aggregator

```promql
# Processing rate
rate(telemetry_events_processed_total[1m])

# Processing delay (end-to-end latency)
histogram_quantile(0.95, rate(telemetry_processing_delay_seconds_bucket[1m]))

# Consumer lag
telemetry_queue_consumer_lag_messages

# Duplicate rate
rate(telemetry_events_processed_total{status="duplicate"}[1m]) / 
rate(telemetry_events_processed_total[1m])
```

#### Database

```promql
# Connection usage
telemetry_db_connections_in_use / telemetry_db_connections_max

# Query duration
histogram_quantile(0.95, rate(telemetry_db_query_duration_seconds_bucket[1m]))

# Transaction rate
rate(telemetry_db_transactions_total[1m])
```

#### NATS

```promql
# Stream size
nats_jetstream_stream_messages

# Consumer lag
nats_jetstream_consumer_num_pending

# Ack rate
rate(nats_jetstream_consumer_ack_floor[1m])
```

### Alert Thresholds

```yaml
# prometheus-alerts.yml
groups:
  - name: performance
    rules:
      - alert: HighProcessingDelay
        expr: histogram_quantile(0.95, rate(telemetry_processing_delay_seconds_bucket[5m])) > 10
        for: 5m
        annotations:
          summary: "P95 processing delay > 10 seconds"
          
      - alert: HighConsumerLag
        expr: telemetry_queue_consumer_lag_messages > 10000
        for: 5m
        annotations:
          summary: "Consumer lag > 10k messages"
          
      - alert: HighDatabaseConnections
        expr: telemetry_db_connections_in_use / telemetry_db_connections_max > 0.8
        for: 5m
        annotations:
          summary: "Database connection pool > 80% utilized"
```

## Load Testing

### Apache Bench (Simple Load Test)

```bash
# Test ingest API throughput
ab -n 10000 -c 100 -T 'application/json' -p event.json \
  http://localhost:8081/api/v1/events

# event.json:
{
  "event_id": "test-event-001",
  "client_id": "load-test-client",
  "ts_ms": 1703347200000,
  "target": "http://test.example.com",
  "network_context": {...},
  "timing_measurements": {...}
}
```

### Vegeta (Advanced Load Test)

```bash
# Install vegeta
go install github.com/tsenart/vegeta@latest

# Create target file
cat > targets.txt <<EOF
POST http://localhost:8081/api/v1/events
Content-Type: application/json
Authorization: Bearer demo-token
@event.json
EOF

# Run load test
echo "POST http://localhost:8081/api/v1/events" | \
  vegeta attack -rate=1000 -duration=60s -timeout=30s \
  -header="Content-Type: application/json" \
  -header="Authorization: Bearer demo-token" \
  -body=@event.json | \
  vegeta report

# Results:
# Requests      [total, rate, throughput]  60000, 1000.00, 998.50
# Duration      [total, attack, wait]      60.1s, 60s, 100ms
# Latencies     [mean, 50, 95, 99, max]    5ms, 4ms, 8ms, 15ms, 50ms
# Success       [ratio]                    99.95%
```

### Distributed Load Testing (k6)

```javascript
// load-test.js
import http from 'k6/http';
import { check } from 'k6';

export const options = {
  stages: [
    { duration: '1m', target: 100 },   // Ramp up
    { duration: '5m', target: 1000 },  // Sustained load
    { duration: '1m', target: 0 },     // Ramp down
  ],
};

export default function () {
  const payload = JSON.stringify({
    event_id: `test-${__VU}-${__ITER}`,
    client_id: `client-${__VU}`,
    ts_ms: Date.now(),
    target: 'http://test.example.com',
    // ... full event payload
  });

  const res = http.post('http://localhost:8081/api/v1/events', payload, {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer demo-token',
    },
  });

  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 100ms': (r) => r.timings.duration < 100,
  });
}
```

```bash
# Run k6 load test
k6 run --vus 100 --duration 5m load-test.js

# Results show:
# - RPS achieved
# - Error rate
# - Latency distribution
# - Resource utilization
```

### Monitoring During Load Test

```bash
# Watch metrics in real-time
watch -n 1 'curl -s http://localhost:8081/metrics | grep -E "telemetry_(ingest|queue)_"'

# Monitor system resources
docker stats

# Check consumer lag
watch -n 1 'docker exec nats nats consumer info telemetry-events aggregator'

# Monitor database
docker exec postgres psql -U telemetry -d telemetry -c \
  "SELECT count(*) FROM pg_stat_activity WHERE state != 'idle';"
```

## Performance Checklist

### Before Production

- [ ] Load test with expected traffic (2x peak)
- [ ] Tune database connection pools
- [ ] Configure autovacuum for high-write tables
- [ ] Set up database monitoring and alerting
- [ ] Implement connection pooling (PgBouncer)
- [ ] Configure NATS stream retention
- [ ] Set up horizontal pod autoscaling (Kubernetes)
- [ ] Enable pprof for performance profiling
- [ ] Configure log levels (reduce in production)
- [ ] Set up distributed tracing sampling

### Regular Maintenance

- [ ] Review slow query logs weekly
- [ ] Run VACUUM ANALYZE monthly
- [ ] Check index usage quarterly
- [ ] Review and optimize Grafana dashboards
- [ ] Update resource limits based on actual usage
- [ ] Test backup and restore procedures
- [ ] Review and adjust alert thresholds

### When Scaling

- [ ] Benchmark before and after changes
- [ ] Monitor key metrics during rollout
- [ ] Have rollback plan ready
- [ ] Test failure scenarios
- [ ] Document configuration changes
- [ ] Update runbooks with new procedures

## Conclusion

Performance tuning is an iterative process. Start with the baseline configuration, measure performance under realistic load, identify bottlenecks, apply targeted optimizations, and repeat.

Key principles:
1. **Measure first**: Profile before optimizing
2. **Test thoroughly**: Load test every significant change
3. **Monitor continuously**: Track metrics in production
4. **Scale gradually**: Incremental improvements are safer
5. **Document everything**: Record what works and what doesn't

For specific performance issues, see [Troubleshooting Guide](TROUBLESHOOTING.md).
