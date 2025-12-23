.PHONY: help up down build test clean migrate migrate-up migrate-down logs

# Default target
help:
	@echo "Network QoE Telemetry Platform"
	@echo ""
	@echo "Available targets:"
	@echo "  validate      - Validate the setup configuration"
	@echo "  test-endpoints- Test all service endpoints"
	@echo "  up            - Start all services with Docker Compose"
	@echo "  down        - Stop all services"
	@echo "  build       - Build all Go binaries"
	@echo "  test        - Run all tests"
	@echo "  clean       - Clean build artifacts and Docker volumes"
	@echo "  migrate     - Run database migrations"
	@echo "  migrate-up  - Apply all pending migrations"
	@echo "  migrate-down- Rollback last migration"
	@echo "  migrate-status - Check migration status"
	@echo "  logs        - Show logs from all services"
	@echo "  dev         - Start development environment"

# Docker Compose operations
up:
	docker-compose up -d
	@echo "Services starting... Use 'make logs' to monitor startup"
	@echo "Grafana: http://localhost:3000 (admin/admin)"
	@echo "Prometheus: http://localhost:9090"
	@echo "Jaeger: http://localhost:16686"
	@echo "Test Target: http://localhost:8080"

down:
	docker-compose down

logs:
	docker-compose logs -f

dev: up
	@echo "Development environment ready!"
	@echo "Run 'make build' to compile binaries"

# Build targets
build:
	@echo "Building probe agent..."
	go build -o bin/probe ./cmd/probe
	@echo "Building ingest API..."
	go build -o bin/ingest ./cmd/ingest
	@echo "Building aggregator..."
	go build -o bin/aggregator ./cmd/aggregator
	@echo "Building diagnoser..."
	go build -o bin/diagnoser ./cmd/diagnoser

# Test targets
test:
	go test -v ./...

test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./...

# Database migration targets
migrate: migrate-up

migrate-up:
	@echo "Applying database migrations..."
	go run ./cmd/migrate -command=up

migrate-down:
	@echo "Rolling back last migration..."
	go run ./cmd/migrate -command=down

migrate-status:
	@echo "Checking migration status..."
	go run ./cmd/migrate -command=status

migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir migrations $$name

# Cleanup targets
clean:
	rm -rf bin/
	docker-compose down -v
	docker system prune -f

# Development helpers
validate:
	@echo "Running setup validation..."
	@./scripts/validate-setup.sh

test-endpoints:
	@echo "Testing all service endpoints..."
	@./scripts/test-endpoints.sh

fmt:
	go fmt ./...

lint:
	golangci-lint run

deps:
	go mod tidy
	go mod download

# Service-specific targets
run-probe:
	./bin/probe

run-ingest:
	./bin/ingest

run-aggregator:
	./bin/aggregator

run-diagnoser:
	./bin/diagnoser