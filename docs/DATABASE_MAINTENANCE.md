# Database Maintenance Guide

## Overview

This guide covers database maintenance operations for the Network QoE Telemetry Platform, including cleanup, health checks, and monitoring.

## Cleanup Utility

The cleanup utility (`bin/cleanup`) provides automated database maintenance for removing old data and monitoring database health.

### Features

- **Events Cleanup**: Removes deduplication records from `events_seen` table older than configured retention period (default: 7 days)
- **Aggregates Cleanup**: Removes old aggregate data from `agg_1m` table older than configured retention period (default: 90 days)
- **Health Checks**: Comprehensive database health monitoring including connections, table sizes, locks, and performance metrics
- **Dry Run Mode**: Preview cleanup operations without actually deleting data

### Usage

#### Build the Cleanup Utility

```bash
make build
# or specifically
go build -o bin/cleanup ./cmd/cleanup
```

#### Run Cleanup (Dry Run)

Preview what would be deleted:

```bash
./bin/cleanup -dry-run
# or using make
make db-cleanup
```

#### Run Actual Cleanup

Execute cleanup with actual data deletion:

```bash
./bin/cleanup
# or using make
make db-cleanup-force
```

#### Run Health Check Only

```bash
./bin/cleanup -health-check
# or using make
make db-health
```

### Configuration Options

All configuration can be provided via command-line flags or environment variables:

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `-db-host` | `DB_HOST` | `localhost` | Database host |
| `-db-port` | `DB_PORT` | `5432` | Database port |
| `-db-name` | `DB_NAME` | `telemetry` | Database name |
| `-db-user` | `DB_USER` | `telemetry` | Database user |
| `-db-password` | `DB_PASSWORD` | `telemetry` | Database password |
| `-events-retention-days` | - | `7` | Days to retain events_seen data |
| `-agg-retention-days` | - | `90` | Days to retain aggregate data |
| `-dry-run` | - | `false` | Preview mode (no deletion) |
| `-health-check` | - | `false` | Run health check only |

### Examples

#### Custom Retention Periods

```bash
# Keep events for 14 days and aggregates for 180 days
./bin/cleanup -events-retention-days=14 -agg-retention-days=180
```

#### Using Environment Variables

```bash
export DB_HOST=postgres.example.com
export DB_USER=telemetry_user
export DB_PASSWORD=secure_password
./bin/cleanup
```

#### Production with Specific Database

```bash
./bin/cleanup \
  -db-host=prod-db.internal \
  -db-user=telemetry_admin \
  -db-password=secret \
  -events-retention-days=3 \
  -agg-retention-days=30
```

## Scheduled Cleanup

### Using Cron

Add to crontab for daily cleanup at 2 AM:

```bash
crontab -e
```

Add this line:

```
0 2 * * * /path/to/Distributed-Telemetry-Platform/scripts/daily-cleanup.sh >> /var/log/telemetry-cleanup.log 2>&1
```

The `daily-cleanup.sh` script supports these environment variables:

```bash
# In your crontab or systemd service
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=telemetry
export DB_USER=telemetry
export DB_PASSWORD=telemetry
export EVENTS_RETENTION_DAYS=7
export AGG_RETENTION_DAYS=90
export LOG_FILE=/var/log/telemetry-cleanup.log
```

### Using Systemd Timer

Create `/etc/systemd/system/telemetry-cleanup.service`:

```ini
[Unit]
Description=Telemetry Database Cleanup
After=postgresql.service

[Service]
Type=oneshot
User=telemetry
Environment="DB_HOST=localhost"
Environment="DB_PORT=5432"
Environment="DB_NAME=telemetry"
Environment="DB_USER=telemetry"
Environment="DB_PASSWORD=telemetry"
Environment="EVENTS_RETENTION_DAYS=7"
Environment="AGG_RETENTION_DAYS=90"
ExecStart=/opt/telemetry/scripts/daily-cleanup.sh
StandardOutput=journal
StandardError=journal
```

Create `/etc/systemd/system/telemetry-cleanup.timer`:

```ini
[Unit]
Description=Daily Telemetry Database Cleanup Timer
Requires=telemetry-cleanup.service

[Timer]
OnCalendar=daily
OnCalendar=02:00
Persistent=true

[Install]
WantedBy=timers.target
```

Enable and start the timer:

```bash
sudo systemctl daemon-reload
sudo systemctl enable telemetry-cleanup.timer
sudo systemctl start telemetry-cleanup.timer
sudo systemctl status telemetry-cleanup.timer
```

### Using Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: telemetry-cleanup
  namespace: telemetry
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  concurrencyPolicy: Forbid
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 3
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: OnFailure
          containers:
          - name: cleanup
            image: telemetry-platform:latest
            command: ["/app/bin/cleanup"]
            args:
              - "-events-retention-days=7"
              - "-agg-retention-days=90"
            env:
              - name: DB_HOST
                value: postgres-service
              - name: DB_PORT
                value: "5432"
              - name: DB_NAME
                value: telemetry
              - name: DB_USER
                valueFrom:
                  secretKeyRef:
                    name: postgres-credentials
                    key: username
              - name: DB_PASSWORD
                valueFrom:
                  secretKeyRef:
                    name: postgres-credentials
                    key: password
```

## Health Check Details

The health check provides comprehensive database monitoring:

### Checks Performed

1. **Database Connectivity**: Basic ping test
2. **Connection Pool Stats**: Open/in-use/idle connections
3. **Table Sizes**: Physical size of events_seen and agg_1m tables
4. **Row Counts**: Number of records in each table
5. **Data Age**: Oldest record in each table
6. **Long-Running Queries**: Queries running longer than 5 minutes
7. **Blocked Queries**: Queries waiting for locks
8. **Index Health**: Top 5 largest indexes
9. **Vacuum Status**: Tables with excessive dead tuples

### Sample Health Check Output

```
✓ Database connectivity OK
✓ Connection pool: Open=5 InUse=2 Idle=3 MaxOpen=25
✓ Table sizes: events_seen=245 MB agg_1m=1.2 GB
✓ Row counts: events_seen=1250000 agg_1m=4320000
✓ Oldest event: 2024-12-16T02:15:30Z (age: 168h)
✓ Oldest aggregate: 2024-10-01T00:00:00Z (age: 2016h)
✓ No long-running queries
✓ No blocked queries
✓ Top 5 indexes:
  - public.agg_1m.agg_1m_pkey: 512 MB
  - public.agg_1m.idx_agg_1m_window: 256 MB
  - public.events_seen.events_seen_pkey: 128 MB
  - public.agg_1m.idx_agg_1m_client_target_window: 64 MB
  - public.events_seen.idx_events_seen_client_ts: 32 MB
✓ All tables are well-maintained (no excessive dead tuples)

✓ Database health check completed successfully
```

### Warning Indicators

The health check will warn about:

- **Long-Running Queries**: Queries active for more than 5 minutes
- **Blocked Queries**: Queries waiting for locks
- **Dead Tuples**: Tables with more than 1000 dead tuples (needs VACUUM)

## Performance Considerations

### Cleanup Operations

- **Transaction Size**: Cleanup runs in single transactions per table
- **Lock Duration**: DELETE operations acquire row locks; may impact concurrent writes
- **Statistics Update**: Runs ANALYZE after cleanup to update query planner statistics

### Best Practices

1. **Run During Off-Peak Hours**: Schedule cleanup when traffic is low (e.g., 2-4 AM)
2. **Monitor Duration**: Track cleanup execution time to detect performance degradation
3. **Test Dry Run First**: Always test with `-dry-run` before production execution
4. **Alert on Failures**: Configure monitoring to alert if cleanup fails
5. **Regular Health Checks**: Run health checks daily to catch issues early

### Batch Deletion (Future Enhancement)

For very large tables, consider implementing batch deletion:

```sql
-- Example batch deletion (not yet implemented)
DELETE FROM events_seen
WHERE created_at < $1
AND ctid IN (
    SELECT ctid FROM events_seen
    WHERE created_at < $1
    LIMIT 10000
);
```

## Monitoring and Alerts

### Metrics to Monitor

1. **Cleanup Duration**: Time taken for each cleanup run
2. **Records Deleted**: Number of records removed per run
3. **Table Growth Rate**: Daily increase in table sizes
4. **Oldest Record Age**: Age of oldest data in each table
5. **Health Check Status**: Success/failure of health checks

### Recommended Alerts

- **Cleanup Failure**: Alert if cleanup exits with non-zero status
- **Excessive Table Size**: Alert if table size exceeds threshold
- **Old Data Retention**: Alert if oldest data exceeds retention + grace period
- **High Dead Tuples**: Alert if dead tuple percentage > 20%
- **Database Locks**: Alert if blocked queries detected

### Prometheus Metrics (Future Enhancement)

```prometheus
# Example metrics that could be exported
telemetry_cleanup_duration_seconds{table="events_seen"}
telemetry_cleanup_records_deleted{table="events_seen"}
telemetry_db_table_size_bytes{table="events_seen"}
telemetry_db_table_rows{table="events_seen"}
telemetry_db_oldest_record_age_seconds{table="events_seen"}
```

## Troubleshooting

### Cleanup Takes Too Long

**Symptoms**: Cleanup runs for hours

**Solutions**:
- Reduce retention periods to delete less data
- Implement batch deletion (delete in chunks)
- Run VACUUM ANALYZE before cleanup to update statistics
- Check for missing indexes on timestamp columns

### Out of Disk Space

**Symptoms**: PostgreSQL errors about disk space

**Solutions**:
- Run cleanup immediately with aggressive retention
- Temporarily increase disk space
- Check for bloated tables (run VACUUM FULL during maintenance window)
- Archive old data before deletion

### Deadlocks During Cleanup

**Symptoms**: Cleanup fails with deadlock errors

**Solutions**:
- Run cleanup during off-peak hours
- Ensure no long-running transactions during cleanup
- Reduce concurrent write load during cleanup window

### Health Check Shows High Dead Tuples

**Symptoms**: Warning about tables needing vacuum

**Solutions**:
- Run manual VACUUM: `VACUUM ANALYZE events_seen;`
- Check autovacuum settings: `SHOW autovacuum;`
- Increase autovacuum frequency if needed
- Consider VACUUM FULL during maintenance window (requires exclusive lock)

## Database Configuration

### Recommended PostgreSQL Settings

```sql
-- Autovacuum settings for high-write tables
ALTER TABLE events_seen SET (
    autovacuum_vacuum_scale_factor = 0.05,
    autovacuum_analyze_scale_factor = 0.02
);

ALTER TABLE agg_1m SET (
    autovacuum_vacuum_scale_factor = 0.1,
    autovacuum_analyze_scale_factor = 0.05
);

-- Connection pooling
max_connections = 100
shared_buffers = 256MB
effective_cache_size = 1GB
```

## Future Enhancements

### Planned Features

1. **Table Partitioning**: Implement time-based partitioning for agg_1m
2. **Automated Partition Management**: Automatic creation and deletion of partitions
3. **Batch Deletion**: Delete in smaller batches to reduce lock time
4. **Metrics Export**: Export cleanup metrics to Prometheus
5. **Email Notifications**: Send email alerts on cleanup failures
6. **Data Archival**: Archive old data to S3/object storage before deletion
7. **Incremental Cleanup**: Spread cleanup operations throughout the day

### Partitioning Strategy (Future)

```sql
-- Example partitioned table structure
CREATE TABLE agg_1m (
    -- columns...
) PARTITION BY RANGE (window_start_ts);

-- Monthly partitions
CREATE TABLE agg_1m_2024_12 PARTITION OF agg_1m
    FOR VALUES FROM ('2024-12-01') TO ('2025-01-01');

-- Automated partition creation and cleanup
-- Old partitions can be dropped instantly: DROP TABLE agg_1m_2024_09;
```

## Support

For issues or questions about database maintenance:

1. Check logs in `/var/log/telemetry-cleanup.log`
2. Run health check: `make db-health`
3. Review PostgreSQL logs for errors
4. Open an issue on GitHub with cleanup logs and health check output
