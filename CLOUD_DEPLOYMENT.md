# Cloud Deployment Guide

Deploy WireScope to any cloud provider in minutes.

## Quick Deploy

### Option 1: AWS EC2 / DigitalOcean / Linode

1. **Create a VM:**
   - Ubuntu 22.04 LTS
   - 2GB RAM minimum (4GB recommended)
   - Open ports: 22, 8081, 3000, 9090

2. **SSH into the VM:**
   ```bash
   ssh ubuntu@YOUR_VM_IP
   ```

3. **Run the deployment script:**
   ```bash
   curl -sSL https://raw.githubusercontent.com/rahulgh33/WireScope/main/scripts/deploy-cloud.sh | sudo bash
   ```

4. **Done!** Your server is running at `http://YOUR_VM_IP:8081`

### Option 2: Google Cloud Platform

```bash
# Create VM
gcloud compute instances create wirescope \
  --machine-type=e2-medium \
  --zone=us-central1-a \
  --image-family=ubuntu-2204-lts \
  --image-project=ubuntu-os-cloud \
  --tags=wirescope

# Add firewall rules
gcloud compute firewall-rules create wirescope-ingest \
  --allow=tcp:8081 \
  --target-tags=wirescope

gcloud compute firewall-rules create wirescope-web \
  --allow=tcp:3000,tcp:9090 \
  --target-tags=wirescope

# SSH and deploy
gcloud compute ssh wirescope
curl -sSL https://raw.githubusercontent.com/rahulgh33/WireScope/main/scripts/deploy-cloud.sh | sudo bash
```

### Option 3: Manual Docker Setup

If you already have a VM with Docker:

```bash
git clone https://github.com/rahulgh33/WireScope.git
cd WireScope

# Build services
make build

# Start everything
docker-compose up -d
./scripts/start-services.sh

# Get your public IP
curl ifconfig.me
```

## Connect Probes

Once deployed, anyone can connect probes from anywhere:

```bash
curl -sSL https://raw.githubusercontent.com/rahulgh33/WireScope/main/quick-probe.sh | bash -s -- YOUR_SERVER_IP
```

## Cloud Provider Quick Links

### AWS
- **Launch EC2:** https://console.aws.amazon.com/ec2/
- **AMI:** Ubuntu Server 22.04 LTS
- **Instance Type:** t3.medium or better
- **Security Group:** Allow 22, 8081, 3000, 9090

### DigitalOcean
- **Create Droplet:** https://cloud.digitalocean.com/droplets/new
- **Image:** Ubuntu 22.04
- **Size:** Basic $12/month or better
- **Firewall:** Allow 22, 8081, 3000, 9090

### Google Cloud
- **Create VM:** https://console.cloud.google.com/compute/instances
- **Machine Type:** e2-medium
- **Boot Disk:** Ubuntu 22.04 LTS
- **Firewall:** Create rules for 8081, 3000, 9090

### Linode
- **Create Linode:** https://cloud.linode.com/linodes/create
- **Image:** Ubuntu 22.04 LTS
- **Plan:** Shared 4GB or better
- **Firewall:** Allow 22, 8081, 3000, 9090

## Cost Estimates

**Monthly costs (approximate):**
- AWS EC2 t3.medium: $30-35
- DigitalOcean Droplet: $12-24
- GCP e2-medium: $25-30
- Linode 4GB: $24

**Free tier options:**
- AWS: 750 hours/month t2.micro for 12 months
- GCP: $300 credit for 90 days
- Oracle Cloud: Always free tier

## Monitoring

After deployment:
- **Grafana:** `http://YOUR_IP:3000` (admin/admin)
- **Prometheus:** `http://YOUR_IP:9090`
- **Health Check:** `curl http://YOUR_IP:8081/health`

## Troubleshooting

**Services not starting?**
```bash
cd /opt/wirescope
./scripts/status-services.sh
tail -f logs/*.log
```

**Firewall issues?**
```bash
sudo ufw status
sudo ufw allow 8081/tcp
```

**Docker issues?**
```bash
docker-compose ps
docker-compose logs
```

## Security

**Important for production:**
1. Change default API tokens in `/opt/wirescope/.env`
2. Change Grafana password
3. Enable HTTPS (use Caddy or nginx with Let's Encrypt)
4. Restrict database access
5. Use strong passwords

## Updating

```bash
cd /opt/wirescope
git pull
make build
./scripts/stop-services.sh
docker-compose down
docker-compose up -d
./scripts/start-services.sh
```
