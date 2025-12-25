# Multi-Machine Deployment Guide

This guide describes how to deploy the Network QoE Telemetry Platform across multiple machines for production use.

## Architecture Overview

```
[Probe Machines]     [Ingest Machines]     [Processing Machines]     [Storage Machines]
├─ probe-1          ├─ ingest-1           ├─ aggregator-1           ├─ postgres-1
├─ probe-2          ├─ ingest-2           ├─ aggregator-2           ├─ postgres-2 (replica)
└─ probe-N          └─ ingest-N           └─ diagnoser-1            └─ nats-cluster
                                          └─ nats-1,2,3
```

## Prerequisites

- Docker and Docker Compose on all machines
- Network connectivity between all machines
- Synchronized time (NTP) across all machines
- DNS resolution or `/etc/hosts` entries for service discovery

## Machine Requirements

### Storage Machines (postgres, nats)
- **CPU**: 4+ cores
- **Memory**: 8GB+ RAM
- **Storage**: 100GB+ SSD for database, 50GB+ for NATS
- **Network**: Stable, low-latency connections

### Processing Machines (aggregator, diagnoser)
- **CPU**: 2+ cores
- **Memory**: 4GB+ RAM
- **Storage**: 20GB+ for logs
- **Network**: Stable connection to storage and ingest

### Ingest Machines (ingest API)
- **CPU**: 2+ cores per service instance
- **Memory**: 2GB+ RAM per instance
- **Storage**: 10GB+ for logs
- **Network**: High bandwidth for probe connections

### Probe Machines
- **CPU**: 1+ core
- **Memory**: 512MB+ RAM
- **Storage**: 5GB+ for logs
- **Network**: Representative of target user network conditions

## Firewall Rules

### Ingest Machines (receive probe traffic)
```bash
# Allow probe connections to ingest API
iptables -A INPUT -p tcp --dport 8080 -j ACCEPT
# Allow monitoring
iptables -A INPUT -p tcp --dport 9090 -j ACCEPT
```

### Storage Machines
```bash
# PostgreSQL
iptables -A INPUT -p tcp --dport 5432 -s <processing_subnet> -j ACCEPT
# NATS JetStream
iptables -A INPUT -p tcp --dport 4222 -s <processing_subnet> -s <ingest_subnet> -j ACCEPT
# NATS monitoring
iptables -A INPUT -p tcp --dport 8222 -s <monitoring_subnet> -j ACCEPT
```

### Processing Machines
```bash
# Aggregator metrics
iptables -A INPUT -p tcp --dport 9091 -s <monitoring_subnet> -j ACCEPT
# Diagnoser metrics
iptables -A INPUT -p tcp --dport 9092 -s <monitoring_subnet> -j ACCEPT
```

### Monitoring Infrastructure
```bash
# Prometheus
iptables -A INPUT -p tcp --dport 9090 -j ACCEPT
# Grafana
iptables -A INPUT -p tcp --dport 3000 -j ACCEPT
# Jaeger UI
iptables -A INPUT -p tcp --dport 16686 -j ACCEPT
# OpenTelemetry Collector
iptables -A INPUT -p tcp --dport 4317 -j ACCEPT
iptables -A INPUT -p tcp --dport 4318 -j ACCEPT
```

## DNS/Host Configuration

Add these entries to `/etc/hosts` on all machines:

```
# Storage tier
10.0.1.10    postgres-primary
10.0.1.11    postgres-replica
10.0.1.20    nats-1
10.0.1.21    nats-2
10.0.1.22    nats-3

# Processing tier
10.0.2.10    aggregator-1
10.0.2.11    aggregator-2
10.0.2.20    diagnoser-1

# Ingest tier
10.0.3.10    ingest-1
10.0.3.11    ingest-2
10.0.3.12    ingest-3

# Monitoring
10.0.4.10    prometheus
10.0.4.11    grafana
10.0.4.12    jaeger
```

## Deployment Steps

### 1. Storage Machines

Deploy PostgreSQL primary and NATS cluster first:

```bash
# On postgres-primary machine (10.0.1.10)
docker compose -f docker-compose.storage.yml up -d postgres-primary

# On postgres-replica machine (10.0.1.11) 
docker compose -f docker-compose.storage.yml up -d postgres-replica

# On NATS machines (10.0.1.20-22)
docker compose -f docker-compose.nats-cluster.yml up -d
```

### 2. Processing Machines

Deploy aggregator and diagnoser services:

```bash
# On aggregator machines (10.0.2.10-11)
docker compose -f docker-compose.processing.yml up -d aggregator

# On diagnoser machine (10.0.2.20)
docker compose -f docker-compose.processing.yml up -d diagnoser
```

### 3. Ingest Machines

Deploy ingest API services with load balancing:

```bash
# On each ingest machine (10.0.3.10-12)
docker compose -f docker-compose.ingest.yml up -d
```

### 4. Probe Machines

Deploy probe agents across target locations:

```bash
# On each probe machine
docker compose -f docker-compose.probe.yml up -d
```

### 5. Monitoring Infrastructure

Deploy observability stack:

```bash
# On monitoring machines
docker compose -f docker-compose.monitoring.yml up -d
```

## Service Discovery

Use Docker's built-in DNS resolution with external networks:

```bash
# Create shared networks
docker network create --driver bridge telemetry-storage
docker network create --driver bridge telemetry-processing  
docker network create --driver bridge telemetry-ingest
docker network create --driver bridge telemetry-monitoring
```

## Health Checks

### Database Connectivity
```bash
# Test from aggregator machine
psql -h postgres-primary -U telemetry -d telemetry -c "SELECT 1"
```

### NATS Connectivity
```bash
# Test from ingest machine
nats server check --server=nats://nats-1:4222
```

### End-to-End Flow
```bash
# Send test event from probe
curl -X POST http://ingest-1:8080/events \
  -H "Authorization: Bearer test-token" \
  -H "Content-Type: application/json" \
  -d @test-event.json
```

## Failure Scenarios

### Database Failover
When postgres-primary fails:
1. Promote postgres-replica to primary
2. Update aggregator configurations
3. Restart aggregator services

### NATS Node Failure
NATS cluster tolerates 1-node failures automatically.
For 2+ node failures:
1. Restore quorum with remaining nodes
2. Re-add failed nodes when available

### Ingest Node Failure
Load balancer automatically routes around failed ingest nodes.
No manual intervention required.

### Network Partitions
- Configure probe timeouts and retry logic
- Use NATS persistence to buffer events during outages
- Monitor queue depths for backpressure

## Monitoring Multi-Machine Setup

Additional monitoring for distributed deployment:

- **Network latency** between machines
- **Cross-machine connectivity** health
- **Service discovery** resolution times
- **Load distribution** across ingest nodes
- **Database replication lag** between primary/replica

## Scaling Recommendations

### Horizontal Scaling
- **Ingest**: Add more ingest machines behind load balancer
- **Processing**: Add aggregator instances (stateless)
- **Storage**: Use PostgreSQL read replicas for query scaling

### Vertical Scaling
- **Database**: Increase memory and CPU for PostgreSQL primary
- **NATS**: Increase memory for larger message buffers
- **Aggregator**: Increase memory for larger in-memory windows

## Security Considerations

- Use TLS for all inter-service communication
- Implement network segmentation with VLANs
- Use service accounts with minimal privileges
- Regular security updates for all components
- Monitor for unusual traffic patterns