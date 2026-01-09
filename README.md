# WireScope

A distributed network telemetry platform for measuring and analyzing network performance from multiple locations.

## What it does

Deploy lightweight probes anywhere (home, office, cloud, remote sites) that continuously measure network performance and send data to a central server. The server aggregates metrics, identifies bottlenecks, and presents everything in real-time dashboards.

**Use cases:**
- Monitor ISP quality from different geographic locations
- Track corporate network performance across branch offices
- Debug connectivity issues with hard data (not just "it's slow")
- Understand latency breakdown (DNS, TCP, TLS, HTTP response time)
- Compare network performance between providers or regions

## Architecture

```
Probe → Ingest API (8081) → NATS Queue → Aggregator → PostgreSQL
                                              ↓
                                    Web UI / Grafana Dashboards
```

**Probe**: Measures DNS resolution, TCP/TLS handshake timing, HTTP latency, and throughput  
**Ingest API**: HTTP endpoint that receives probe measurements with authentication  
**NATS**: Durable message queue ensuring no data loss  
**Aggregator**: Calculates 1-minute window statistics (P50/P95 percentiles), deduplicates events  
**PostgreSQL**: Stores aggregated time-series data  
**Web UI**: Real-time dashboard with charts and AI-powered queries  
**Grafana**: Advanced visualization and alerting

## Quick Start

See [QUICKSTART.md](QUICKSTART.md) for detailed setup instructions.

### Prerequisites

- **macOS/Linux** (tested on macOS with M1/M2)
- **Docker Desktop** installed and running
- **Go 1.21+** for building from source
- **No local PostgreSQL** on port 5432 (conflicts with Docker)

### 1. Initial Setup (5 minutes)

```bash
# Clone repository
git clone https://github.com/rahulgh33/WireScope.git
cd WireScope

# Stop any local postgres that might conflict
brew services stop postgresql@16 postgresql@15 postgresql 2>/dev/null || true

# Start Docker services (postgres, nats)
docker-compose up -d

# Build Go binaries
make build

# Run database migrations
make migrate

# Start all services
./scripts/start-services.sh
```

### 2. Verify Services Running

```bash
./scripts/status-services.sh
```

You should see:
- ✓ PostgreSQL, NATS running
- ✓ Ingest API, Aggregator, AI Agent running

### 3. Start Local Test Probes

```bash
# Probe 1 - monitoring example.com
./bin/probe \
  --ingest-url http://localhost:8081/events \
  --client-id local-test-1 \
  --target http://example.com \
  --interval 30s \
  --tracing-enabled=false &

# Probe 2 - monitoring another target
./bin/probe \
  --ingest-url http://localhost:8081/events \
  --client-id local-test-2 \
  --target http://info.cern.ch \
  --interval 30s \
  --tracing-enabled=false &
```

### 4. View Data

**Database (raw check):**
```bash
docker exec wirescope-postgres-1 psql -U telemetry -d telemetry \
  -c "SELECT client_id, target, count_total FROM agg_1m ORDER BY window_start_ts DESC LIMIT 10;"
```

**Web UI:** http://localhost:5173 (start with `cd web && npm run dev`)  
**Grafana:** http://localhost:3000 (admin/admin)  
**Prometheus:** http://localhost:9090

Data appears within ~60 seconds (aggregation window).

## Remote Probe Deployment

### Same Network (Local IP)

On your Mac, find your IP:
```bash
ifconfig | grep "inet " | grep -v 127.0.0.1
```

On remote machine:
```bash
wget https://github.com/rahulgh33/WireScope/releases/latest/download/probe-linux-amd64 -O probe
chmod +x probe

./probe \
  --ingest-url http://YOUR_MAC_IP:8081/events \
  --client-id remote-office-1 \
  --target https://google.com \
  --interval 60s \
  --tracing-enabled=false
```

### Different Network (Internet)

Use ngrok or cloud deployment:
```bash
# On your Mac
ngrok http 8081

# On remote machine, use the ngrok URL
./probe \
  --ingest-url https://YOUR-NGROK-URL.ngrok-free.dev/events \
  --client-id remote-site-1 \
  --target https://google.com \
  --interval 60s \
  --tracing-enabled=false
```

**Note:** For production, deploy the server to cloud (see [CLOUD_DEPLOYMENT.md](CLOUD_DEPLOYMENT.md)).

## Building from source

```bash
# Build everything
make build

# Or individual components
go build -o bin/probe ./cmd/probe
go build -o bin/ingest ./cmd/ingest
go build -o bin/aggregator ./cmd/aggregator
go build -o bin/diagnoser ./cmd/diagnoser
go build -o bin/ai-agent ./cmd/ai-agent
```

## Configuration

### Probe Options
- `--ingest-url`: Ingest API endpoint (required)
- `--api-token`: Authentication token (optional if server auth is disabled)
- `--target`: URL to monitor (required, e.g., http://example.com)
- `--client-id`: Unique identifier for this probe (auto-generated if omitted)
- `--interval`: Time between measurements (default: 60s)
- `--tracing-enabled`: Enable OpenTelemetry tracing (default: true)
- `--interface`: Network type: wifi, ethernet, cellular (default: ethernet)
- `--vpn`: Set true if using VPN (default: false)

### Ingest API Environment Variables
- `DB_USER`, `DB_PASSWORD`, `DB_HOST`, `DB_NAME`, `DB_PORT`: Database connection
- `NATS_URL`: NATS server URL (default: nats://localhost:4222)
- `API_TOKENS`: Comma-separated valid tokens (leave empty to disable auth)

### Aggregator Environment Variables
- `DB_USER`, `DB_PASSWORD`, `DB_HOST`, `DB_NAME`, `DB_PORT`: Database connection
- `NATS_URL`: NATS server URL

### Command Line Flags
All services support `--help` to see available options:
```bash
./bin/probe --help
./bin/ingest --help
./bin/aggregator --help
```

## Common Issues & Solutions

### "role telemetry does not exist"

**Cause:** Local Postgres running on port 5432 conflicts with Docker container  
**Fix:** 
```bash
brew services stop postgresql@16
brew services stop postgresql@15
# Then restart: docker-compose down -v && docker-compose up -d
make migrate
```

### No data appearing in database

**Cause:** Probe not sending events (missing --api-token or wrong URL)  
**Check:**
```bash
# Probe logs should show "Successfully sent event..."
tail -f logs/probe-*.log

# Ingest should receive requests
curl http://localhost:8081/metrics | grep ingest_requests_total

# Check aggregator is running
ps aux | grep aggregator
tail -f logs/aggregator.log
```

### Port already in use

**Cause:** Service already running or port conflict  
**Fix:**
```bash
# Stop everything
./scripts/stop-services.sh
docker-compose down

# Check what's using the port
lsof -i :8081  # or whichever port

# Restart
docker-compose up -d
./scripts/start-services.sh
```

## Running without Docker

```bash
# Start PostgreSQL and NATS somehow (or use docker-compose for just those)
docker-compose up -d postgres nats

# Run database migrations
make migrate-up

# Start services
./bin/ingest &
./bin/aggregator &
./bin/diagnoser &
./bin/ai-agent &

# Start web UI
cd web && npm install && npm run dev
```

## How it works

### Exactly-once processing

Events have UUIDs. The aggregator writes them to `events_seen` table before updating aggregates. If it crashes and reprocesses the same event, the INSERT fails (primary key conflict) and the aggregate doesn't change.

### Time windows

Aggregates are per-minute windows. An event at `2024-01-15T10:23:45Z` goes into the `2024-01-15T10:23:00Z` window.

Late events (up to 2 minutes old) are accepted and update the existing window.

### Percentiles

Stores raw samples in memory during each window (up to 10k samples). At window end, sorts them and picks P50/P95. If over 10k samples, uniformly downsamples.

### Diagnosis

Runs every minute. Looks at the last 10 windows to establish a baseline. Flags problems:
- **DNS-bound**: DNS time is >60% of total latency
- **Handshake-bound**: TCP/TLS time jumped 2σ above baseline  
- **Server-bound**: TTFB increased but connection times are normal
- **Throughput-bound**: Download speed dropped >30%

### Backpressure

- Probe: bounded queue (100 events), exponential backoff if ingest API is down
- Ingest API: rate limiting per client (token bucket)
- Aggregator: limits in-flight messages (100), only ACKs after DB commit

### Dead letter queue

If an event fails processing 5 times, it goes to DLQ. Check there for poison messages.

## API

### Ingest
```
POST /events
Authorization: Bearer {token}
Content-Type: application/json

{
  "event_id": "uuid",
  "client_id": "probe-1",
  "target": "https://example.com",
  "ts_ms": 1705330425000,
  "dns_ms": 15.2,
  "tcp_ms": 45.6,
  "tls_ms": 89.3,
  "ttfb_ms": 123.4,
  "total_ms": 156.7,
  "throughput_mbps": 85.2,
  "error_stage": null
}
```

### AI Agent
```
POST /api/v1/ai/query
Content-Type: application/json

{
  "query": "Which probes had high latency in the last hour?"
}
```

## Web UI

Login: `admin` / `admin123` (change in production)

Features:
- Real-time dashboard with charts
- Client/target filtering
- AI chat for querying data
- Dark mode
- Admin panel for managing probes

## Testing

```bash
# Unit tests
make test

# Integration tests (requires running services)
go test ./internal/integration -v

# Property tests
go test ./internal/... -run Property -v

# Failure mode tests
./scripts/test-aggregator-restart.sh
./scripts/test-backpressure.sh
./scripts/test-dlq-routing.sh
```

## Monitoring

Prometheus metrics at:
- Ingest API: `:8081/metrics`
- Aggregator: `:8082/metrics`
- Diagnoser: `:8083/metrics`

Jaeger traces: `:16686`  
Grafana: `:3001` (admin/admin)

Key metrics:
- `ingest_events_total` - Events received
- `ingest_events_rejected_total` - Auth failures, rate limits
- `aggregator_events_processed_total` - By outcome (success/duplicate/error)
- `aggregator_processing_delay_seconds` - End-to-end latency
- `aggregator_dedup_rate` - % of duplicates
- `nats_consumer_lag` - Queue backlog

## Deployment

### Same network
Just use local IPs. If server is at `192.168.1.100`, point probes there.

### Different networks
Need to expose port 8081 to the internet:
- Port forward on your router, or
- Use ngrok: `ngrok http 8081`, or  
- Deploy server to cloud (AWS/GCP/Azure)

### Production checklist
- Change default credentials
- Use real API tokens (not `demo-token`)
- Set up TLS (put nginx in front)
- Configure retention (see Database section)
- Set up alerts in Grafana
- Back up PostgreSQL

## Database

Tables:
- `events_seen`: Deduplication state (event_id primary key)
- `agg_1m`: Per-minute aggregates (client_id, target, window_start_ts)
- `diagnosis_history`: Diagnosis results over time

Cleanup runs daily via `bin/cleanup`:
- Deletes `events_seen` > 7 days old
- Deletes `agg_1m` > 90 days old

Adjust retention in `.env`:
```
DEDUP_RETENTION_DAYS=7
AGGREGATE_RETENTION_DAYS=90
```

## Performance

Tested with:
- 1000 probes × 60s interval = 16.7 events/sec
- Single aggregator handles 1000+ events/sec
- Single ingest API handles 5000+ req/sec
- P95 end-to-end latency < 2 seconds

For higher load, run multiple ingest API instances behind a load balancer.

## License

GNU GPL v3

## Contributing

Open an issue or PR.
