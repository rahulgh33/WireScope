# Troubleshooting Guide - Network QoE Telemetry Platform

This guide helps diagnose and resolve common issues with the Network QoE Telemetry Platform.

## Quick Diagnostic Commands

```bash
# Check all services status
docker-compose ps

# View logs for specific service
docker-compose logs -f [postgres|nats|prometheus|grafana]

# Check NATS stream status
docker exec nats nats stream info telemetry-events

# Check database connectivity
docker exec postgres pg_isready -U telemetry

# Query recent aggregates
docker exec postgres psql -U telemetry -d telemetry -c \
  "SELECT COUNT(*) as total FROM agg_1m WHERE window_start_ts > NOW() - INTERVAL '1 hour';"

# Check for errors in events
docker exec postgres psql -U telemetry -d telemetry -c \
  "SELECT COUNT(*) as errors FROM agg_1m WHERE count_error > 0;"
```

## Common Issues

### 1. Services Won't Start

#### Problem: Docker Compose fails to start

**Symptoms**:
```
ERROR: Couldn't connect to Docker daemon
```

**Solution**:
```bash
# Start Docker Desktop (macOS) or Docker service (Linux)
# macOS: Open Docker Desktop application
# Linux:
sudo systemctl start docker
sudo systemctl enable docker

# Verify Docker is running
docker ps
```

#### Problem: Port conflicts

**Symptoms**:
```
ERROR: for postgres Cannot start service postgres: 
driver failed programming external connectivity on endpoint postgres: 
Bind for 0.0.0.0:5432 failed: port is already allocated
```

**Solution**:
```bash
# Find process using the port
lsof -i :5432

# Kill the process or stop the conflicting service
# Then retry
make up
```

#### Problem: Insufficient resources

**Symptoms**:
- Services crash immediately after starting
- Out of memory errors

**Solution**:
```bash
# Increase Docker resources in Docker Desktop:
# Settings → Resources → Advanced
# - CPUs: 4+
# - Memory: 4GB+
# - Swap: 1GB+

# Or reduce services by commenting them out in docker-compose.yml
```

### 2. Database Issues

#### Problem: Migration fails

**Symptoms**:
```
Error: connection refused
Error: migration failed
```

**Solution**:
```bash
# Wait for PostgreSQL to be ready
docker exec postgres pg_isready -U telemetry

# If not ready, check logs
docker-compose logs postgres

# Retry migration
make migrate-up

# If migrations are stuck, reset (WARNING: deletes data)
make migrate-down
make migrate-up
```

#### Problem: "relation does not exist" errors

**Symptoms**:
```
ERROR: relation "agg_1m" does not exist
```

**Solution**:
```bash
# Check if migrations were applied
docker exec postgres psql -U telemetry -d telemetry -c "\dt"

# If tables missing, run migrations
make migrate-up

# Verify tables exist
docker exec postgres psql -U telemetry -d telemetry -c \
  "SELECT table_name FROM information_schema.tables WHERE table_schema='public';"
```

#### Problem: Database connection timeout

**Symptoms**:
```
Error: dial tcp 127.0.0.1:5432: i/o timeout
```

**Solution**:
```bash
# Check PostgreSQL is running
docker ps | grep postgres

# Check PostgreSQL logs
docker-compose logs postgres | tail -50

# Restart PostgreSQL
docker-compose restart postgres

# Check network connectivity
docker network ls
docker network inspect distributed-telemetry-platform_default
```

### 3. NATS JetStream Issues

#### Problem: Stream not found

**Symptoms**:
```
Error: stream not found: telemetry-events
```

**Solution**:
```bash
# List existing streams
docker exec nats nats stream ls

# Create stream manually if needed
docker exec nats nats stream add telemetry-events \
  --subjects "telemetry.events" \
  --storage file \
  --retention limits \
  --max-age 7d

# Verify stream was created
docker exec nats nats stream info telemetry-events
```

#### Problem: High consumer lag

**Symptoms**:
- Grafana shows `telemetry_queue_consumer_lag_messages` > 1000
- Processing delay increases

**Solution**:
```bash
# Check consumer status
docker exec nats nats consumer info telemetry-events aggregator

# Check aggregator is running and not stuck
ps aux | grep aggregator

# Check aggregator logs for errors
# Look for: database timeouts, processing errors

# Scale aggregator (if using multiple instances)
# Start additional aggregator instances
./bin/aggregator &

# Monitor lag reduction
docker exec nats nats consumer info telemetry-events aggregator | grep Lag
```

#### Problem: Messages stuck in pending

**Symptoms**:
```
  Ack Pending: 5,000 messages
```

**Solution**:
```bash
# Check if aggregator is acknowledging messages
# Look for: "ACK sent for message"

# Check for stuck transactions
docker exec postgres psql -U telemetry -d telemetry -c \
  "SELECT pid, state, wait_event_type, query_start, query 
   FROM pg_stat_activity 
   WHERE state != 'idle' 
   AND query_start < NOW() - INTERVAL '5 minutes';"

# If aggregator is stuck, restart it
pkill -f bin/aggregator
./bin/aggregator
```

### 4. Ingest API Issues

#### Problem: Authentication failures

**Symptoms**:
```
HTTP 401 Unauthorized
Error: invalid API token
```

**Solution**:
```bash
# Verify API token in probe configuration
# Default token: "demo-token"

# Check ingest API logs for authentication attempts
# Look for: "Authentication failed"

# Verify token matches between probe and ingest
# Probe: --api-token demo-token
# Ingest: Accepts any token in development mode

# For production, check token validation logic
```

#### Problem: Rate limiting

**Symptoms**:
```
HTTP 429 Too Many Requests
Error: rate limit exceeded
```

**Solution**:
```bash
# Check rate limit metrics
curl http://localhost:8081/metrics | grep rate_limited

# Increase rate limit in ingest configuration
# Or reduce probe frequency
./bin/probe --interval 30s  # Instead of 10s

# For production, implement distributed rate limiting with Redis
```

#### Problem: Events not being published to NATS

**Symptoms**:
- Ingest API receives requests (200 OK)
- No messages in NATS stream
- Aggregator receives no events

**Solution**:
```bash
# Check NATS connection in ingest logs
# Look for: "Connected to NATS" or connection errors

# Verify NATS is accessible
telnet localhost 4222

# Check ingest API NATS configuration
# Default: nats://localhost:4222

# Restart ingest API
pkill -f bin/ingest
./bin/ingest
```

### 5. Probe Issues

#### Problem: DNS resolution failures

**Symptoms**:
```
Error: lookup target.example.com: no such host
Error: DNS resolution failed
```

**Solution**:
```bash
# Test DNS resolution manually
nslookup target.example.com
dig target.example.com

# Check /etc/resolv.conf for DNS servers
cat /etc/resolv.conf

# Try with different target
./bin/probe --target http://8.8.8.8

# Use IP address instead of hostname
./bin/probe --target http://192.168.1.1
```

#### Problem: Connection timeouts

**Symptoms**:
```
Error: dial tcp: i/o timeout
Error: context deadline exceeded
```

**Solution**:
```bash
# Test connectivity manually
curl -v http://target:port

# Check firewall rules
# macOS: System Preferences → Security & Privacy → Firewall
# Linux: sudo iptables -L

# Increase timeout in probe
# (Default: 30 seconds for most operations)

# Check network connectivity
ping target
traceroute target
```

#### Problem: TLS certificate errors

**Symptoms**:
```
Error: x509: certificate signed by unknown authority
Error: TLS handshake failed
```

**Solution**:
```bash
# Test TLS connection manually
openssl s_client -connect target:443

# For self-signed certificates, skip verification (dev only)
# Add flag to probe: --insecure-skip-verify

# For production, ensure proper CA certificates
# Update system CA bundle or provide custom CA file
```

### 6. Aggregator Issues

#### Problem: Duplicate events not deduplicated

**Symptoms**:
- Same event processed multiple times
- Aggregate counters inflated

**Solution**:
```bash
# Check deduplication table
docker exec postgres psql -U telemetry -d telemetry -c \
  "SELECT COUNT(*) FROM events_seen;"

# Check for UUID collisions (should be 0)
docker exec postgres psql -U telemetry -d telemetry -c \
  "SELECT event_id, COUNT(*) FROM events_seen GROUP BY event_id HAVING COUNT(*) > 1;"

# Verify transaction isolation
# Check aggregator logs for "Event already processed"

# If deduplication failing, check:
# 1. Event ID generation (must be deterministic)
# 2. Transaction commit/rollback logic
# 3. Database constraints on events_seen table
```

#### Problem: Percentile calculations incorrect

**Symptoms**:
- P50 > P95
- Negative values
- Unrealistic outliers

**Solution**:
```bash
# Check raw samples in aggregator logs
# Enable debug logging: --log-level=debug

# Verify sample collection
# Look for: "Collected N samples for window"

# Check for data type issues
docker exec postgres psql -U telemetry -d telemetry -c \
  "SELECT client_id, target, window_start_ts, 
          dns_p50, dns_p95, tcp_p50, tcp_p95, ttfb_p50, ttfb_p95
   FROM agg_1m 
   WHERE dns_p50 > dns_p95 OR tcp_p50 > tcp_p95 OR ttfb_p50 > ttfb_p95
   LIMIT 10;"

# If issue persists, check percentile calculation logic
# Ensure samples are sorted before calculation
```

#### Problem: Late events rejected

**Symptoms**:
```
Warning: Event too late, dropping
Window already flushed
```

**Solution**:
```bash
# Check clock skew between probe and aggregator
date  # On both machines

# Increase late event tolerance (currently 2 minutes)
# Modify aggregator configuration

# Sync time with NTP
# macOS: System Preferences → Date & Time → Set automatically
# Linux: 
sudo ntpdate -s time.nist.gov
# Or use chronyd/ntpd

# Check event recv_ts_ms vs ts_ms in database
docker exec postgres psql -U telemetry -d telemetry -c \
  "SELECT * FROM agg_1m WHERE window_start_ts > NOW() - INTERVAL '10 minutes' LIMIT 5;"
```

### 7. Diagnoser Issues

#### Problem: No diagnosis labels

**Symptoms**:
- `diagnosis_label` column is NULL for all records
- Diagnoser appears to be running but not labeling

**Solution**:
```bash
# Check diagnoser logs
# Look for: "Running diagnosis" and "Analyzed N aggregates"

# Verify sufficient data for baseline
# Need at least 10 windows (10 minutes) per client-target pair
docker exec postgres psql -U telemetry -d telemetry -c \
  "SELECT client_id, target, COUNT(*) as window_count 
   FROM agg_1m 
   GROUP BY client_id, target;"

# Check if aggregates have percentile data
docker exec postgres psql -U telemetry -d telemetry -c \
  "SELECT COUNT(*) FROM agg_1m WHERE ttfb_p95 IS NOT NULL;"

# Manually run diagnosis for debugging
./bin/diagnoser --run-once
```

#### Problem: Incorrect diagnosis labels

**Symptoms**:
- Healthy connections labeled as problematic
- Vice versa

**Solution**:
```bash
# Review diagnosis thresholds
# DNS-bound: DNS p95 ≥ 60% of total latency AND exceeds baseline by 50%
# Server-bound: TTFB p95 exceeds baseline by 2σ or 100%
# Throughput-bound: Throughput p50 drops ≥30% below baseline

# Check baseline calculation
# Uses last 10 windows (simple moving average)

# Query aggregate data manually
docker exec postgres psql -U telemetry -d telemetry -c \
  "SELECT client_id, target, window_start_ts, 
          dns_p95, tcp_p95, tls_p95, ttfb_p95, throughput_p50, diagnosis_label
   FROM agg_1m 
   WHERE diagnosis_label IS NOT NULL
   ORDER BY window_start_ts DESC 
   LIMIT 20;"

# Verify diagnosis logic matches expected behavior
# Check diagnoser source code if needed
```

### 8. Monitoring & Observability Issues

#### Problem: Grafana dashboards show no data

**Symptoms**:
- Empty panels in Grafana
- "No data" messages

**Solution**:
```bash
# Check Prometheus is scraping metrics
curl http://localhost:9090/api/v1/targets

# Verify metrics are being exported
curl http://localhost:8081/metrics  # Ingest API
curl http://localhost:8082/metrics  # Aggregator
curl http://localhost:8083/metrics  # Diagnoser

# Check Grafana datasource configuration
# Grafana → Configuration → Data Sources → Prometheus
# URL should be: http://prometheus:9090

# Reimport dashboards if needed
# Grafana → Dashboards → Import → Upload JSON file

# Check time range in Grafana (top right)
# Ensure it matches when you ran the demo
```

#### Problem: Jaeger shows no traces

**Symptoms**:
- Empty trace view in Jaeger UI
- No spans captured

**Solution**:
```bash
# Check OpenTelemetry Collector is running
docker ps | grep otel-collector

# Verify collector configuration
cat config/otel-collector.yml

# Check if services are exporting traces
# Look for: "Trace exporter initialized" in logs

# Test Jaeger API
curl http://localhost:16686/api/services

# If empty, check trace sampling rate
# Ensure traces are being generated for requests
```

#### Problem: High cardinality causing Prometheus issues

**Symptoms**:
- Prometheus running slow
- "out of memory" errors
- High CPU usage

**Solution**:
```bash
# Check cardinality
curl http://localhost:9090/api/v1/status/tsdb | jq '.data.numSeries'

# If >100k series, reduce cardinality:
# 1. Hash client_id and target in metric labels
# 2. Use fewer label values
# 3. Implement metric retention policies

# Restart Prometheus with more resources
# Edit docker-compose.yml:
#   prometheus:
#     environment:
#       - PROMETHEUS_RETENTION=7d
#     deploy:
#       resources:
#         limits:
#           memory: 2G
```

### 9. Performance Issues

#### Problem: Slow query performance

**Symptoms**:
- Aggregator taking seconds to write
- Database CPU at 100%
- Slow Grafana dashboard loading

**Solution**:
```bash
# Check for missing indexes
docker exec postgres psql -U telemetry -d telemetry -c \
  "SELECT schemaname, tablename, indexname 
   FROM pg_indexes 
   WHERE schemaname = 'public';"

# Analyze slow queries
docker exec postgres psql -U telemetry -d telemetry -c \
  "SELECT query, mean_exec_time, calls 
   FROM pg_stat_statements 
   ORDER BY mean_exec_time DESC 
   LIMIT 10;"

# Run VACUUM and ANALYZE
docker exec postgres psql -U telemetry -d telemetry -c \
  "VACUUM ANALYZE;"

# Check table bloat
./bin/cleanup --health-check

# Consider partitioning large tables
# See DATABASE_MAINTENANCE.md
```

#### Problem: High memory usage

**Symptoms**:
- Aggregator using >1GB RAM
- Out of memory crashes

**Solution**:
```bash
# Check memory usage
docker stats

# Reduce in-memory sample buffer
# Currently stores up to 10k samples per window per client-target
# Consider flushing more frequently or downsampling earlier

# Monitor Go heap
# Enable pprof endpoint and analyze
curl http://localhost:6060/debug/pprof/heap > heap.prof
go tool pprof -http=:8080 heap.prof

# Increase container memory limits
# Edit docker-compose.yml or deployment config
```

### 10. Data Quality Issues

#### Problem: Missing data points

**Symptoms**:
- Gaps in time-series charts
- Expected windows missing from agg_1m

**Solution**:
```bash
# Check for probe failures
# Review probe logs for errors

# Check for aggregator processing errors
# Look for: "Failed to process event"

# Query for gaps
docker exec postgres psql -U telemetry -d telemetry -c \
  "SELECT generate_series(
     (SELECT MIN(window_start_ts) FROM agg_1m),
     (SELECT MAX(window_start_ts) FROM agg_1m),
     interval '1 minute'
   ) AS expected_window
   EXCEPT
   SELECT window_start_ts FROM agg_1m
   ORDER BY expected_window DESC
   LIMIT 20;"

# Check NATS for message loss
docker exec nats nats stream info telemetry-events
# Look for: Stream-Lost messages (should be 0)
```

#### Problem: Inconsistent aggregates

**Symptoms**:
- Sum of per-client counts ≠ total count
- Percentiles don't match raw data

**Solution**:
```bash
# Verify data integrity
docker exec postgres psql -U telemetry -d telemetry -c \
  "SELECT client_id, target, window_start_ts, 
          count_total, count_success, count_error
   FROM agg_1m
   WHERE count_total != (count_success + count_error)
   LIMIT 10;"

# Check for race conditions in aggregator
# Ensure proper locking for concurrent window updates

# Verify transaction isolation level
docker exec postgres psql -U telemetry -d telemetry -c \
  "SHOW default_transaction_isolation;"

# Should be "read committed" or higher
```

## Getting Help

### Log Collection

When reporting issues, collect logs:

```bash
# Infrastructure logs
docker-compose logs > infrastructure.log

# Application logs
# Redirect stdout/stderr when starting services:
./bin/ingest > ingest.log 2>&1 &
./bin/aggregator > aggregator.log 2>&1 &
./bin/diagnoser > diagnoser.log 2>&1 &
./bin/probe > probe.log 2>&1 &

# System information
docker version > system-info.txt
docker-compose version >> system-info.txt
go version >> system-info.txt
uname -a >> system-info.txt
```

### Health Check Script

Run comprehensive health check:

```bash
./scripts/validate-setup.sh

# Or manually check each component
make db-health
docker exec nats nats stream info telemetry-events
curl http://localhost:8081/health
curl http://localhost:9090/-/healthy
curl http://localhost:3000/api/health
```

### Community Support

- GitHub Issues: https://github.com/rahulgh33/Distributed-Telemetry-Platform/issues
- Documentation: See `docs/` directory
- Examples: See `scripts/` directory for test scripts

### Additional Resources

- [Demo Guide](DEMO.md) - Step-by-step walkthrough
- [Database Maintenance](DATABASE_MAINTENANCE.md) - Database operations
- [Performance Tuning](PERFORMANCE.md) - Optimization tips
- [Deployment Guide](DEPLOYMENT.md) - Production deployment
- [API Documentation](API.md) - API reference
