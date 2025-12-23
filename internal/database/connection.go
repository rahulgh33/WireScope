package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

// ConnectionConfig holds database connection pool configuration
type ConnectionConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DefaultConnectionConfig returns a connection config with sensible defaults
func DefaultConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		Host:            "localhost",
		Port:            5432,
		User:            "telemetry",
		Password:        "telemetry",
		Database:        "telemetry",
		SSLMode:         "disable",
		MaxOpenConns:    25,  // Maximum number of open connections
		MaxIdleConns:    5,   // Maximum number of idle connections
		ConnMaxLifetime: time.Hour,   // Maximum lifetime of a connection
		ConnMaxIdleTime: time.Minute * 5, // Maximum idle time before closing
	}
}

// Connection wraps a database connection with additional utilities
type Connection struct {
	db     *sql.DB
	config *ConnectionConfig
}

// NewConnection creates a new database connection with connection pooling
func NewConnection(config *ConnectionConfig) (*Connection, error) {
	if config == nil {
		config = DefaultConnectionConfig()
	}

	// Build connection string
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.Database, config.SSLMode)

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	conn := &Connection{
		db:     db,
		config: config,
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return conn, nil
}

// DB returns the underlying sql.DB instance
func (c *Connection) DB() *sql.DB {
	return c.db
}

// Ping tests the database connection
func (c *Connection) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

// Close closes the database connection
func (c *Connection) Close() error {
	return c.db.Close()
}

// Stats returns database connection pool statistics
func (c *Connection) Stats() sql.DBStats {
	return c.db.Stats()
}

// BeginTx starts a new transaction with the given options
func (c *Connection) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return c.db.BeginTx(ctx, opts)
}

// ExecContext executes a query without returning any rows
func (c *Connection) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return c.db.ExecContext(ctx, query, args...)
}

// QueryContext executes a query that returns rows
func (c *Connection) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return c.db.QueryContext(ctx, query, args...)
}

// QueryRowContext executes a query that is expected to return at most one row
func (c *Connection) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return c.db.QueryRowContext(ctx, query, args...)
}

// IsConnectionError checks if an error is a connection-related error
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	// Check for PostgreSQL connection errors
	if pqErr, ok := err.(*pq.Error); ok {
		// Connection failure error codes
		switch pqErr.Code {
		case "08000", "08003", "08006", "08001", "08004":
			return true
		}
	}

	// Check for common connection error patterns
	switch err {
	case sql.ErrConnDone:
		return true
	default:
		return false
	}
}

// IsRetryableError checks if an error is retryable (transient)
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Connection errors are retryable
	if IsConnectionError(err) {
		return true
	}

	// Check for PostgreSQL retryable errors
	if pqErr, ok := err.(*pq.Error); ok {
		switch pqErr.Code {
		case "40001": // serialization_failure
			return true
		case "40P01": // deadlock_detected
			return true
		case "53000": // insufficient_resources
			return true
		case "53100": // disk_full
			return true
		case "53200": // out_of_memory
			return true
		case "53300": // too_many_connections
			return true
		}
	}

	return false
}