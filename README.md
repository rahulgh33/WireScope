# Network QoE Telemetry Platform

A distributed system for measuring, collecting, aggregating, and analyzing network quality of experience metrics.

## Architecture

The platform consists of four core components:

- **Probe Agent**: CLI tool that measures network performance metrics (DNS, TCP, TLS, HTTP timings and throughput)
- **Ingest API**: HTTP service that receives telemetry events from probes
- **Aggregator**: Consumer service that processes events and creates time-windowed aggregates
- **Diagnoser**: Rule-based service that analyzes aggregated data and labels performance bottlenecks

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Go 1.21 or later
- Make

### Development Environment

1. Start the development environment:
   ```bash
   make dev
   ```

2. This will start all required services:
   - PostgreSQL (port 5432)
   - NATS JetStream (ports 4222, 8222)
   - Prometheus (port 9090)
   - Grafana (port 3000, admin/admin)
   - Jaeger (port 16686)
   - OpenTelemetry Collector
   - Test Target Server (port 8080)

3. Apply database migrations:
   ```bash
   make migrate
   ```

4. Build the application binaries:
   ```bash
   make build
   ```

### Available Services

- **Grafana Dashboard**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Jaeger Tracing**: http://localhost:16686
- **Test Target Server**: http://localhost:8080

### Development Commands

```bash
# Start all services
make up

# Stop all services
make down

# View logs
make logs

# Build binaries
make build

# Run tests
make test

# Clean up
make clean

# Database migrations
make migrate-up      # Apply migrations
make migrate-down    # Rollback last migration
make migrate-create  # Create new migration
```

## Project Structure

```
├── cmd/                    # Application entry points
│   ├── probe/             # Probe agent CLI
│   ├── ingest/            # Ingest API service
│   ├── aggregator/        # Aggregator consumer
│   └── diagnoser/         # Diagnoser service
├── internal/              # Private application code
├── pkg/                   # Public library code
├── config/                # Configuration files
│   ├── prometheus.yml     # Prometheus configuration
│   ├── otel-collector.yml # OpenTelemetry configuration
│   ├── grafana/           # Grafana provisioning
│   └── test-endpoints/    # Test server files
├── migrations/            # Database migrations
├── docker-compose.yml     # Development environment
├── Makefile              # Build and development commands
└── go.mod                # Go module definition
```

## Database Schema

The platform uses PostgreSQL with the following main tables:

- `events_seen`: Deduplication table for exactly-once processing
- `agg_1m`: One-minute aggregated metrics with percentiles
- `alerts`: Optional alerting table

## Configuration

Configuration is handled through environment variables with sensible defaults for development. See `config/config.go` for available options.

## Testing

The platform includes a test target server with the following endpoints:

- `/health` - Fast response for latency testing
- `/slow?ms=N` - Configurable delay endpoint
- `/fixed/1mb.bin` - 1MB file for throughput testing

## Next Steps

1. Implement the probe agent (Task 6)
2. Build the ingest API (Task 7)
3. Create the aggregator consumer (Task 8)
4. Add percentile calculations (Task 9)

For detailed implementation tasks, see `.kiro/specs/network-qoe-telemetry-platform/tasks.md`.