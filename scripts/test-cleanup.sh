#!/bin/bash
# Test script for database cleanup functionality

set -e

echo "=== Database Cleanup Test ==="
echo ""

# Database connection
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-telemetry}"
DB_USER="${DB_USER:-telemetry}"
export PGPASSWORD="${DB_PASSWORD:-telemetry}"

PSQL="psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -t -c"

# 1. Insert test data
echo "1. Inserting test data..."

# Insert old events (10 days ago)
$PSQL "
INSERT INTO events_seen (event_id, client_id, ts_ms, created_at)
VALUES 
    (gen_random_uuid(), 'test-client-1', 1000000, NOW() - INTERVAL '10 days'),
    (gen_random_uuid(), 'test-client-2', 1000001, NOW() - INTERVAL '10 days'),
    (gen_random_uuid(), 'test-client-3', 1000002, NOW() - INTERVAL '10 days');
" > /dev/null

# Insert recent events (1 day ago)
$PSQL "
INSERT INTO events_seen (event_id, client_id, ts_ms, created_at)
VALUES 
    (gen_random_uuid(), 'test-client-4', 1000003, NOW() - INTERVAL '1 day'),
    (gen_random_uuid(), 'test-client-5', 1000004, NOW() - INTERVAL '1 day');
" > /dev/null

# Insert old aggregates (100 days ago)
$PSQL "
INSERT INTO agg_1m (client_id, target, window_start_ts, count_total, count_success)
VALUES 
    ('test-client-1', 'old.example.com', NOW() - INTERVAL '100 days', 10, 10),
    ('test-client-2', 'old.example.com', NOW() - INTERVAL '100 days', 10, 10);
" > /dev/null

# Insert recent aggregates (1 day ago)
$PSQL "
INSERT INTO agg_1m (client_id, target, window_start_ts, count_total, count_success)
VALUES 
    ('test-client-3', 'new.example.com', NOW() - INTERVAL '1 day', 10, 10),
    ('test-client-4', 'new.example.com', NOW() - INTERVAL '1 day', 10, 10);
" > /dev/null

echo "✓ Test data inserted"

# 2. Check initial counts
echo ""
echo "2. Checking initial data counts..."
EVENTS_COUNT=$($PSQL "SELECT COUNT(*) FROM events_seen;" | xargs)
AGG_COUNT=$($PSQL "SELECT COUNT(*) FROM agg_1m;" | xargs)
echo "   events_seen: $EVENTS_COUNT records"
echo "   agg_1m: $AGG_COUNT records"

# 3. Run dry-run cleanup
echo ""
echo "3. Running cleanup (dry-run)..."
./bin/cleanup -dry-run -events-retention-days=7 -agg-retention-days=90

# 4. Verify no data deleted in dry-run
echo ""
echo "4. Verifying dry-run didn't delete data..."
EVENTS_AFTER_DRYRUN=$($PSQL "SELECT COUNT(*) FROM events_seen;" | xargs)
AGG_AFTER_DRYRUN=$($PSQL "SELECT COUNT(*) FROM agg_1m;" | xargs)

if [ "$EVENTS_AFTER_DRYRUN" -eq "$EVENTS_COUNT" ] && [ "$AGG_AFTER_DRYRUN" -eq "$AGG_COUNT" ]; then
    echo "✓ Dry-run correctly preserved all data"
else
    echo "✗ ERROR: Dry-run deleted data!"
    exit 1
fi

# 5. Run actual cleanup
echo ""
echo "5. Running actual cleanup..."
./bin/cleanup -events-retention-days=7 -agg-retention-days=90

# 6. Verify correct data deleted
echo ""
echo "6. Verifying cleanup results..."
EVENTS_AFTER=$($PSQL "SELECT COUNT(*) FROM events_seen;" | xargs)
AGG_AFTER=$($PSQL "SELECT COUNT(*) FROM agg_1m;" | xargs)
OLD_EVENTS_REMAINING=$($PSQL "SELECT COUNT(*) FROM events_seen WHERE created_at < NOW() - INTERVAL '7 days';" | xargs)
OLD_AGG_REMAINING=$($PSQL "SELECT COUNT(*) FROM agg_1m WHERE window_start_ts < NOW() - INTERVAL '90 days';" | xargs)

echo "   events_seen: $EVENTS_AFTER records (expected: 2)"
echo "   agg_1m: $AGG_AFTER records (expected: 2)"
echo "   old events remaining: $OLD_EVENTS_REMAINING (expected: 0)"
echo "   old aggregates remaining: $OLD_AGG_REMAINING (expected: 0)"

if [ "$EVENTS_AFTER" -eq 2 ] && [ "$AGG_AFTER" -eq 2 ] && \
   [ "$OLD_EVENTS_REMAINING" -eq 0 ] && [ "$OLD_AGG_REMAINING" -eq 0 ]; then
    echo "✓ Cleanup correctly deleted old data and preserved recent data"
else
    echo "✗ ERROR: Cleanup did not work as expected!"
    exit 1
fi

# 7. Run health check
echo ""
echo "7. Running health check..."
./bin/cleanup -health-check

# 8. Cleanup test data
echo ""
echo "8. Cleaning up test data..."
$PSQL "DELETE FROM events_seen WHERE client_id LIKE 'test-client-%';" > /dev/null
$PSQL "DELETE FROM agg_1m WHERE client_id LIKE 'test-client-%';" > /dev/null
echo "✓ Test data cleaned up"

echo ""
echo "=== All cleanup tests passed! ==="
