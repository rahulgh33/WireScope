package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Repository provides common database operations
type Repository struct {
	conn *Connection
}

// NewRepository creates a new repository instance
func NewRepository(conn *Connection) *Repository {
	return &Repository{
		conn: conn,
	}
}

// Connection returns the underlying database connection
func (r *Repository) Connection() *Connection {
	return r.conn
}

// WithTransaction executes a function within a database transaction
func (r *Repository) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	return r.WithTransactionOptions(ctx, nil, fn)
}

// WithTransactionOptions executes a function within a database transaction with specific options
func (r *Repository) WithTransactionOptions(ctx context.Context, opts *sql.TxOptions, fn func(*sql.Tx) error) error {
	tx, err := r.conn.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // Re-throw panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction failed: %v, rollback failed: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// RetryableOperation executes an operation with exponential backoff retry logic
func (r *Repository) RetryableOperation(ctx context.Context, maxRetries int, operation func() error) error {
	var lastErr error
	backoff := time.Millisecond * 100

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Wait with exponential backoff
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
				if backoff > time.Second*10 {
					backoff = time.Second * 10 // Cap at 10 seconds
				}
			}
		}

		lastErr = operation()
		if lastErr == nil {
			return nil // Success
		}

		// Only retry if the error is retryable
		if !IsRetryableError(lastErr) {
			return lastErr
		}
	}

	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, lastErr)
}

// HealthCheck performs a basic health check on the database
func (r *Repository) HealthCheck(ctx context.Context) error {
	// Test basic connectivity
	if err := r.conn.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Test a simple query
	var result int
	err := r.conn.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("database query test failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("database query returned unexpected result: %d", result)
	}

	return nil
}

// GetConnectionStats returns database connection pool statistics
func (r *Repository) GetConnectionStats() sql.DBStats {
	return r.conn.Stats()
}

// EventsSeenRepository provides operations for the events_seen table
type EventsSeenRepository struct {
	*Repository
}

// NewEventsSeenRepository creates a new events_seen repository
func NewEventsSeenRepository(conn *Connection) *EventsSeenRepository {
	return &EventsSeenRepository{
		Repository: NewRepository(conn),
	}
}

// InsertEventSeen inserts an event ID into the deduplication table
// Returns true if the event was newly inserted, false if it already existed
func (r *EventsSeenRepository) InsertEventSeen(ctx context.Context, eventID, clientID string, tsMs int64) (bool, error) {
	query := `
		INSERT INTO events_seen (event_id, client_id, ts_ms) 
		VALUES ($1, $2, $3) 
		ON CONFLICT (event_id) DO NOTHING`

	result, err := r.conn.ExecContext(ctx, query, eventID, clientID, tsMs)
	if err != nil {
		return false, fmt.Errorf("failed to insert event_seen: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

// CleanupOldEvents removes events older than the specified duration
func (r *EventsSeenRepository) CleanupOldEvents(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	query := "DELETE FROM events_seen WHERE created_at < $1"

	result, err := r.conn.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old events: %w", err)
	}

	rowsDeleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get deleted rows count: %w", err)
	}

	return rowsDeleted, nil
}

// AggregatesRepository provides operations for the agg_1m table
type AggregatesRepository struct {
	*Repository
}

// NewAggregatesRepository creates a new aggregates repository
func NewAggregatesRepository(conn *Connection) *AggregatesRepository {
	return &AggregatesRepository{
		Repository: NewRepository(conn),
	}
}

// WindowedAggregate represents a time-windowed aggregate record
type WindowedAggregate struct {
	ClientID             string
	Target               string
	WindowStartTs        time.Time
	CountTotal           int64
	CountSuccess         int64
	CountError           int64
	DNSErrorCount        int64
	TCPErrorCount        int64
	TLSErrorCount        int64
	HTTPErrorCount       int64
	ThroughputErrorCount int64
	DNSP50               *float64
	DNSP95               *float64
	TCPP50               *float64
	TCPP95               *float64
	TLSP50               *float64
	TLSP95               *float64
	TTFBP50              *float64
	TTFBP95              *float64
	ThroughputP50        *float64
	ThroughputP95        *float64
	DiagnosisLabel       *string
	UpdatedAt            time.Time
}

// UpsertAggregate inserts or updates an aggregate record
func (r *AggregatesRepository) UpsertAggregate(ctx context.Context, agg *WindowedAggregate) error {
	query := `
		INSERT INTO agg_1m (
			client_id, target, window_start_ts, count_total, count_success, count_error,
			dns_error_count, tcp_error_count, tls_error_count, http_error_count, throughput_error_count,
			dns_p50, dns_p95, tcp_p50, tcp_p95, tls_p50, tls_p95, 
			ttfb_p50, ttfb_p95, throughput_p50, throughput_p95, diagnosis_label, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
		) ON CONFLICT (client_id, target, window_start_ts) 
		DO UPDATE SET 
			count_total = $4,
			count_success = $5,
			count_error = $6,
			dns_error_count = $7,
			tcp_error_count = $8,
			tls_error_count = $9,
			http_error_count = $10,
			throughput_error_count = $11,
			dns_p50 = $12,
			dns_p95 = $13,
			tcp_p50 = $14,
			tcp_p95 = $15,
			tls_p50 = $16,
			tls_p95 = $17,
			ttfb_p50 = $18,
			ttfb_p95 = $19,
			throughput_p50 = $20,
			throughput_p95 = $21,
			diagnosis_label = $22,
			updated_at = $23`

	_, err := r.conn.ExecContext(ctx, query,
		agg.ClientID, agg.Target, agg.WindowStartTs,
		agg.CountTotal, agg.CountSuccess, agg.CountError,
		agg.DNSErrorCount, agg.TCPErrorCount, agg.TLSErrorCount,
		agg.HTTPErrorCount, agg.ThroughputErrorCount,
		agg.DNSP50, agg.DNSP95, agg.TCPP50, agg.TCPP95,
		agg.TLSP50, agg.TLSP95, agg.TTFBP50, agg.TTFBP95,
		agg.ThroughputP50, agg.ThroughputP95, agg.DiagnosisLabel,
		agg.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert aggregate: %w", err)
	}

	return nil
}

// GetAggregatesByWindow retrieves aggregates for a specific time window
func (r *AggregatesRepository) GetAggregatesByWindow(ctx context.Context, windowStart time.Time) ([]*WindowedAggregate, error) {
	query := `
		SELECT client_id, target, window_start_ts, count_total, count_success, count_error,
			   dns_error_count, tcp_error_count, tls_error_count, http_error_count, throughput_error_count,
			   dns_p50, dns_p95, tcp_p50, tcp_p95, tls_p50, tls_p95,
			   ttfb_p50, ttfb_p95, throughput_p50, throughput_p95, diagnosis_label, updated_at
		FROM agg_1m 
		WHERE window_start_ts = $1
		ORDER BY client_id, target`

	rows, err := r.conn.QueryContext(ctx, query, windowStart)
	if err != nil {
		return nil, fmt.Errorf("failed to query aggregates by window: %w", err)
	}
	defer rows.Close()

	var aggregates []*WindowedAggregate
	for rows.Next() {
		agg := &WindowedAggregate{}
		err := rows.Scan(
			&agg.ClientID, &agg.Target, &agg.WindowStartTs,
			&agg.CountTotal, &agg.CountSuccess, &agg.CountError,
			&agg.DNSErrorCount, &agg.TCPErrorCount, &agg.TLSErrorCount,
			&agg.HTTPErrorCount, &agg.ThroughputErrorCount,
			&agg.DNSP50, &agg.DNSP95, &agg.TCPP50, &agg.TCPP95,
			&agg.TLSP50, &agg.TLSP95, &agg.TTFBP50, &agg.TTFBP95,
			&agg.ThroughputP50, &agg.ThroughputP95, &agg.DiagnosisLabel,
			&agg.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan aggregate row: %w", err)
		}
		aggregates = append(aggregates, agg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating aggregate rows: %w", err)
	}

	return aggregates, nil
}

// GetHistoricalAggregates fetches the most recent N windows for baseline calculation
// Used by the diagnosis engine to establish baseline metrics
func (r *Repository) GetHistoricalAggregates(ctx context.Context, clientID, target string, limit int) ([]WindowedAggregate, error) {
	query := `
		SELECT 
			client_id, target, window_start_ts,
			count_total, count_success, count_error,
			dns_error_count, tcp_error_count, tls_error_count,
			http_error_count, throughput_error_count,
			dns_p50, dns_p95, tcp_p50, tcp_p95,
			tls_p50, tls_p95, ttfb_p50, ttfb_p95,
			throughput_p50, throughput_p95, diagnosis_label,
			updated_at
		FROM agg_1m
		WHERE client_id = $1 AND target = $2
		ORDER BY window_start_ts DESC
		LIMIT $3
	`

	rows, err := r.conn.QueryContext(ctx, query, clientID, target, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query historical aggregates: %w", err)
	}
	defer rows.Close()

	var aggregates []WindowedAggregate
	for rows.Next() {
		var agg WindowedAggregate
		err := rows.Scan(
			&agg.ClientID, &agg.Target, &agg.WindowStartTs,
			&agg.CountTotal, &agg.CountSuccess, &agg.CountError,
			&agg.DNSErrorCount, &agg.TCPErrorCount, &agg.TLSErrorCount,
			&agg.HTTPErrorCount, &agg.ThroughputErrorCount,
			&agg.DNSP50, &agg.DNSP95, &agg.TCPP50, &agg.TCPP95,
			&agg.TLSP50, &agg.TLSP95, &agg.TTFBP50, &agg.TTFBP95,
			&agg.ThroughputP50, &agg.ThroughputP95, &agg.DiagnosisLabel,
			&agg.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan aggregate row: %w", err)
		}
		aggregates = append(aggregates, agg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating aggregate rows: %w", err)
	}

	return aggregates, nil
}
