# Architecture

## Data flow

```
Probe → Ingest API → NATS → Aggregator → PostgreSQL
                              ↓
                         Diagnoser → AI Agent
                              ↓
                         Web UI (WebSocket)
```

## Components

### Probe
- Lightweight binary (5MB)
- Measures latency, packet loss, jitter to configured endpoints
- Sends metrics every 10s
- Automatic retry with exponential backoff

### Ingest API
- REST endpoint for probe metrics
- Rate limiting per probe
- Authentication via API key
- Publishes to NATS subject `telemetry.raw`

### Aggregator
- Consumes from NATS
- Computes 1m, 5m, 1h aggregates (avg, p50, p95, p99)
- Deduplicates based on `(probe_id, target, timestamp)`
- Writes to PostgreSQL

### Diagnoser
- Monitors aggregates for anomalies
- Threshold-based alerts (configurable)
- Creates diagnosis records in database
- Notifies AI agent and Web UI

### AI Agent
- Analyzes diagnosed issues
- Uses OpenAI API for root cause analysis
- Stores analysis in database
- Real-time updates via WebSocket

### Web UI
- React SPA with dark mode
- Real-time metrics via WebSocket
- Historical data via REST API
- Probe management and configuration

## Database schema

Key tables:
- `probes`: Probe registration and metadata
- `targets`: Network endpoints to measure
- `agg_1m`, `agg_5m`, `agg_1h`: Aggregated metrics
- `diagnoses`: Detected issues
- `ai_analysis`: Root cause analysis

## Message queue

NATS JetStream subjects:
- `telemetry.raw`: Raw probe data
- `telemetry.agg`: Aggregated data
- `telemetry.diagnoses`: Detected issues

Consumer groups ensure exactly-once processing.

## Observability

- Prometheus metrics on `/metrics`
- Structured JSON logs
- Grafana dashboards included
- OpenTelemetry tracing (optional)
