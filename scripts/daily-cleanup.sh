#!/bin/bash
# Daily database cleanup script
# Can be scheduled with cron: 0 2 * * * /path/to/daily-cleanup.sh

set -e

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CLEANUP_BIN="$PROJECT_ROOT/bin/cleanup"

# Log file
LOG_FILE="${LOG_FILE:-/var/log/telemetry-cleanup.log}"

# Database connection from environment or defaults
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-telemetry}"
DB_USER="${DB_USER:-telemetry}"
DB_PASSWORD="${DB_PASSWORD:-telemetry}"

# Retention settings
EVENTS_RETENTION_DAYS="${EVENTS_RETENTION_DAYS:-7}"
AGG_RETENTION_DAYS="${AGG_RETENTION_DAYS:-90}"

# Check if cleanup binary exists
if [ ! -f "$CLEANUP_BIN" ]; then
    echo "ERROR: Cleanup binary not found at $CLEANUP_BIN"
    echo "Run 'make build' to compile the cleanup utility"
    exit 1
fi

# Log function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log "Starting daily database cleanup"
log "Events retention: $EVENTS_RETENTION_DAYS days"
log "Aggregates retention: $AGG_RETENTION_DAYS days"

# Run cleanup
"$CLEANUP_BIN" \
    -db-host="$DB_HOST" \
    -db-port="$DB_PORT" \
    -db-name="$DB_NAME" \
    -db-user="$DB_USER" \
    -db-password="$DB_PASSWORD" \
    -events-retention-days="$EVENTS_RETENTION_DAYS" \
    -agg-retention-days="$AGG_RETENTION_DAYS" \
    2>&1 | tee -a "$LOG_FILE"

EXIT_CODE=${PIPESTATUS[0]}

if [ $EXIT_CODE -eq 0 ]; then
    log "Cleanup completed successfully"
else
    log "ERROR: Cleanup failed with exit code $EXIT_CODE"
    exit $EXIT_CODE
fi

# Optional: Run health check after cleanup
log "Running post-cleanup health check"
"$CLEANUP_BIN" \
    -db-host="$DB_HOST" \
    -db-port="$DB_PORT" \
    -db-name="$DB_NAME" \
    -db-user="$DB_USER" \
    -db-password="$DB_PASSWORD" \
    -health-check \
    2>&1 | tee -a "$LOG_FILE"

log "Daily cleanup finished"
