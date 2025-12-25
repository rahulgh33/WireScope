# Demo Guide - Network QoE Telemetry Platform

This guide walks through a complete demonstration of the Network QoE Telemetry Platform, from setup to viewing results in Grafana.

## Prerequisites

Ensure you have completed the [Quick Start](../README.md#quick-start) setup:
- ✅ All services running (`make up`)
- ✅ Database migrations applied (`make migrate`)
- ✅ Binaries built (`make build`)

## Demo Scenario

We'll demonstrate:
1. Starting all platform services
2. Running probe agents to measure network performance
3. Watching data flow through the pipeline
4. Viewing aggregated metrics in Grafana
5. Observing automated diagnosis

## Step 1: Start All Services

### 1.1 Start Infrastructure

```bash
# Start PostgreSQL, NATS, Prometheus, Grafana, Jaeger
make up

# Verify services are healthy
docker-compose ps
```

Expected output: All services should show "Up" status.

### 1.2 Run Ingest API

In a new terminal:

```bash
./bin/ingest
```

Expected output:
```
2025/12/23 10:00:00 Starting ingest API on :8081
2025/12/23 10:00:00 Metrics available at /metrics
2025/12/23 10:00:00 Health check at /health
```

### 1.3 Run Aggregator

In another new terminal:

```bash
./bin/aggregator
```

Expected output:
```
2025/12/23 10:00:01 Starting aggregator consumer
2025/12/23 10:00:01 Connected to NATS at nats://localhost:4222
2025/12/23 10:00:01 Connected to database telemetry@localhost:5432
2025/12/23 10:00:01 Listening for events on stream: telemetry-events
```

### 1.4 Run Diagnoser

In another new terminal:

```bash
./bin/diagnoser
```

Expected output:
```
2025/12/23 10:00:02 Starting diagnoser service
2025/12/23 10:00:02 Connected to database
2025/12/23 10:00:02 Running diagnosis every 60 seconds
```

## Step 2: Generate Telemetry Data

### 2.1 Run a Single Probe Measurement

Test with a single measurement to the test target server:

```bash
./bin/probe \
  --target http://localhost:8080 \
  --api-url http://localhost:8081/api/v1/events \
  --api-token demo-token \
  --client-id demo-client-001 \
  --interval 10s \
  --count 1
```

Expected output:
```
2025/12/23 10:00:05 Probe agent starting
2025/12/23 10:00:05 Client ID: demo-client-001
2025/12/23 10:00:05 Target: http://localhost:8080
2025/12/23 10:00:05 Measuring network performance...
2025/12/23 10:00:05 DNS: 2.3ms, TCP: 0.5ms, TLS: N/A, TTFB: 1.2ms, Throughput: 125.4 MB/s
2025/12/23 10:00:05 Event sent successfully
```

### 2.2 Run Continuous Probes

Start multiple probe agents for continuous data collection:

```bash
# Probe 1: Fast target (localhost)
./bin/probe \
  --target http://localhost:8080 \
  --api-url http://localhost:8081/api/v1/events \
  --api-token demo-token \
  --client-id client-fast \
  --interval 10s &

# Probe 2: Slow target (simulated delay)
./bin/probe \
  --target http://localhost:8080/slow?ms=500 \
  --api-url http://localhost:8081/api/v1/events \
  --api-token demo-token \
  --client-id client-slow \
  --interval 10s &

# Probe 3: External target (Google)
./bin/probe \
  --target https://www.google.com \
  --api-url http://localhost:8081/api/v1/events \
  --api-token demo-token \
  --client-id client-google \
  --interval 10s &
```

Let these run for 2-3 minutes to collect sufficient data.

## Step 3: Observe Data Flow

### 3.1 Watch Ingest API Logs

In the ingest API terminal, you should see:

```
2025/12/23 10:01:00 Received event from client-fast for http://localhost:8080
2025/12/23 10:01:00 Event published to NATS stream
2025/12/23 10:01:10 Received event from client-slow for http://localhost:8080/slow?ms=500
2025/12/23 10:01:10 Event published to NATS stream
```

### 3.2 Watch Aggregator Logs

In the aggregator terminal, you should see:

```
2025/12/23 10:01:00 Processing event: client-fast, http://localhost:8080
2025/12/23 10:01:00 Window: 2025-12-23 10:01:00, Samples: 1
2025/12/23 10:01:10 Processing event: client-slow, http://localhost:8080/slow?ms=500
2025/12/23 10:01:10 Window: 2025-12-23 10:01:00, Samples: 2
2025/12/23 10:02:00 Flushing window: 2025-12-23 10:01:00
2025/12/23 10:02:00 Calculated P50 TTFB: 250.5ms, P95 TTFB: 500.2ms
2025/12/23 10:02:00 Aggregate saved for client-fast, http://localhost:8080
```

### 3.3 Check NATS Stream Stats

```bash
docker exec -it nats nats stream info telemetry-events
```

Expected output shows messages being processed:
```
Statistics:

             Messages: 120
                Bytes: 15.2 KB
             FirstSeq: 1
              LastSeq: 120
           Num Consumers: 1
```

### 3.4 Query Database

Check the aggregates table:

```bash
docker exec -it postgres psql -U telemetry -d telemetry -c \
  "SELECT client_id, target, window_start_ts, count_total, ttfb_p95, diagnosis_label 
   FROM agg_1m 
   ORDER BY window_start_ts DESC 
   LIMIT 10;"
```

Expected output:
```
   client_id   |              target              |   window_start_ts   | count_total | ttfb_p95 | diagnosis_label
---------------+----------------------------------+---------------------+-------------+----------+------------------
 client-fast   | http://localhost:8080            | 2025-12-23 10:02:00 |           6 |     1.5  | healthy
 client-slow   | http://localhost:8080/slow?ms=500| 2025-12-23 10:02:00 |           6 |   502.3  | server-bound
 client-google | https://www.google.com           | 2025-12-23 10:02:00 |           6 |    45.2  | healthy
```

## Step 4: View Results in Grafana

### 4.1 Access Grafana

Open http://localhost:3000 in your browser:
- Username: `admin`
- Password: `admin`

### 4.2 Network Performance Dashboard

Navigate to **Dashboards** → **Network QoE** → **Network Performance**

You should see:

1. **Overview Row**:
   - Total events processed
   - Active clients
   - Active targets
   - Current error rate

2. **Latency Trends**:
   - DNS P50/P95 over time
   - TCP P50/P95 over time
   - TLS P50/P95 over time
   - TTFB P50/P95 over time

3. **Throughput Trends**:
   - Throughput P50/P95 over time
   - Per-client throughput comparison

4. **Error Analysis**:
   - Error count by stage (DNS, TCP, TLS, HTTP, Throughput)
   - Error rate percentage
   - Per-client error breakdown

5. **Diagnosis Distribution**:
   - Pie chart showing diagnosis labels
   - Count of healthy vs problematic windows

### 4.3 Platform Health Dashboard

Navigate to **Dashboards** → **Network QoE** → **Platform Health**

You should see:

1. **Ingest API Metrics**:
   - Request rate
   - Error rate
   - Response times
   - Rate limiting events

2. **Queue Metrics**:
   - NATS consumer lag
   - Messages pending acknowledgment
   - Processing rate

3. **Aggregator Metrics**:
   - Events processed per second
   - Duplicate events detected
   - Window flush operations
   - Database transaction times

4. **Database Metrics**:
   - Connection pool usage
   - Query duration
   - Table sizes
   - Transaction throughput

## Step 5: Automated Diagnosis

### 5.1 Wait for Diagnosis

The diagnoser runs every 60 seconds. After a few minutes, check the logs:

```
2025/12/23 10:03:00 Running diagnosis for window: 2025-12-23 10:02:00
2025/12/23 10:03:00 Analyzed 3 aggregates
2025/12/23 10:03:00 Found 1 issue: client-slow → server-bound
2025/12/23 10:03:00 Diagnosis complete
```

### 5.2 View Diagnosis in Grafana

In the Network Performance dashboard, the "Diagnosis Distribution" panel will show:
- `healthy`: 2 windows (66%)
- `server-bound`: 1 window (33%)

### 5.3 Query Diagnosis Results

```bash
docker exec -it postgres psql -U telemetry -d telemetry -c \
  "SELECT client_id, target, window_start_ts, diagnosis_label, ttfb_p95 
   FROM agg_1m 
   WHERE diagnosis_label IS NOT NULL 
   ORDER BY window_start_ts DESC 
   LIMIT 5;"
```

## Step 6: Test Failure Scenarios

### 6.1 Test Duplicate Handling

Send the same event twice:

```bash
# First send
./bin/probe --target http://localhost:8080 --client-id test-dup --count 1

# Send again (will be deduplicated)
./bin/probe --target http://localhost:8080 --client-id test-dup --count 1
```

Check aggregator logs - you should see:
```
2025/12/23 10:05:00 Event already processed (duplicate): <uuid>
2025/12/23 10:05:00 Skipping duplicate event
```

### 6.2 Test Backpressure

Generate high load:

```bash
# Start 10 probes with 1-second intervals
for i in {1..10}; do
  ./bin/probe \
    --target http://localhost:8080 \
    --client-id client-load-$i \
    --interval 1s &
done
```

Monitor queue lag in Grafana or Prometheus:
```
telemetry_queue_consumer_lag_messages
```

### 6.3 Test Aggregator Restart

1. Stop the aggregator (Ctrl+C in its terminal)
2. Wait 30 seconds
3. Restart: `./bin/aggregator`

The aggregator should:
- Resume from last acknowledged position
- Continue processing without data loss
- Not duplicate already-processed events

## Step 7: Cleanup

Stop all probes:

```bash
# Kill all probe processes
pkill -f "bin/probe"
```

Stop services:

```bash
# Keep infrastructure running
# Just stop the application services with Ctrl+C in each terminal

# Or stop everything
make down
```

## Expected Results Summary

After running this demo for 5-10 minutes:

| Metric | Expected Value |
|--------|----------------|
| Events Collected | 100-300 |
| Aggregates Created | 10-30 (one per minute per client-target pair) |
| Duplicate Events | 0 (unless explicitly tested) |
| Processing Delay | <1 second (P95) |
| Error Rate | <1% (for localhost targets) |
| Diagnosis Accuracy | 100% (server-bound diagnosis for /slow endpoint) |

## Troubleshooting

### No Data in Grafana

1. Check probe is running: `ps aux | grep probe`
2. Check ingest API logs for received events
3. Check NATS stream: `docker exec nats nats stream info telemetry-events`
4. Check aggregator is consuming messages
5. Query database directly to verify aggregates exist

### High Error Rate

1. Check target server is accessible: `curl http://localhost:8080/health`
2. Check probe logs for error messages
3. Verify network connectivity
4. Check for DNS resolution issues

### Aggregator Not Processing

1. Verify NATS connection: Check aggregator logs
2. Check database connection: `docker exec postgres pg_isready`
3. Verify stream exists: `docker exec nats nats stream ls`
4. Check consumer lag in Prometheus

### Diagnosis Not Running

1. Check diagnoser logs
2. Verify sufficient data (need at least 10 windows for baseline)
3. Query agg_1m table directly
4. Check diagnoser is connecting to database

## Next Steps

- Explore [Performance Tuning](PERFORMANCE.md) for optimization
- Review [Troubleshooting Guide](TROUBLESHOOTING.md) for common issues
- See [Deployment Guide](DEPLOYMENT.md) for production deployment
- Check [API Documentation](API.md) for integration details
