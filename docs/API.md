# API Documentation - Network QoE Telemetry Platform

## Ingest API

### Base URL

```
http://localhost:8081/api/v1
```

### Authentication

All API requests require Bearer token authentication:

```bash
curl -H "Authorization: Bearer YOUR_API_TOKEN" \
  http://localhost:8081/api/v1/events
```

**Development token**: `demo-token`

### Endpoints

#### POST /api/v1/events

Submit a telemetry event for processing.

**Request Headers**:
- `Content-Type: application/json`
- `Authorization: Bearer <token>`

**Request Body**:

```json
{
  "schema_version": 1,
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "client_id": "probe-001",
  "ts_ms": 1703347200000,
  "target": "https://api.example.com",
  "network_context": {
    "source_ip": "192.168.1.100",
    "network_type": "wifi",
    "isp": "Example ISP"
  },
  "timing_measurements": {
    "dns_ms": 15.3,
    "tcp_ms": 25.7,
    "tls_ms": 45.2,
    "ttfb_ms": 125.8,
    "throughput_mbps": 95.4
  },
  "error_stage": null
}
```

**Field Descriptions**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `schema_version` | integer | Yes | Schema version (currently 1) |
| `event_id` | UUID string | Yes | Unique event identifier (UUIDv4) |
| `client_id` | string | Yes | Unique probe/client identifier |
| `ts_ms` | integer | Yes | Event timestamp (Unix milliseconds) |
| `target` | string | Yes | Target URL being measured |
| `network_context` | object | No | Network metadata |
| `network_context.source_ip` | string | No | Source IP address |
| `network_context.network_type` | string | No | Network type (wifi, cellular, ethernet) |
| `network_context.isp` | string | No | Internet Service Provider |
| `timing_measurements` | object | Yes | Timing measurements |
| `timing_measurements.dns_ms` | float | No | DNS resolution time (ms) |
| `timing_measurements.tcp_ms` | float | No | TCP handshake time (ms) |
| `timing_measurements.tls_ms` | float | No | TLS handshake time (ms) |
| `timing_measurements.ttfb_ms` | float | No | Time to first byte (ms) |
| `timing_measurements.throughput_mbps` | float | No | Throughput (Mbps) |
| `error_stage` | string | No | Stage where error occurred (dns, tcp, tls, http, throughput) |

**Success Response**:

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "status": "ok",
  "event_id": "550e8400-e29b-41d4-a716-446655440000",
  "recv_ts_ms": 1703347201234
}
```

**Error Responses**:

```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "invalid_request",
  "message": "missing required field: event_id"
}
```

```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{
  "error": "unauthorized",
  "message": "invalid or missing API token"
}
```

```http
HTTP/1.1 429 Too Many Requests
Content-Type: application/json

{
  "error": "rate_limit_exceeded",
  "message": "rate limit exceeded for client: probe-001",
  "retry_after": 60
}
```

```http
HTTP/1.1 500 Internal Server Error
Content-Type: application/json

{
  "error": "internal_error",
  "message": "failed to process event"
}
```

**Rate Limiting**:
- Default: 100 requests/second per `client_id`
- Burst: 60 requests
- Headers returned:
  - `X-RateLimit-Limit`: Rate limit
  - `X-RateLimit-Remaining`: Remaining requests
  - `X-RateLimit-Reset`: Reset timestamp

**Example Requests**:

```bash
# Basic request
curl -X POST http://localhost:8081/api/v1/events \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer demo-token" \
  -d '{
    "schema_version": 1,
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "client_id": "probe-001",
    "ts_ms": 1703347200000,
    "target": "https://api.example.com",
    "timing_measurements": {
      "dns_ms": 15.3,
      "tcp_ms": 25.7,
      "ttfb_ms": 125.8
    }
  }'

# Python example
import requests
import uuid
import time

event = {
    "schema_version": 1,
    "event_id": str(uuid.uuid4()),
    "client_id": "probe-001",
    "ts_ms": int(time.time() * 1000),
    "target": "https://api.example.com",
    "timing_measurements": {
        "dns_ms": 15.3,
        "tcp_ms": 25.7,
        "ttfb_ms": 125.8
    }
}

response = requests.post(
    "http://localhost:8081/api/v1/events",
    json=event,
    headers={"Authorization": "Bearer demo-token"}
)

print(response.status_code, response.json())

# Go example
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
    "time"
    "github.com/google/uuid"
)

type Event struct {
    SchemaVersion int64      `json:"schema_version"`
    EventID       string     `json:"event_id"`
    ClientID      string     `json:"client_id"`
    TsMs          int64      `json:"ts_ms"`
    Target        string     `json:"target"`
    Measurements  struct {
        DnsMs  float64 `json:"dns_ms"`
        TcpMs  float64 `json:"tcp_ms"`
        TtfbMs float64 `json:"ttfb_ms"`
    } `json:"timing_measurements"`
}

func main() {
    event := Event{
        SchemaVersion: 1,
        EventID:       uuid.New().String(),
        ClientID:      "probe-001",
        TsMs:          time.Now().UnixMilli(),
        Target:        "https://api.example.com",
    }
    event.Measurements.DnsMs = 15.3
    event.Measurements.TcpMs = 25.7
    event.Measurements.TtfbMs = 125.8

    data, _ := json.Marshal(event)
    req, _ := http.NewRequest("POST", 
        "http://localhost:8081/api/v1/events",
        bytes.NewBuffer(data))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer demo-token")

    client := &http.Client{}
    resp, _ := client.Do(req)
    defer resp.Body.Close()
}
```

#### GET /health

Health check endpoint.

**Response**:

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime_seconds": 3600,
  "components": {
    "database": "healthy",
    "nats": "healthy"
  }
}
```

#### GET /metrics

Prometheus metrics endpoint.

**Response**: Prometheus text format

```
# HELP telemetry_ingest_requests_total Total number of ingest API requests
# TYPE telemetry_ingest_requests_total counter
telemetry_ingest_requests_total{status="success"} 12345

# HELP telemetry_ingest_duration_seconds Ingest API request duration
# TYPE telemetry_ingest_duration_seconds histogram
telemetry_ingest_duration_seconds_bucket{le="0.005"} 1000
telemetry_ingest_duration_seconds_bucket{le="0.01"} 1500
...
```

## Database Schema

### Table: agg_1m

One-minute aggregated metrics.

**Columns**:

| Column | Type | Description |
|--------|------|-------------|
| `client_id` | VARCHAR(255) | Client identifier (part of PK) |
| `target` | VARCHAR(255) | Target URL (part of PK) |
| `window_start_ts` | TIMESTAMP | Window start time (part of PK) |
| `count_total` | BIGINT | Total event count |
| `count_success` | BIGINT | Successful event count |
| `count_error` | BIGINT | Error event count |
| `dns_error_count` | BIGINT | DNS errors |
| `tcp_error_count` | BIGINT | TCP errors |
| `tls_error_count` | BIGINT | TLS errors |
| `http_error_count` | BIGINT | HTTP errors |
| `throughput_error_count` | BIGINT | Throughput measurement errors |
| `dns_p50` | DOUBLE PRECISION | DNS P50 (ms) |
| `dns_p95` | DOUBLE PRECISION | DNS P95 (ms) |
| `tcp_p50` | DOUBLE PRECISION | TCP P50 (ms) |
| `tcp_p95` | DOUBLE PRECISION | TCP P95 (ms) |
| `tls_p50` | DOUBLE PRECISION | TLS P50 (ms) |
| `tls_p95` | DOUBLE PRECISION | TLS P95 (ms) |
| `ttfb_p50` | DOUBLE PRECISION | TTFB P50 (ms) |
| `ttfb_p95` | DOUBLE PRECISION | TTFB P95 (ms) |
| `throughput_p50` | DOUBLE PRECISION | Throughput P50 (Mbps) |
| `throughput_p95` | DOUBLE PRECISION | Throughput P95 (Mbps) |
| `diagnosis_label` | VARCHAR(50) | Diagnosis label (nullable) |
| `updated_at` | TIMESTAMP | Last update time |

**Primary Key**: `(client_id, target, window_start_ts)`

**Indexes**:
- `idx_agg_1m_window` on `window_start_ts`
- `idx_agg_1m_diagnosis` on `diagnosis_label` (partial)
- `idx_agg_1m_client_target_window` on `(client_id, target, window_start_ts DESC)`

**Example Queries**:

```sql
-- Get latest aggregates for a client
SELECT * FROM agg_1m
WHERE client_id = 'probe-001'
  AND window_start_ts > NOW() - INTERVAL '1 hour'
ORDER BY window_start_ts DESC;

-- Get aggregates with issues
SELECT * FROM agg_1m
WHERE diagnosis_label IS NOT NULL
  AND diagnosis_label != 'healthy'
ORDER BY window_start_ts DESC
LIMIT 100;

-- Calculate average latency by target
SELECT 
    target,
    AVG(ttfb_p50) as avg_p50,
    AVG(ttfb_p95) as avg_p95,
    COUNT(*) as window_count
FROM agg_1m
WHERE window_start_ts > NOW() - INTERVAL '24 hours'
GROUP BY target
ORDER BY avg_p95 DESC;

-- Get error rate by client
SELECT 
    client_id,
    SUM(count_error)::float / NULLIF(SUM(count_total), 0) * 100 as error_rate_pct,
    SUM(count_total) as total_events
FROM agg_1m
WHERE window_start_ts > NOW() - INTERVAL '1 hour'
GROUP BY client_id
ORDER BY error_rate_pct DESC;
```

### Table: events_seen

Deduplication table for exactly-once processing.

**Columns**:

| Column | Type | Description |
|--------|------|-------------|
| `event_id` | UUID | Event UUID (PK) |
| `client_id` | VARCHAR(255) | Client identifier |
| `ts_ms` | BIGINT | Event timestamp |
| `created_at` | TIMESTAMP | Record creation time |

**Indexes**:
- Primary key on `event_id`
- `idx_events_seen_client_ts` on `(client_id, ts_ms)`
- `idx_events_seen_created_at` on `created_at`

## Metrics

### Ingest API Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `telemetry_ingest_requests_total` | Counter | `status` | Total requests (success/error) |
| `telemetry_ingest_duration_seconds` | Histogram | - | Request duration |
| `telemetry_ingest_rate_limited_total` | Counter | `client_id` | Rate limit events |
| `telemetry_ingest_auth_failures_total` | Counter | - | Authentication failures |

### Queue Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `telemetry_queue_consumer_lag_messages` | Gauge | `consumer` | Consumer lag (messages) |
| `telemetry_queue_ack_pending_messages` | Gauge | `consumer` | Pending acknowledgments |
| `telemetry_queue_published_total` | Counter | `subject` | Published messages |
| `telemetry_queue_processing_duration_seconds` | Histogram | - | Message processing time |

### Aggregator Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `telemetry_events_processed_total` | Counter | `status` | Events processed (success/duplicate/error) |
| `telemetry_processing_delay_seconds` | Histogram | - | End-to-end latency |
| `telemetry_windows_flushed_total` | Counter | - | Windows flushed to database |
| `telemetry_dedup_rate` | Gauge | - | Duplicate event rate |
| `telemetry_dlq_messages_total` | Counter | - | Dead letter queue messages |

### Database Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `telemetry_db_connections_in_use` | Gauge | - | Active connections |
| `telemetry_db_connections_max` | Gauge | - | Max connection pool size |
| `telemetry_db_query_duration_seconds` | Histogram | `operation` | Query duration |
| `telemetry_db_transactions_total` | Counter | `status` | Transactions (commit/rollback) |

## Error Codes

| Code | HTTP Status | Description | Resolution |
|------|-------------|-------------|------------|
| `invalid_request` | 400 | Malformed request body | Check JSON syntax and required fields |
| `invalid_schema_version` | 400 | Unsupported schema version | Use schema version 1 |
| `invalid_event_id` | 400 | Invalid UUID format | Use UUIDv4 format |
| `invalid_timestamp` | 400 | Timestamp out of range | Use Unix milliseconds |
| `unauthorized` | 401 | Missing or invalid token | Provide valid Bearer token |
| `rate_limit_exceeded` | 429 | Too many requests | Wait and retry after specified seconds |
| `internal_error` | 500 | Server error | Check logs, retry with backoff |
| `service_unavailable` | 503 | Service temporarily down | Retry with exponential backoff |

## Best Practices

### Event ID Generation

```bash
# Use UUIDv4 for event IDs
# Option 1: Random UUID (most common)
event_id=$(uuidgen)

# Option 2: Deterministic UUID (for deduplication)
# Hash of (client_id + target + ts_ms)
event_id=$(echo "${client_id}${target}${ts_ms}" | sha256sum | awk '{print substr($1,1,32)}')
```

### Error Handling

```python
import requests
from requests.adapters import HTTPAdapter
from requests.packages.urllib3.util.retry import Retry

# Configure retry strategy
retry_strategy = Retry(
    total=3,
    status_forcelist=[429, 500, 502, 503, 504],
    method_whitelist=["POST"],
    backoff_factor=1  # 1s, 2s, 4s
)

adapter = HTTPAdapter(max_retries=retry_strategy)
session = requests.Session()
session.mount("http://", adapter)
session.mount("https://", adapter)

# Use session for requests
response = session.post(
    "http://localhost:8081/api/v1/events",
    json=event,
    headers={"Authorization": "Bearer demo-token"},
    timeout=10
)
```

### Batch Processing

For high-throughput scenarios:

```python
import asyncio
import aiohttp

async def send_event(session, event):
    async with session.post(
        "http://localhost:8081/api/v1/events",
        json=event,
        headers={"Authorization": "Bearer demo-token"}
    ) as response:
        return await response.json()

async def send_batch(events):
    async with aiohttp.ClientSession() as session:
        tasks = [send_event(session, event) for event in events]
        return await asyncio.gather(*tasks)

# Send 100 events concurrently
results = asyncio.run(send_batch(events))
```

### Clock Skew Handling

```python
import time
import ntplib

def get_server_time():
    """Get accurate time from NTP server"""
    try:
        client = ntplib.NTPClient()
        response = client.request('pool.ntp.org')
        return int(response.tx_time * 1000)
    except:
        # Fallback to local time
        return int(time.time() * 1000)

# Use in event generation
event["ts_ms"] = get_server_time()
```

## OpenAPI Specification

```yaml
openapi: 3.0.0
info:
  title: Network QoE Telemetry Platform API
  version: 1.0.0
  description: API for submitting network telemetry events

servers:
  - url: http://localhost:8081/api/v1
    description: Development server

security:
  - bearerAuth: []

paths:
  /events:
    post:
      summary: Submit telemetry event
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/TelemetryEvent'
      responses:
        '200':
          description: Event accepted
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/SuccessResponse'
        '400':
          description: Bad request
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
        '401':
          description: Unauthorized
        '429':
          description: Rate limit exceeded
        '500':
          description: Internal server error

  /health:
    get:
      summary: Health check
      security: []
      responses:
        '200':
          description: Service healthy
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/HealthResponse'

components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer

  schemas:
    TelemetryEvent:
      type: object
      required:
        - schema_version
        - event_id
        - client_id
        - ts_ms
        - target
        - timing_measurements
      properties:
        schema_version:
          type: integer
          example: 1
        event_id:
          type: string
          format: uuid
          example: "550e8400-e29b-41d4-a716-446655440000"
        client_id:
          type: string
          example: "probe-001"
        ts_ms:
          type: integer
          format: int64
          example: 1703347200000
        target:
          type: string
          format: uri
          example: "https://api.example.com"
        network_context:
          type: object
          properties:
            source_ip:
              type: string
            network_type:
              type: string
            isp:
              type: string
        timing_measurements:
          type: object
          properties:
            dns_ms:
              type: number
              format: float
            tcp_ms:
              type: number
              format: float
            tls_ms:
              type: number
              format: float
            ttfb_ms:
              type: number
              format: float
            throughput_mbps:
              type: number
              format: float
        error_stage:
          type: string
          enum: [dns, tcp, tls, http, throughput]
          nullable: true

    SuccessResponse:
      type: object
      properties:
        status:
          type: string
          example: "ok"
        event_id:
          type: string
          format: uuid
        recv_ts_ms:
          type: integer
          format: int64

    ErrorResponse:
      type: object
      properties:
        error:
          type: string
        message:
          type: string
        retry_after:
          type: integer
          nullable: true

    HealthResponse:
      type: object
      properties:
        status:
          type: string
        version:
          type: string
        uptime_seconds:
          type: integer
        components:
          type: object
```

## Support

For API questions or issues:
- GitHub Issues: https://github.com/rahulgh33/Distributed-Telemetry-Platform/issues
- Documentation: See `docs/` directory
