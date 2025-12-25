# Authentication & Configuration Guide

## Configuring User Authentication

WireScope now supports configurable username and password authentication instead of hardcoded credentials.

### Environment Variables

Set custom users using the `AUTH_USERS` environment variable:

```bash
# Format: username:password:role,username2:password2:role2
export AUTH_USERS="alice:SecurePass123:admin,bob:ViewPass456:viewer,charlie:OpPass789:operator"
```

### Individual User Passwords

For the default users (admin, viewer, operator), you can set individual passwords:

```bash
export ADMIN_PASSWORD="your-secure-admin-password"
export VIEWER_PASSWORD="your-viewer-password"
export OPERATOR_PASSWORD="your-operator-password"
```

### User Roles

Three roles are supported:

- **admin**: Full access to all features including system settings and user management
- **operator**: Can manage probes, targets, and view diagnostics
- **viewer**: Read-only access to dashboards and metrics

### Docker Compose Example

```yaml
services:
  ingest:
    environment:
      - AUTH_USERS=alice:SecurePass123:admin,bob:ViewPass456:viewer
      # Or use individual passwords:
      - ADMIN_PASSWORD=MySecureAdminPass123
      - VIEWER_PASSWORD=ViewerPass456
```

## HTTPS/TLS Configuration

Enable HTTPS for secure communications:

### Using Certificate Files

Create a configuration file (e.g., `config.yaml`):

```yaml
server:
  port: 8443
  tls:
    enabled: true
    cert_file: /path/to/cert.pem
    key_file: /path/to/key.pem
    min_version: "1.2"  # TLS 1.2 or 1.3
```

### Generating Self-Signed Certificates (Development Only)

```bash
# Generate self-signed certificate
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes \
  -subj "/CN=localhost"
```

### Using Let's Encrypt (Production)

1. Install Certbot:
```bash
sudo apt-get install certbot
```

2. Get certificate:
```bash
sudo certbot certonly --standalone -d yourdomain.com
```

3. Update config:
```yaml
server:
  tls:
    enabled: true
    cert_file: /etc/letsencrypt/live/yourdomain.com/fullchain.pem
    key_file: /etc/letsencrypt/live/yourdomain.com/privkey.pem
```

### Nginx Reverse Proxy (Recommended for Production)

```nginx
server {
    listen 443 ssl http2;
    server_name yourdomain.com;
    
    ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    # WebSocket support
    location /api/v1/ws {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## Redis Rate Limiting

Enable distributed rate limiting with Redis:

### Configuration

```yaml
rate_limit:
  enabled: true
  redis:
    enabled: true
    address: "localhost:6379"
    password: ""  # Optional
    db: 0
  requests_per_second: 100
  window_seconds: 60
```

### Environment Variables

```bash
export REDIS_URL="redis://localhost:6379"
export RATE_LIMIT_RPS=100
export RATE_LIMIT_WINDOW=60
```

### Docker Compose with Redis

```yaml
services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    command: redis-server --appendonly yes

  ingest:
    depends_on:
      - redis
    environment:
      - REDIS_URL=redis://redis:6379
      - RATE_LIMIT_RPS=100

volumes:
  redis_data:
```

## Complete Production Example

```yaml
# config.yaml
server:
  port: 8443
  tls:
    enabled: true
    cert_file: /etc/ssl/certs/wirescope.pem
    key_file: /etc/ssl/private/wirescope.key
    min_version: "1.2"

auth:
  # Set via AUTH_USERS environment variable

rate_limit:
  enabled: true
  redis:
    enabled: true
    address: "redis:6379"
  requests_per_second: 1000
  window_seconds: 60

database:
  url: "postgres://user:pass@postgres:5432/telemetry?sslmode=require"
  max_connections: 50

nats:
  url: "nats://nats:4222"
  
logging:
  level: info
  format: json
```

## Docker Compose Production Setup

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: telemetry
      POSTGRES_USER: telemetry
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

  nats:
    image: nats:latest
    command: "-js"

  ingest:
    image: wirescope/ingest:latest
    environment:
      - AUTH_USERS=${AUTH_USERS}
      - REDIS_URL=redis://redis:6379
      - NATS_URL=nats://nats:4222
      - DATABASE_URL=postgres://telemetry:${DB_PASSWORD}@postgres:5432/telemetry
      - RATE_LIMIT_RPS=1000
    depends_on:
      - postgres
      - redis
      - nats

volumes:
  postgres_data:
  redis_data:
```

## Security Checklist

- [ ] Change default admin password
- [ ] Enable HTTPS/TLS
- [ ] Use strong, unique passwords
- [ ] Enable Redis for distributed rate limiting
- [ ] Configure firewall rules
- [ ] Set up log monitoring
- [ ] Enable database SSL
- [ ] Regular security updates
- [ ] Backup encryption keys
- [ ] Monitor failed login attempts

## Testing Your Configuration

```bash
# Test authentication
curl -X POST https://localhost:8443/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"your-password"}' \
  -k

# Test rate limiting
for i in {1..150}; do
  curl -X POST http://localhost:8081/events \
    -H "Authorization: Bearer your-token" \
    -d '{"client_id":"test","target":"https://example.com"}' &
done
wait

# Verify TLS
openssl s_client -connect localhost:8443 -showcerts
```
