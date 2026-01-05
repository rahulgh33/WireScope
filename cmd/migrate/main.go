package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rahulgh33/wirescope/config"
	"github.com/rahulgh33/wirescope/internal/database"
)

//go:embed migrations
var migrationFS embed.FS

func main() {
	var (
		command = flag.String("command", "up", "Migration command: up, down, status")
		timeout = flag.Duration("timeout", 30*time.Second, "Operation timeout")
	)
	flag.Parse()

	// Load configuration
	cfg := config.Load()

	// Create database connection
	dbConfig := &database.ConnectionConfig{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		Database:        cfg.Database.Database,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpenConns:    5,  // Lower for migration tool
		MaxIdleConns:    2,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: time.Minute * 5,
	}

	conn, err := database.NewConnection(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close()

	// Load migrations from embedded filesystem
	migrations, err := database.LoadMigrationsFromFS(migrationFS, "migrations")
	if err != nil {
		log.Fatalf("Failed to load migrations: %v", err)
	}

	if len(migrations) == 0 {
		log.Println("No migrations found")
		return
	}

	// Create migration manager
	manager := database.NewMigrationManager(conn)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Execute command
	switch *command {
	case "up":
		if err := manager.Up(ctx, migrations); err != nil {
			log.Fatalf("Migration up failed: %v", err)
		}
		fmt.Println("All migrations applied successfully")

	case "down":
		if err := manager.Down(ctx, migrations); err != nil {
			log.Fatalf("Migration down failed: %v", err)
		}
		fmt.Println("Migration rolled back successfully")

	case "status":
		if err := manager.Status(ctx, migrations); err != nil {
			log.Fatalf("Migration status failed: %v", err)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", *command)
		fmt.Fprintf(os.Stderr, "Available commands: up, down, status\n")
		os.Exit(1)
	}
}