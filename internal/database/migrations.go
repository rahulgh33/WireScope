package database

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Migration represents a database migration
type Migration struct {
	Version int
	Name    string
	UpSQL   string
	DownSQL string
}

// MigrationManager handles database migrations
type MigrationManager struct {
	conn *Connection
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(conn *Connection) *MigrationManager {
	return &MigrationManager{
		conn: conn,
	}
}

// createMigrationsTable creates the migrations tracking table if it doesn't exist
func (m *MigrationManager) createMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP DEFAULT NOW()
		)`

	_, err := m.conn.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	return nil
}

// getAppliedMigrations returns a map of applied migration versions
func (m *MigrationManager) getAppliedMigrations(ctx context.Context) (map[int]bool, error) {
	query := "SELECT version FROM schema_migrations ORDER BY version"
	rows, err := m.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("failed to scan migration version: %w", err)
		}
		applied[version] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating migration rows: %w", err)
	}

	return applied, nil
}

// LoadMigrationsFromFS loads migrations from an embedded filesystem
func LoadMigrationsFromFS(migrationFS fs.FS, dir string) ([]*Migration, error) {
	var migrations []*Migration

	err := fs.WalkDir(migrationFS, dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Only process .up.sql files
		if !strings.HasSuffix(path, ".up.sql") {
			return nil
		}

		// Extract version and name from filename
		filename := filepath.Base(path)
		parts := strings.SplitN(filename, "_", 2)
		if len(parts) < 2 {
			return fmt.Errorf("invalid migration filename format: %s", filename)
		}

		version, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("invalid migration version in filename %s: %w", filename, err)
		}

		name := strings.TrimSuffix(parts[1], ".up.sql")

		// Read up migration
		upContent, err := fs.ReadFile(migrationFS, path)
		if err != nil {
			return fmt.Errorf("failed to read up migration %s: %w", path, err)
		}

		// Read corresponding down migration
		downPath := strings.Replace(path, ".up.sql", ".down.sql", 1)
		downContent, err := fs.ReadFile(migrationFS, downPath)
		if err != nil {
			return fmt.Errorf("failed to read down migration %s: %w", downPath, err)
		}

		migrations = append(migrations, &Migration{
			Version: version,
			Name:    name,
			UpSQL:   string(upContent),
			DownSQL: string(downContent),
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk migration directory: %w", err)
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// Up applies all pending migrations
func (m *MigrationManager) Up(ctx context.Context, migrations []*Migration) error {
	if err := m.createMigrationsTable(ctx); err != nil {
		return err
	}

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if applied[migration.Version] {
			continue // Skip already applied migrations
		}

		if err := m.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %d (%s): %w", 
				migration.Version, migration.Name, err)
		}

		fmt.Printf("Applied migration %d: %s\n", migration.Version, migration.Name)
	}

	return nil
}

// Down rolls back the last applied migration
func (m *MigrationManager) Down(ctx context.Context, migrations []*Migration) error {
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	// Find the highest applied migration
	var lastMigration *Migration
	for i := len(migrations) - 1; i >= 0; i-- {
		if applied[migrations[i].Version] {
			lastMigration = migrations[i]
			break
		}
	}

	if lastMigration == nil {
		return fmt.Errorf("no migrations to roll back")
	}

	if err := m.rollbackMigration(ctx, lastMigration); err != nil {
		return fmt.Errorf("failed to rollback migration %d (%s): %w", 
			lastMigration.Version, lastMigration.Name, err)
	}

	fmt.Printf("Rolled back migration %d: %s\n", lastMigration.Version, lastMigration.Name)
	return nil
}

// applyMigration applies a single migration
func (m *MigrationManager) applyMigration(ctx context.Context, migration *Migration) error {
	tx, err := m.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute the migration SQL
	if _, err := tx.ExecContext(ctx, migration.UpSQL); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record the migration as applied
	insertQuery := `
		INSERT INTO schema_migrations (version, name, applied_at) 
		VALUES ($1, $2, $3)`
	
	if _, err := tx.ExecContext(ctx, insertQuery, migration.Version, migration.Name, time.Now()); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}

	return nil
}

// rollbackMigration rolls back a single migration
func (m *MigrationManager) rollbackMigration(ctx context.Context, migration *Migration) error {
	tx, err := m.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute the rollback SQL
	if _, err := tx.ExecContext(ctx, migration.DownSQL); err != nil {
		return fmt.Errorf("failed to execute rollback SQL: %w", err)
	}

	// Remove the migration record
	deleteQuery := "DELETE FROM schema_migrations WHERE version = $1"
	if _, err := tx.ExecContext(ctx, deleteQuery, migration.Version); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback transaction: %w", err)
	}

	return nil
}

// Status returns the current migration status
func (m *MigrationManager) Status(ctx context.Context, migrations []*Migration) error {
	if err := m.createMigrationsTable(ctx); err != nil {
		return err
	}

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	fmt.Println("Migration Status:")
	fmt.Println("Version | Name | Status")
	fmt.Println("--------|------|-------")

	for _, migration := range migrations {
		status := "Pending"
		if applied[migration.Version] {
			status = "Applied"
		}
		fmt.Printf("%7d | %s | %s\n", migration.Version, migration.Name, status)
	}

	return nil
}