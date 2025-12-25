# Deployment

## Local (Docker Compose)

```bash
docker-compose up -d
```

Services:
- Ingest API: http://localhost:8080
- Web UI: http://localhost:3000
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3001

## Kubernetes

```bash
kubectl apply -f deploy/k8s/
```

Or use Helm:
```bash
helm install wirescope deploy/helm/
```

Required:
- PostgreSQL (use CloudSQL, RDS, or external instance)
- NATS (nats.io Helm chart or external)

## Cloud (AWS)

```bash
./scripts/deploy-aws.sh
```

Creates:
- ECS cluster with ingest, aggregator, diagnoser, ai-agent services
- RDS PostgreSQL instance
- ALB for ingest API
- S3 bucket for static UI assets

## Distributed probes

Install probe on remote machines:
```bash
curl -sSL https://github.com/rahulgh33/WireScope/releases/latest/download/install.sh | bash
```

Configure probe:
```yaml
server_url: https://your-ingest-api.com
api_key: your-key-here
targets:
  - endpoint: google.com
    port: 443
```

## Environment variables

See `config/*.example.yaml` for all configuration options.

Key variables:
- `DATABASE_URL`: PostgreSQL connection string
- `NATS_URL`: NATS server URL
- `OPENAI_API_KEY`: For AI agent (optional)
- `JWT_SECRET`: For multi-tenancy authentication

## Monitoring

Check service health:
```bash
curl http://localhost:8080/health
```

View metrics:
```bash
curl http://localhost:8080/metrics
```

## Backups

Database:
```bash
pg_dump $DATABASE_URL > backup.sql
```

NATS (for message retention):
```bash
nats stream backup telemetry-stream
```
