# WireScope

Network telemetry platform for measuring and analyzing network performance from distributed probes.

## What it does

Deploys lightweight probes anywhere (home, office, cloud servers) that measure network performance and send data to a central server. The server aggregates the data, identifies problems, and shows everything in a web dashboard.

Useful for:
- Monitoring ISP quality from different locations
- Tracking corporate network performance across offices
- Debugging network issues with actual data
- Understanding where latency comes from (DNS? Handshake? Server?)

## Architecture

```
Probe → Ingest API → NATS → Aggregator → PostgreSQL → Web UI
                                              ↓
                                         Diagnoser
                                              ↓
                                          AI Agent
```

**Probe**: Measures DNS, TCP/TLS handshake, HTTP latency, and throughput  
**Ingest API**: Receives probe data via HTTP  
**NATS**: Message queue (handles retries, ordering)  
**Aggregator**: Calculates per-minute P50/P95 percentiles, deduplicates events  
**Diagnoser**: Identifies bottlenecks (DNS issues, slow servers, etc.)  
**AI Agent**: Answer questions about the data in natural language  
**Web UI**: Dashboard with charts, real-time updates

## Quick Start

### Server (your computer)

```bash
git clone https://github.com/rahulgh33/WireScope.git
cd WireScope
docker-compose up -d
```

Open http://localhost:3000

### Probe (any other machine)

```bash
curl -sSL https://raw.githubusercontent.com/rahulgh33/WireScope/main/scripts/install-probe.sh | bash -s -- YOUR_SERVER_IP
```

Or manually:
```bash
# Download for your OS from releases
wget https://github.com/rahulgh33/WireScope/releases/latest/download/probe-linux-amd64
chmod +x probe-linux-amd64

# Run it
./probe-linux-amd64 \
  --ingest-url http://YOUR_SERVER_IP:8081/events \
  --api-token demo-token \
  --target https://google.com \
  --client-id my-probe
```

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

Set via environment variables or command flags.

### Probe
- `--ingest-url`: Where to send data (required)
- `--api-token`: Authentication token (required)
- `--target`: URL to monitor (required)
- `--client-id`: Unique identifier for this probe
- `--interval`: Seconds between measurements (default: 60)

### Ingest API
- `NATS_URL`: NATS connection string
- `API_TOKENS`: Comma-separated valid tokens
- `PORT`: HTTP port (default: 8081)

### Aggregator
- `NATS_URL`: NATS connection string
- `DATABASE_URL`: PostgreSQL connection string
- `AGGREGATION_WINDOW`: Window size in seconds (default: 60)

### Diagnoser
- `DATABASE_URL`: PostgreSQL connection string
- `CHECK_INTERVAL`: How often to run diagnosis (default: 60s)

### AI Agent
- `DATABASE_URL`: PostgreSQL connection string  
- `OPENAI_API_KEY`: Your OpenAI key (optional, for AI features)
- `PORT`: HTTP port (default: 9000)

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
