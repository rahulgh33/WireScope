# Remote Setup Guide: Mac Server + Remote Desktop Client

This guide will help you set up WireScope with:
- **Server**: Your Mac (192.168.1.39)
- **Client**: Remote desktop running the probe

## Step 1: Prepare Your Mac (Server)

### 1.1 Start the server services

```bash
cd /path/to/WireScope

# Start all services
docker-compose up -d

# Verify services are running
docker-compose ps
```

You should see these services running:
- postgres
- nats
- prometheus
- grafana
- jaeger
- otel-collector

### 1.2 Build and start the Go services

```bash
# Build all binaries
make build

# Start ingest API (receives probe data)
./bin/ingest \
  --nats-url nats://localhost:4222 \
  --api-token demo-token \
  --port 8081 &

# Start aggregator (processes data)
./bin/aggregator \
  --nats-url nats://localhost:4222 \
  --database-url "postgres://telemetry:telemetry@localhost:5432/telemetry?sslmode=disable" &

# Start diagnoser (identifies issues)
./bin/diagnoser \
  --database-url "postgres://telemetry:telemetry@localhost:5432/telemetry?sslmode=disable" &

# Optional: Start AI agent (natural language queries)
./bin/ai-agent \
  --database-url "postgres://telemetry:telemetry@localhost:5432/telemetry?sslmode=disable" &
```

Or use the convenience script:
```bash
./scripts/start-services.sh
```

### 1.3 Check your Mac's firewall settings

You need to allow incoming connections on port 8081 (ingest API).

**Option A: Using GUI**
1. System Preferences → Security & Privacy → Firewall
2. Click "Firewall Options"
3. Add the `ingest` binary or allow all incoming connections

**Option B: Using command line**
```bash
# Allow the ingest binary
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --add ./bin/ingest
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --unblockapp ./bin/ingest
```

### 1.4 Get your Mac's IP address

Your current IP is: **192.168.1.39**

To verify or get updated IP:
```bash
ifconfig | grep "inet " | grep -v 127.0.0.1
```

### 1.5 Test the server is accessible

```bash
# Test locally
curl http://localhost:8081/health

# Test from another device on your network
curl http://192.168.1.39:8081/health
```

Expected response: `{"status":"ok"}`

## Step 2: Set Up Remote Desktop (Client)

### 2.1 Determine the OS of your remote desktop

Check if it's:
- Linux (Ubuntu, Debian, etc.)
- Windows
- macOS

### 2.2 Install the probe

**For Linux:**
```bash
# Download the probe
wget https://github.com/rahulgh33/WireScope/releases/latest/download/probe-linux-amd64 -O probe
chmod +x probe

# Or build from source if you have Go installed
git clone https://github.com/rahulgh33/WireScope.git
cd WireScope
go build -o probe ./cmd/probe
```

**For Windows:**
```powershell
# Download from releases
Invoke-WebRequest -Uri "https://github.com/rahulgh33/WireScope/releases/latest/download/probe-windows-amd64.exe" -OutFile "probe.exe"

# Or build from source
git clone https://github.com/rahulgh33/WireScope.git
cd WireScope
go build -o probe.exe ./cmd/probe
```

**For macOS:**
```bash
# Download the probe
curl -L https://github.com/rahulgh33/WireScope/releases/latest/download/probe-darwin-amd64 -o probe
chmod +x probe
```

### 2.3 Run the probe pointing to your Mac

**Important:** Replace `192.168.1.39` with your Mac's actual IP address if it has changed.

```bash
./probe \
  --ingest-url http://192.168.1.39:8081/events \
  --api-token demo-token \
  --target https://google.com \
  --client-id remote-desktop-1 \
  --interval 60
```

**What each flag means:**
- `--ingest-url`: Your Mac's IP + port 8081
- `--api-token`: Must match the token configured on the server
- `--target`: Website to monitor (you can change this)
- `--client-id`: Unique name for this probe (you can customize)
- `--interval`: Seconds between measurements (60 = 1 minute)

### 2.4 Running as a background service

**Linux (systemd):**
```bash
sudo nano /etc/systemd/system/wirescope-probe.service
```

Paste this content:
```ini
[Unit]
Description=WireScope Network Probe
After=network.target

[Service]
Type=simple
User=YOUR_USERNAME
WorkingDirectory=/home/YOUR_USERNAME
ExecStart=/home/YOUR_USERNAME/probe \
  --ingest-url http://192.168.1.39:8081/events \
  --api-token demo-token \
  --target https://google.com \
  --client-id remote-desktop-1 \
  --interval 60
Restart=always

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable wirescope-probe
sudo systemctl start wirescope-probe
sudo systemctl status wirescope-probe
```

**Windows (Task Scheduler):**
1. Open Task Scheduler
2. Create Basic Task
3. Trigger: When the computer starts
4. Action: Start a program
5. Program: `C:\path\to\probe.exe`
6. Arguments: `--ingest-url http://192.168.1.39:8081/events --api-token demo-token --target https://google.com --client-id remote-desktop-1 --interval 60`

## Step 3: Verify Everything Works

### 3.1 Check probe logs

The probe should output logs showing measurements:
```
2025-12-31 10:00:00 [INFO] Starting probe remote-desktop-1
2025-12-31 10:00:00 [INFO] Target: https://google.com
2025-12-31 10:00:00 [INFO] Ingest URL: http://192.168.1.39:8081/events
2025-12-31 10:00:01 [INFO] DNS: 15ms, TCP: 25ms, TLS: 45ms, HTTP: 120ms
2025-12-31 10:00:01 [INFO] Event sent successfully
```

### 3.2 Check server logs

On your Mac, check the ingest service logs:
```bash
# If using the start script
tail -f logs/ingest.log

# Or check docker logs if running in docker
docker-compose logs -f
```

You should see incoming events from the remote probe.

### 3.3 Access the Web UI

Open your browser and go to:
- **Web UI**: http://localhost:3000 (or http://192.168.1.39:3000)
- **Grafana**: http://localhost:3000
- **Prometheus**: http://localhost:9090

You should see data from your remote probe appearing in the dashboard!

## Troubleshooting

### Probe can't connect to server

1. **Check server is running:**
   ```bash
   curl http://localhost:8081/health
   ```

2. **Check firewall on Mac:**
   ```bash
   # Temporarily disable to test
   sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate off
   # Re-enable after testing
   sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate on
   ```

3. **Check if both machines are on same network:**
   - If remote desktop is on different network, you'll need port forwarding or VPN
   - Your Mac needs to be reachable from the remote desktop's network

4. **Test with telnet/nc:**
   ```bash
   # From remote desktop
   telnet 192.168.1.39 8081
   # or
   nc -zv 192.168.1.39 8081
   ```

### No data showing in UI

1. **Check database connection:**
   ```bash
   docker exec -it wirescope-postgres-1 psql -U telemetry -d telemetry -c "SELECT COUNT(*) FROM telemetry_events;"
   ```

2. **Check NATS connection:**
   ```bash
   curl http://localhost:8222/streaming/channelsz
   ```

3. **Restart aggregator:**
   ```bash
   pkill aggregator
   ./bin/aggregator --nats-url nats://localhost:4222 --database-url "postgres://telemetry:telemetry@localhost:5432/telemetry?sslmode=disable" &
   ```

### Multiple targets

To monitor multiple websites from your remote probe:

```bash
./probe \
  --ingest-url http://192.168.1.39:8081/events \
  --api-token demo-token \
  --target https://google.com \
  --target https://cloudflare.com \
  --target https://github.com \
  --client-id remote-desktop-1 \
  --interval 60
```

## Advanced: Internet-Facing Setup

If your remote desktop is on a different network (not same LAN):

### Option 1: Port Forwarding
1. Configure your router to forward port 8081 to 192.168.1.39:8081
2. Use your public IP or domain name in the probe configuration
3. **Security warning:** Enable authentication and HTTPS in production!

### Option 2: VPN/Tunnel
Use ngrok or similar:
```bash
# On your Mac
ngrok http 8081

# Note the URL (e.g., https://abc123.ngrok.io)
# Use this URL in your probe configuration
```

### Option 3: Cloud Server
Deploy the server stack to AWS/GCP/Azure and point both your Mac and remote desktop probes to it.

## Next Steps

1. ✅ Server running on Mac
2. ✅ Probe running on remote desktop
3. ✅ Data flowing and visible in dashboard
4. 📊 Explore the UI and see metrics
5. 🔍 Try the diagnoser to identify network issues
6. 🤖 Use AI agent to ask questions about your data

Enjoy monitoring your network! 🚀
