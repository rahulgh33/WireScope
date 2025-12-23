# Milestone B Demo: Reliability Validation

This directory contains automated test scripts to validate the reliability features implemented in Milestone B.

## Prerequisites

### Required Services
Ensure the following services are running:
- PostgreSQL (port 5432)
- NATS with JetStream (port 4222)
- Ingest API (port 8080)
- Aggregator (with metrics on port 9090)

### Required Tools
- `curl` - HTTP client
- `psql` - PostgreSQL client
- `jq` - JSON processor
- `nats` CLI (optional, for DLQ test) - Install: `go install github.com/nats-io/natscli/nats@latest`

### Starting Services

```bash
# Start infrastructure
docker-compose up -d postgres nats

# Start ingest API
./bin/ingest \
  --port=8080 \
  --nats-url=nats://localhost:4222 \
  --api-tokens=test-token \
  --rate-limit=100 \
  --rate-limit-burst=20

# Start aggregator
./bin/aggregator \
  --nats-url=nats://localhost:4222 \
  --db-host=localhost \
  --db-port=5432 \
  --db-name=telemetry \
  --db-user=telemetry \
  --db-password=telemetry \
  --metrics-port=9090
```

## Test Scripts

### 1. Duplicate Event Handling (`test-duplicate-handling.sh`)

**Tests:** Exactly-once aggregate effects via deduplication

**What it does:**
- Sends the same event multiple times
- Verifies aggregate count remains 1
- Confirms events_seen table has single entry

**Run:**
```bash
./scripts/test-duplicate-handling.sh
```

**Expected outcome:**
- ✓ All duplicate events accepted (HTTP 202)
- ✓ Aggregate `count_total` remains 1
- ✓ Database shows exactly one event in events_seen

**Key validation:**
- Requirement 3.3: Exactly-once aggregate effects
- Requirement 4.1: Transactional consistency

---

### 2. Aggregator Restart (`test-aggregator-restart.sh`)

**Tests:** Safe continuation after aggregator restart

**What it does:**
- Sends events while aggregator is running
- Stops aggregator
- Sends more events (queued in NATS)
- Restarts aggregator
- Verifies all events are processed

**Run:**
```bash
./scripts/test-aggregator-restart.sh
```

**Manual steps required:**
1. Script will prompt you to stop the aggregator (Ctrl+C or `pkill -f bin/aggregator`)
2. Script will prompt you to restart the aggregator

**Expected outcome:**
- ✓ Events processed before shutdown
- ✓ Events queued during downtime
- ✓ All queued events processed after restart
- ✓ No data loss
- ✓ Deduplication works after restart

**Key validation:**
- Requirement 3.1: At-least-once delivery
- Requirement 8.4: Durable NATS JetStream queues
- Requirement 8.5: Consumer durability

---

### 3. Backpressure Mechanisms (`test-backpressure.sh`)

**Tests:** Rate limiting and queue backpressure

**What it does:**
- Sends events at normal rate (should succeed)
- Bursts 150 events rapidly (should trigger rate limiting)
- Waits for token bucket to refill
- Tests recovery

**Run:**
```bash
./scripts/test-backpressure.sh
```

**Expected outcome:**
- ✓ Normal rate: all events accepted (HTTP 202)
- ✓ Burst rate: some events rate-limited (HTTP 429)
- ✓ Recovery: rate limiter allows traffic after cooldown
- ✓ Queue lag metrics available

**Key validation:**
- Requirement 8.2: Bounded local queue in probe
- Requirement 8.3: Rate limiting per client_id
- Requirement 8.4: In-flight message limits

---

### 4. DLQ Poison Messages (`test-dlq-routing.sh`)

**Tests:** Dead letter queue routing for unprocessable events

**What it does:**
- Publishes malformed events directly to NATS
- Waits for max delivery attempts (5 retries)
- Verifies events are routed to DLQ
- Checks DLQ metrics

**Run:**
```bash
./scripts/test-dlq-routing.sh
```

**Note:** Requires `nats` CLI tool. Install with:
```bash
go install github.com/nats-io/natscli/nats@latest
```

**Expected outcome:**
- ✓ Malformed events published to main stream
- ✓ Aggregator retries up to MaxDeliver (5 times)
- ✓ Events routed to DLQ after max retries
- ✓ DLQ metrics counter increases
- ✓ System continues processing valid events

**Key validation:**
- Requirement 8.4: DLQ for poison messages
- Requirement 8.1: Graceful error handling

---

## Running All Tests

Run all tests in sequence:

```bash
#!/bin/bash
echo "Running Milestone B Demo Tests..."
echo ""

./scripts/test-duplicate-handling.sh
echo ""
echo "===================================="
echo ""

./scripts/test-backpressure.sh
echo ""
echo "===================================="
echo ""

# Interactive test - requires manual steps
./scripts/test-aggregator-restart.sh
echo ""
echo "===================================="
echo ""

./scripts/test-dlq-routing.sh
echo ""
echo "===================================="
echo ""

echo "All Milestone B tests complete!"
```

## Monitoring During Tests

### Check Metrics

**Ingest API metrics:**
```bash
curl http://localhost:8080/metrics | grep ingest_requests
```

**Aggregator metrics:**
```bash
curl http://localhost:9090/metrics | grep -E '(events_processed|queue_lag|dedup_rate|dlq)'
```

### Check Database

**Count aggregates:**
```bash
psql -h localhost -U telemetry -d telemetry -c "SELECT COUNT(*) FROM agg_1m;"
```

**Count events seen:**
```bash
psql -h localhost -U telemetry -d telemetry -c "SELECT COUNT(*) FROM events_seen;"
```

**View recent aggregates:**
```bash
psql -h localhost -U telemetry -d telemetry -c "
  SELECT client_id, target, count_total, count_success, count_error 
  FROM agg_1m 
  ORDER BY window_start_ts DESC 
  LIMIT 10;
"
```

### Check NATS Queue

**Stream info:**
```bash
nats stream info telemetry-events
```

**DLQ info:**
```bash
nats stream info telemetry-events-dlq
```

## Troubleshooting

### Tests fail with connection errors

**Issue:** Cannot connect to services

**Solution:**
```bash
# Check services are running
docker-compose ps
netstat -an | grep -E '(4222|5432|8080|9090)'

# Check ingest API
curl http://localhost:8080/health

# Check aggregator metrics endpoint
curl http://localhost:9090/metrics
```

### DLQ test times out

**Issue:** Poison messages not reaching DLQ

**Possible causes:**
1. MaxDeliver setting too high (check aggregator config)
2. AckWait too long (default 30s per retry = 150s total)
3. Aggregator not processing messages

**Solution:**
Check aggregator logs for:
```bash
tail -f <aggregator-log> | grep -E '(DLQ|exceeded max|unmarshal error)'
```

### Rate limiting not triggering

**Issue:** Backpressure test doesn't see 429 responses

**Possible causes:**
1. Rate limit configured too high
2. Burst size too large
3. Token bucket not working

**Solution:**
Restart ingest API with lower limits:
```bash
./bin/ingest --rate-limit=10 --rate-limit-burst=5 ...
```

## Success Criteria

All tests should pass with:
- ✓ Duplicate events properly deduplicated
- ✓ Aggregator restarts without data loss
- ✓ Rate limiting activates under burst traffic
- ✓ Poison messages routed to DLQ
- ✓ All metrics updating correctly

## Next Steps

After validating Milestone B:
1. Review metrics in Grafana (if configured)
2. Proceed to Milestone C: Observability & Operations
3. Implement comprehensive monitoring and alerting
