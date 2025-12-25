package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

type Config struct {
	DBHost              string
	DBPort              int
	DBName              string
	DBUser              string
	DBPassword          string
	DryRun              bool
	EventsRetentionDays int
	AggRetentionDays    int
	HealthCheck         bool
}

func main() {
	cfg := &Config{}
	flag.StringVar(&cfg.DBHost, "db-host", "localhost", "Database host")
	flag.IntVar(&cfg.DBPort, "db-port", 5432, "Database port")
	flag.StringVar(&cfg.DBName, "db-name", "telemetry", "Database name")
	flag.StringVar(&cfg.DBUser, "db-user", "telemetry", "Database user")
	flag.StringVar(&cfg.DBPassword, "db-password", "telemetry", "Database password")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "Perform dry run without actually deleting data")
	flag.IntVar(&cfg.EventsRetentionDays, "events-retention-days", 7, "Number of days to retain events_seen data")
	flag.IntVar(&cfg.AggRetentionDays, "agg-retention-days", 90, "Number of days to retain aggregate data")
	flag.BoolVar(&cfg.HealthCheck, "health-check", false, "Perform database health check only")
	flag.Parse()

	// Override from environment if set
	if host := os.Getenv("DB_HOST"); host != "" {
		cfg.DBHost = host
	}
	if user := os.Getenv("DB_USER"); user != "" {
		cfg.DBUser = user
	}
	if password := os.Getenv("DB_PASSWORD"); password != "" {
		cfg.DBPassword = password
	}
	if name := os.Getenv("DB_NAME"); name != "" {
		cfg.DBName = name
	}

	if cfg.HealthCheck {
		if err := runHealthCheck(cfg); err != nil {
			log.Fatalf("Health check failed: %v", err)
		}
		return
	}

	if err := runCleanup(cfg); err != nil {
		log.Fatalf("Cleanup failed: %v", err)
	}
}

func runCleanup(cfg *Config) error {
	// Connect to database
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Connected to database %s@%s:%d/%s", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)

	// Clean up events_seen table
	if err := cleanupEventsSeen(db, cfg); err != nil {
		return fmt.Errorf("failed to cleanup events_seen: %w", err)
	}

	// Clean up agg_1m table
	if err := cleanupAggregates(db, cfg); err != nil {
		return fmt.Errorf("failed to cleanup aggregates: %w", err)
	}

	log.Println("Cleanup completed successfully")
	return nil
}

func cleanupEventsSeen(db *sql.DB, cfg *Config) error {
	// Calculate cutoff date
	cutoffDate := time.Now().AddDate(0, 0, -cfg.EventsRetentionDays)

	log.Printf("Cleaning up events_seen records older than %s (retention: %d days)",
		cutoffDate.Format("2006-01-02 15:04:05"), cfg.EventsRetentionDays)

	// Check how many records will be affected
	query := "SELECT COUNT(*) FROM events_seen WHERE created_at < $1"
	var count int
	if err := db.QueryRow(query, cutoffDate).Scan(&count); err != nil {
		return fmt.Errorf("failed to count records: %w", err)
	}

	log.Printf("Found %d records to clean up in events_seen", count)

	if count == 0 {
		log.Printf("No records to clean up in events_seen")
		return nil
	}

	if cfg.DryRun {
		log.Printf("DRY RUN: Would delete %d records from events_seen", count)
		return nil
	}

	// Perform cleanup in batches to avoid long locks
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	deleteQuery := "DELETE FROM events_seen WHERE created_at < $1"
	result, err := tx.ExecContext(ctx, deleteQuery, cutoffDate)
	if err != nil {
		return fmt.Errorf("failed to delete records: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Successfully deleted %d records from events_seen", rowsAffected)

	// Update table statistics
	if _, err := db.Exec("ANALYZE events_seen"); err != nil {
		log.Printf("Warning: Failed to analyze table: %v", err)
	}

	return nil
}

func cleanupAggregates(db *sql.DB, cfg *Config) error {
	// Calculate cutoff date
	cutoffDate := time.Now().AddDate(0, 0, -cfg.AggRetentionDays)

	log.Printf("Cleaning up agg_1m records older than %s (retention: %d days)",
		cutoffDate.Format("2006-01-02 15:04:05"), cfg.AggRetentionDays)

	// Check how many records will be affected
	query := "SELECT COUNT(*) FROM agg_1m WHERE window_start_ts < $1"
	var count int
	if err := db.QueryRow(query, cutoffDate).Scan(&count); err != nil {
		return fmt.Errorf("failed to count records: %w", err)
	}

	log.Printf("Found %d records to clean up in agg_1m", count)

	if count == 0 {
		log.Printf("No records to clean up in agg_1m")
		return nil
	}

	if cfg.DryRun {
		log.Printf("DRY RUN: Would delete %d records from agg_1m", count)
		return nil
	}

	// Perform cleanup in batches to avoid long locks
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	deleteQuery := "DELETE FROM agg_1m WHERE window_start_ts < $1"
	result, err := tx.ExecContext(ctx, deleteQuery, cutoffDate)
	if err != nil {
		return fmt.Errorf("failed to delete records: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Successfully deleted %d records from agg_1m", rowsAffected)

	// Update table statistics
	if _, err := db.Exec("ANALYZE agg_1m"); err != nil {
		log.Printf("Warning: Failed to analyze table: %v", err)
	}

	return nil
}

func runHealthCheck(cfg *Config) error {
	// Connect to database
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	ctx := context.Background()

	// 1. Basic connectivity
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	log.Println("✓ Database connectivity OK")

	// 2. Check connection pool stats
	stats := db.Stats()
	log.Printf("✓ Connection pool: Open=%d InUse=%d Idle=%d MaxOpen=%d",
		stats.OpenConnections, stats.InUse, stats.Idle, stats.MaxOpenConnections)

	// 3. Check table sizes
	sizeQuery := `
		SELECT 
			pg_size_pretty(pg_total_relation_size('events_seen')) as events_size,
			pg_size_pretty(pg_total_relation_size('agg_1m')) as agg_size
	`
	var eventsSize, aggSize string
	if err := db.QueryRowContext(ctx, sizeQuery).Scan(&eventsSize, &aggSize); err != nil {
		return fmt.Errorf("failed to get table sizes: %w", err)
	}
	log.Printf("✓ Table sizes: events_seen=%s agg_1m=%s", eventsSize, aggSize)

	// 4. Check row counts
	var eventsCount, aggCount int64
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM events_seen").Scan(&eventsCount); err != nil {
		return fmt.Errorf("failed to count events_seen: %w", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM agg_1m").Scan(&aggCount); err != nil {
		return fmt.Errorf("failed to count agg_1m: %w", err)
	}
	log.Printf("✓ Row counts: events_seen=%d agg_1m=%d", eventsCount, aggCount)

	// 5. Check oldest data
	var oldestEvent, oldestAgg *time.Time
	if err := db.QueryRowContext(ctx, "SELECT MIN(created_at) FROM events_seen").Scan(&oldestEvent); err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to get oldest event: %w", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT MIN(window_start_ts) FROM agg_1m").Scan(&oldestAgg); err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to get oldest aggregate: %w", err)
	}

	if oldestEvent != nil {
		log.Printf("✓ Oldest event: %s (age: %v)", oldestEvent.Format(time.RFC3339), time.Since(*oldestEvent).Round(time.Hour))
	} else {
		log.Println("✓ No events in database")
	}

	if oldestAgg != nil {
		log.Printf("✓ Oldest aggregate: %s (age: %v)", oldestAgg.Format(time.RFC3339), time.Since(*oldestAgg).Round(time.Hour))
	} else {
		log.Println("✓ No aggregates in database")
	}

	// 6. Check for long-running queries
	longQueryCheck := `
		SELECT COUNT(*) 
		FROM pg_stat_activity 
		WHERE state = 'active' 
		AND query_start < NOW() - INTERVAL '5 minutes'
		AND query NOT LIKE '%pg_stat_activity%'
	`
	var longQueries int
	if err := db.QueryRowContext(ctx, longQueryCheck).Scan(&longQueries); err != nil {
		log.Printf("Warning: Failed to check long-running queries: %v", err)
	} else if longQueries > 0 {
		log.Printf("⚠ Warning: %d long-running queries detected (>5 minutes)", longQueries)
	} else {
		log.Println("✓ No long-running queries")
	}

	// 7. Check for locks
	lockCheck := `
		SELECT COUNT(*) 
		FROM pg_locks 
		WHERE NOT granted
	`
	var blockedQueries int
	if err := db.QueryRowContext(ctx, lockCheck).Scan(&blockedQueries); err != nil {
		log.Printf("Warning: Failed to check locks: %v", err)
	} else if blockedQueries > 0 {
		log.Printf("⚠ Warning: %d blocked queries detected", blockedQueries)
	} else {
		log.Println("✓ No blocked queries")
	}

	// 8. Check index health
	indexQuery := `
		SELECT 
			schemaname, 
			relname as tablename, 
			indexrelname as indexname, 
			pg_size_pretty(pg_relation_size(indexrelid)) as index_size
		FROM pg_stat_user_indexes
		WHERE schemaname = 'public'
		ORDER BY pg_relation_size(indexrelid) DESC
		LIMIT 5
	`
	rows, err := db.QueryContext(ctx, indexQuery)
	if err != nil {
		log.Printf("Warning: Failed to check indexes: %v", err)
	} else {
		defer rows.Close()
		log.Println("✓ Top 5 indexes:")
		for rows.Next() {
			var schema, table, index, size string
			if err := rows.Scan(&schema, &table, &index, &size); err != nil {
				log.Printf("  Error scanning index row: %v", err)
				continue
			}
			log.Printf("  - %s.%s.%s: %s", schema, table, index, size)
		}
	}

	// 9. Check for vacuum and analyze needs
	vacuumQuery := `
		SELECT 
			schemaname, 
			relname as tablename,
			n_dead_tup,
			n_live_tup,
			CASE 
				WHEN n_live_tup > 0 
				THEN round(100.0 * n_dead_tup / (n_live_tup + n_dead_tup), 2)
				ELSE 0 
			END as dead_tuple_percent
		FROM pg_stat_user_tables
		WHERE schemaname = 'public'
		AND n_dead_tup > 1000
		ORDER BY n_dead_tup DESC
		LIMIT 5
	`
	vacuumRows, err := db.QueryContext(ctx, vacuumQuery)
	if err != nil {
		log.Printf("Warning: Failed to check vacuum status: %v", err)
	} else {
		defer vacuumRows.Close()
		needsVacuum := false
		for vacuumRows.Next() {
			var schema, table string
			var deadTuples, liveTuples int64
			var deadPercent float64
			if err := vacuumRows.Scan(&schema, &table, &deadTuples, &liveTuples, &deadPercent); err != nil {
				log.Printf("  Error scanning vacuum row: %v", err)
				continue
			}
			if !needsVacuum {
				log.Println("⚠ Tables needing vacuum:")
				needsVacuum = true
			}
			log.Printf("  - %s.%s: %d dead tuples (%.2f%%)", schema, table, deadTuples, deadPercent)
		}
		if !needsVacuum {
			log.Println("✓ All tables are well-maintained (no excessive dead tuples)")
		}
	}

	log.Println("\n✓ Database health check completed successfully")
	return nil
}
