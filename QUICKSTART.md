# WireScope Quick Start Guide

## Prerequisites

⚠️ **CRITICAL**: Stop local PostgreSQL before starting:
```bash
brew services stop postgresql@16
brew services stop postgresql@15
# Verify port 5432 is free
lsof -i :5432
```

Local Postgres conflicts with the Docker container and causes "role telemetry does not exist" errors.

## Starting the Server (Your Mac)

### 1. Start Docker Services
```bash
cd /Users/rahulghosh/Documents/distributed_telemetry/Distributed-Telemetry-Platform/Distributed-Telemetry-Platform
docker-compose up -d
```

### 2. Start Go Services
```bash
./scripts/start-services.sh
```

### 3. Check Everything is Running
```bash
./scripts/status-services.sh
```

You should see:
- ✓ PostgreSQL, NATS, Grafana, Prometheus running
- ✓ Ingest API, Aggregator, AI Agent running

### 4. Start Web UI
```bash
cd web && npm run dev
```
Access at: http://localhost:5173

## Connecting Probes

### Local Probe (Same Machine)
```bash
./bin/probe \
  --ingest-url http://localhost:8081/events \
  --client-id mac-local \
  --target http://example.com \
  --interval 60s \
  --tracing-enabled=false
```

**Note:** `--api-token` is optional when server authentication is disabled (default).

### Remote Probe (Different Subnet - Use ngrok)

**On your Mac:**
```bash
# Start ngrok tunnel (keep running)
ngrok http 8081
# Copy the URL (e.g., https://abc-xyz.ngrok-free.dev)
```

**On remote machine:**
```bash
# Download probe (Linux)
wget https://github.com/rahulgh33/WireScope/releases/latest/download/probe-linux-amd64 -O probe
chmod +x probe

# Run probe (api-token optional)
./probe \
  --ingest-url https://YOUR-NGROK-URL.ngrok-free.dev/events \
  --client-id remote-probe-1 \
  --target http://example.com \
  --interval 60s \
  --tracing-enabled=false
```

**For same network (no ngrok):** Replace URL with `http://YOUR_MAC_IP:8081/events`  
Find your IP: `ifconfig | grep "inet " | grep -v 127.0.0.1`

## Viewing Data

### Web UI (Main Dashboard)
- **URL:** http://localhost:5173
- Shows: Active clients, real-time metrics, AI queries

### Grafana Dashboards
- **URL:** http://localhost:3000 (admin/admin)
- **Network Performance** - Main probe data dashboard
- **Platform Health** - System metrics
- **Ingest/Aggregator/Queue** - Service metrics

### Prometheus
- **URL:** http://localhost:9090
- Raw metrics and custom queries

### Jaeger (Distributed Tracing)
- **URL:** http://localhost:16686
- Full request traces

### Database Queries
```bash
# Connect to database
docker exec -it wirescope-postgres-1 psql -U telemetry -d telemetry

# View recent aggregated data
SELECT client_id, target, window_start_ts, count_total 
FROM agg_1m 
ORDER BY window_start_ts DESC 
LIMIT 10;

# View all clients
SELECT DISTINCT client_id FROM agg_1m;

# Exit
\q
```

## Common Issues & Fixes

### "role telemetry does not exist" Error

**Cause:** Local Postgres running on port 5432 conflicts with Docker  
**Fix:**
```bash
# Stop local postgres permanently
brew services stop postgresql@16
brew services stop postgresql@15

# Reset Docker database
docker-compose down -v
docker-compose up -d postgres nats
sleep 10
make migrate
./scripts/start-services.sh
```

**Prevent forever:** Keep local postgres stopped or use different port (5433)

### Database Tables Missing
```bash
make migrate
```

### Services Not Running
```bash
# Check status
./scripts/status-services.sh

# View logs
tail -f logs/*.log

# Restart everything
./scripts/stop-services.sh
docker-compose down
docker-compose up -d
./scripts/start-services.sh
```

### Probe Connection Issues
**404 errors:** ngrok tunnel died, restart it
**Timeout errors:** Check firewall, verify server IP/URL
**Measurement errors:** Try different target (e.g., https://www.purdue.edu)

### No Data in Web UI
1. Check database has data:
   ```bash
   docker exec wirescope-postgres-1 psql -U telemetry -d telemetry -c "SELECT COUNT(*) FROM agg_1m;"
   ```
2. Check aggregator is running:
   ```bash
   ps aux | grep aggregator
   ```
3. Check aggregator logs:
   ```bash
   tail -f logs/aggregator.log
   ```

### Ngrok Tunnel Issues
- Free tier: URLs change on restart
- Keep terminal open while testing
- Check terminal output for current URL

## Stopping Everything

```bash
# Stop Go services
./scripts/stop-services.sh

# Stop Docker services
docker-compose down

# Stop ngrok
# Ctrl+C in ngrok terminal
```

## Building Changes

```bash
# Rebuild all binaries
make build

# Rebuild specific component
go build -o bin/probe ./cmd/probe
go build -o bin/ingest ./cmd/ingest
go build -o bin/aggregator ./cmd/aggregator

# Rebuild for Linux (for remote probes)
GOOS=linux GOARCH=amd64 go build -o bin/probe-linux-amd64 ./cmd/probe
```

## Key Ports

- **8081** - Ingest API (receive probe data)
- **3000** - Grafana
- **5173** - Web UI (Vite dev server)
- **9000** - AI Agent
- **9090** - Prometheus
- **4222** - NATS
- **5432** - PostgreSQL
- **16686** - Jaeger UI

## Architecture Flow

```
Probe → ngrok (if remote) → Ingest API (8081) → NATS → Aggregator → PostgreSQL
                                                                         ↓
                                                                    Web UI / Grafana
```

## Quick Commands Reference

```bash
# Status check
./scripts/status-services.sh

# View logs
tail -f logs/aggregator.log
tail -f logs/ingest.log

# Database check
docker exec wirescope-postgres-1 psql -U telemetry -d telemetry -c "SELECT DISTINCT client_id FROM agg_1m;"

# Restart services
./scripts/stop-services.sh && ./scripts/start-services.sh

# Rebuild everything
make build

# Start Web UI
cd web && npm run dev
```

## Typical Workflow

1. **Start server:** `docker-compose up -d && ./scripts/start-services.sh`
2. **Start ngrok:** `ngrok http 8081` (copy URL)
3. **Connect probe:** Use ngrok URL in `--ingest-url`
4. **View data:** http://localhost:5173 or http://localhost:3000
5. **Check issues:** `./scripts/status-services.sh` and `tail -f logs/*.log`

## Notes

- **Database credentials:** telemetry/telemetry (already configured)
- **API token:** demo-token (can be changed in ingest startup)
- **Aggregation window:** 60 seconds (data appears after ~1 minute)
- **Free ngrok:** URL changes on restart, need to update probe
- **Production:** Use cloud deployment (see CLOUD_DEPLOYMENT.md)
