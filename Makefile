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

# Build probe for remote deployment
build-probe:
	@echo "Building probe agent..."
	go build -o bin/probe ./cmd/probe
	@echo "Probe built successfully: bin/probe"
	
# Create probe deployment package
probe-package: build-probe
	@echo "Creating probe deployment package..."
	@mkdir -p deploy/probe-package
	@cp bin/probe deploy/probe-package/
	@echo "#!/bin/bash" > deploy/probe-package/run-probe.sh
	@echo "# WireScope Probe Agent" >> deploy/probe-package/run-probe.sh
	@echo "# Usage: ./run-probe.sh <server-ip> [interval-seconds]" >> deploy/probe-package/run-probe.sh
	@echo "" >> deploy/probe-package/run-probe.sh
	@echo "SERVER_IP=\$$1" >> deploy/probe-package/run-probe.sh
	@echo "INTERVAL=\$${2:-10}" >> deploy/probe-package/run-probe.sh
	@echo "" >> deploy/probe-package/run-probe.sh
	@echo "if [ -z \"\$$SERVER_IP\" ]; then" >> deploy/probe-package/run-probe.sh
	@echo "  echo \"Usage: ./run-probe.sh <server-ip> [interval-seconds]\"" >> deploy/probe-package/run-probe.sh
	@echo "  echo \"Example: ./run-probe.sh 192.168.1.100 10\"" >> deploy/probe-package/run-probe.sh
	@echo "  exit 1" >> deploy/probe-package/run-probe.sh
	@echo "fi" >> deploy/probe-package/run-probe.sh
	@echo "" >> deploy/probe-package/run-probe.sh
	@echo "export INGEST_URL=\"http://\$$SERVER_IP:8081/ingest\"" >> deploy/probe-package/run-probe.sh
	@echo "export PROBE_INTERVAL=\"\$$INTERVAL\"" >> deploy/probe-package/run-probe.sh
	@echo "export PROBE_ID=\"probe-\$$HOSTNAME-\$$RANDOM\"" >> deploy/probe-package/run-probe.sh
	@echo "" >> deploy/probe-package/run-probe.sh
	@echo "echo \"Starting WireScope Probe Agent...\"" >> deploy/probe-package/run-probe.sh
	@echo "echo \"Server: \$$INGEST_URL\"" >> deploy/probe-package/run-probe.sh
	@echo "echo \"Interval: \$$INTERVAL seconds\"" >> deploy/probe-package/run-probe.sh
	@echo "echo \"Probe ID: \$$PROBE_ID\"" >> deploy/probe-package/run-probe.sh
	@echo "echo \"\"" >> deploy/probe-package/run-probe.sh
	@echo "echo \"Press Ctrl+C to stop\"" >> deploy/probe-package/run-probe.sh
	@echo "echo \"\"" >> deploy/probe-package/run-probe.sh
	@echo "" >> deploy/probe-package/run-probe.sh
	@echo "./probe" >> deploy/probe-package/run-probe.sh
	@chmod +x deploy/probe-package/run-probe.sh
	@echo "INGEST_URL=http://YOUR_SERVER_IP:8081/ingest" > deploy/probe-package/.env.example
	@echo "PROBE_INTERVAL=10" >> deploy/probe-package/.env.example
	@echo "PROBE_ID=probe-remote-1" >> deploy/probe-package/.env.example
	@echo ""
	@echo "✓ Probe package created: deploy/probe-package/"
	@echo ""
	@echo "To deploy:"
	@echo "  1. Transfer deploy/probe-package/ to remote machine"
	@echo "  2. Run: ./run-probe.sh <your-mac-ip> [interval]"
	@echo "  3. Example: ./run-probe.sh 192.168.1.100 10"
	@echo "Building cleanup utility..."
	go build -o bin/cleanup ./cmd/cleanup
	@echo "Building AI agent service..."
	go build -o bin/ai-agent ./cmd/ai-agent
	@echo "Building AI agent CLI..."
	go build -o bin/telemetry-ai ./cmd/telemetry-ai

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

# Database maintenance targets
db-cleanup:
	@echo "Running database cleanup (dry-run)..."
	./bin/cleanup -dry-run

db-cleanup-force:
	@echo "Running database cleanup (actual deletion)..."
	./bin/cleanup

db-health:
	@echo "Running database health check..."
	./bin/cleanup -health-check

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

run-ai-agent:
	./bin/ai-agent

run-ai-cli:
	./bin/telemetry-ai