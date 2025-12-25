# Scaling Guide

## When to scale

- Ingest API: > 1000 req/sec sustained
- Aggregator: Queue lag > 1000 messages for > 5 minutes
- Database: Connection pool exhaustion, slow queries

## Horizontal scaling

### Ingest API

Multiple instances behind load balancer:
```bash
# Run 3 instances
docker-compose up --scale ingest=3
```

Use Redis for shared rate limiting across instances.

### Aggregator

NATS consumer groups automatically load balance:
```bash
# Run 2 aggregators - they'll share the work
docker-compose up --scale aggregator=2
```

### Database

Read replicas for queries:
```sql
-- Route reads to replica
SELECT * FROM agg_1m WHERE ... -- Use replica
```

Partition tables by time:
```sql
-- Partition agg_1m by month
CREATE TABLE agg_1m_202501 PARTITION OF agg_1m
  FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
```

## Monitoring scaled deployments

Key metrics:
- Load balancer request distribution
- Per-instance CPU/memory
- Queue consumer lag per instance
- Database connection count per instance

## Limits

Current architecture tested to:
- 10k events/sec (10 aggregator instances)
- 100k probes (distributed)
- 1 year data retention (with partitioning)
