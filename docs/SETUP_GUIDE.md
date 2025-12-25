# Company Setup Guide

This guide walks through setting up WireScope with one server and multiple client machines.

## Scenario

You're a company that wants to monitor network performance from various locations:
- **1 server** (your office, data center, or cloud VM) - runs all backend services
- **Multiple clients** (employee laptops, branch offices, remote sites) - each runs a probe

## Overview

**Clients are auto-registered.** There's no manual "add client" step. When a probe first sends data with a valid API token, it appears in the dashboard automatically.

Each probe:
1. Generates or uses a configured `client_id`
2. Sends telemetry to server's ingest API
3. Shows up in Web UI immediately after first event

## Step 1: Set up the server

Pick one machine with stable internet. This runs the central platform.

### Requirements
- Docker + Docker Compose
- Open ports: 8080 (ingest API), 3000 (web UI), 4222 (NATS - optional if probes are external)
- 4GB RAM minimum (8GB recommended)
- Linux, macOS, or Windows

### Installation

```bash
# Clone repo
git clone https://github.com/rahulgh33/WireScope.git
cd WireScope

# Start all services
docker-compose up -d

# Check status
docker-compose ps
```

Services running:
- Ingest API: `http://SERVER_IP:8080`
- Web UI: `http://SERVER_IP:3000`
- PostgreSQL: Internal (port 5432)
- NATS: Internal (port 4222)
- Prometheus: `http://SERVER_IP:9090`
- Grafana: `http://SERVER_IP:3001`

### Get your server IP

```bash
# Linux/Mac
hostname -I | awk '{print $1}'

# Or check
ip addr show
```

### Create API tokens

Edit `docker-compose.yml`:
```yaml
services:
  ingest:
    environment:
      - API_TOKENS=token-office,token-remote1,token-remote2
```

Restart:
```bash
docker-compose restart ingest
```

**Best practice:** Use different tokens per location/team for tracking and revocation.

## Step 2: Install probes on client machines

Each machine that will monitor network performance needs the probe binary.

### Automatic install

```bash
curl -sSL https://raw.githubusercontent.com/rahulgh33/WireScope/main/scripts/install-probe.sh | bash -s -- SERVER_IP API_TOKEN
```

Example:
```bash
curl -sSL https://raw.githubusercontent.com/rahulgh33/WireScope/main/scripts/install-probe.sh | bash -s -- 192.168.1.100 token-office
```

### Manual install

1. Download binary for your OS:
```bash
# Linux AMD64
wget https://github.com/rahulgh33/WireScope/releases/latest/download/probe-linux-amd64

# macOS (Apple Silicon)
wget https://github.com/rahulgh33/WireScope/releases/latest/download/probe-darwin-arm64

# Windows
# Download probe-windows-amd64.exe
```

2. Make executable (Linux/Mac):
```bash
chmod +x probe-linux-amd64
mv probe-linux-amd64 /usr/local/bin/probe
```

3. Run it:
```bash
probe \
  --ingest-url http://192.168.1.100:8080/events \
  --api-token token-office \
  --target https://google.com \
  --client-id office-laptop-john
```

### Run as service (Linux systemd)

Create `/etc/systemd/system/wirescope-probe.service`:
```ini
[Unit]
Description=WireScope Network Probe
After=network.target

[Service]
Type=simple
User=nobody
ExecStart=/usr/local/bin/probe \
  --ingest-url http://192.168.1.100:8080/events \
  --api-token token-office \
  --target https://google.com \
  --client-id office-server-1 \
  --interval 60
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable wirescope-probe
sudo systemctl start wirescope-probe
sudo systemctl status wirescope-probe
```

### Run as service (macOS)

Create `~/Library/LaunchAgents/com.wirescope.probe.plist`:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.wirescope.probe</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/probe</string>
        <string>--ingest-url</string>
        <string>http://192.168.1.100:8080/events</string>
        <string>--api-token</string>
        <string>token-office</string>
        <string>--target</string>
        <string>https://google.com</string>
        <string>--client-id</string>
        <string>johns-macbook</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
```

Load it:
```bash
launchctl load ~/Library/LaunchAgents/com.wirescope.probe.plist
launchctl start com.wirescope.probe
```

## Step 3: Configure what to monitor

Each probe can monitor multiple targets. Edit the probe command to add more:

**Multiple targets** (requires code change or config file - current version uses single target):
```bash
# Current: Single target via flag
probe --target https://api.example.com

# TODO: Support config file with multiple targets
```

For now, run multiple probe processes:
```bash
# Terminal 1
probe --target https://google.com --client-id office-to-google &

# Terminal 2
probe --target https://aws.amazon.com --client-id office-to-aws &
```

## Step 4: View clients in dashboard

1. Open web UI: `http://SERVER_IP:3000`
2. Go to **Clients** page
3. See all probes that have sent data

Clients appear automatically after first telemetry event (usually within 60 seconds).

## Client management

### View clients
Web UI → Clients page shows:
- Client ID
- Status (active/inactive/warning)
- Last seen timestamp
- Average latency
- Error count

### Delete clients
Click "Delete" button on any client card (requires two clicks to confirm).

Deletes all data for that client:
- All aggregates
- Diagnostics
- AI analyses

### Stop receiving data from client
Just stop the probe process on that machine:
```bash
# systemd
sudo systemctl stop wirescope-probe

# macOS
launchctl unload ~/Library/LaunchAgents/com.wirescope.probe.plist

# Manual process
pkill probe
```

Client will show as "inactive" after ~5 minutes of no data.

## Firewall configuration

### Server (open these ports)
- `8080/tcp` - Ingest API (probes connect here)
- `3000/tcp` - Web UI (browser access)
- `4222/tcp` - NATS (only if using external NATS cluster)

### Clients (outbound only)
- Needs outbound HTTPS to server's ingest API (port 8080)
- No inbound ports required

### Example iptables (server)
```bash
# Allow ingest API
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT

# Allow web UI
sudo iptables -A INPUT -p tcp --dport 3000 -j ACCEPT
```

## Security best practices

1. **Use strong API tokens**
   - Not `demo-token` in production
   - Generate with: `openssl rand -hex 32`

2. **Enable HTTPS**
   - Put nginx/Caddy in front
   - Use Let's Encrypt for free TLS

3. **Firewall the ingest API**
   - Only allow known IP ranges if possible
   - Or use VPN for probe connections

4. **Rotate tokens**
   - Change tokens periodically
   - Revoke compromised tokens by removing from `API_TOKENS` env var

5. **Monitor access**
   - Check Prometheus metrics for auth failures
   - `ingest_auth_failures_total{reason="invalid_token"}`

## Troubleshooting

### Probe can't reach server
```bash
# Test connectivity
curl http://SERVER_IP:8080/health

# Should return: {"status":"ok"}
```

### Client not showing up in UI
```bash
# Check probe logs
journalctl -u wirescope-probe -f

# Look for:
# - "Event published successfully"
# - HTTP errors (403 = bad token, 429 = rate limited)
```

### Server issues
```bash
# Check all services
docker-compose ps

# View logs
docker-compose logs -f ingest
docker-compose logs -f aggregator
docker-compose logs -f web

# Check database
docker-compose exec postgres psql -U telemetry -c "SELECT COUNT(*) FROM agg_1m;"
```

## Example deployment: 3 offices + remote workers

```
Topology:
┌─────────────┐
│ HQ Server   │ (cloud VM, 4 vCPUs, 8GB RAM)
│ 192.0.2.10  │ Runs: docker-compose
└──────┬──────┘
       │
       ├─── Office 1 (5 probes)
       ├─── Office 2 (3 probes)
       ├─── Office 3 (2 probes)
       └─── Remote workers (10 laptops)
```

**Setup:**
1. Deploy server on cloud VM
2. Open ports 8080, 3000 in security group
3. Install probe on all machines with same server URL
4. Use different tokens per office for tracking
5. Monitor from web UI

**Cost:** ~$40/month for server (DigitalOcean 8GB VM), probes are free.
