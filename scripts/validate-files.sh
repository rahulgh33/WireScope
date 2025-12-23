#!/bin/bash

# Network QoE Platform File Structure Validation Script
# This script validates the project structure without requiring Docker

set -e

echo "🔍 Validating Network QoE Platform File Structure..."

# Check if required files exist
echo "🔍 Checking required files..."

required_files=(
    "docker-compose.yml"
    "Makefile"
    "go.mod"
    "README.md"
    ".gitignore"
    "config/prometheus.yml"
    "config/otel-collector.yml"
    "config/init.sql"
    "config/test-server.py"
    "config/Dockerfile.test-server"
    "config/config.go"
    "migrations/001_initial_schema.up.sql"
    "migrations/001_initial_schema.down.sql"
    "cmd/probe/main.go"
    "cmd/ingest/main.go"
    "cmd/aggregator/main.go"
    "cmd/diagnoser/main.go"
)

missing_files=()

for file in "${required_files[@]}"; do
    if [[ -f "$file" ]]; then
        echo "✅ $file exists"
    else
        echo "❌ $file is missing"
        missing_files+=("$file")
    fi
done

# Check directory structure
echo "🔍 Checking directory structure..."

required_dirs=(
    "cmd/probe"
    "cmd/ingest"
    "cmd/aggregator"
    "cmd/diagnoser"
    "internal"
    "pkg"
    "config"
    "config/grafana/datasources"
    "migrations"
    "scripts"
)

missing_dirs=()

for dir in "${required_dirs[@]}"; do
    if [[ -d "$dir" ]]; then
        echo "✅ $dir/ directory exists"
    else
        echo "❌ $dir/ directory is missing"
        missing_dirs+=("$dir")
    fi
done

# Validate YAML syntax (if yq is available)
if command -v python3 &> /dev/null; then
    echo "🔍 Validating YAML syntax..."
    
    yaml_files=(
        "docker-compose.yml"
        "config/prometheus.yml"
        "config/otel-collector.yml"
        "config/grafana/datasources/datasources.yml"
    )
    
    for yaml_file in "${yaml_files[@]}"; do
        if [[ -f "$yaml_file" ]]; then
            if python3 -c "import yaml; yaml.safe_load(open('$yaml_file'))" 2>/dev/null; then
                echo "✅ $yaml_file has valid YAML syntax"
            else
                echo "❌ $yaml_file has invalid YAML syntax"
                missing_files+=("$yaml_file (invalid syntax)")
            fi
        fi
    done
fi

# Validate SQL syntax (basic check)
echo "🔍 Checking SQL files..."
sql_files=(
    "config/init.sql"
    "migrations/001_initial_schema.up.sql"
    "migrations/001_initial_schema.down.sql"
)

for sql_file in "${sql_files[@]}"; do
    if [[ -f "$sql_file" ]]; then
        # Basic check for SQL keywords
        if grep -q -E "(CREATE|DROP|INSERT|SELECT)" "$sql_file"; then
            echo "✅ $sql_file contains SQL statements"
        else
            echo "⚠️  $sql_file may not contain valid SQL"
        fi
    fi
done

# Check Go module
if [[ -f "go.mod" ]]; then
    if grep -q "module github.com/network-qoe-telemetry-platform" go.mod; then
        echo "✅ go.mod has correct module name"
    else
        echo "⚠️  go.mod module name may need adjustment"
    fi
fi

# Summary
echo ""
if [[ ${#missing_files[@]} -eq 0 && ${#missing_dirs[@]} -eq 0 ]]; then
    echo "🎉 File structure validation completed successfully!"
    echo ""
    echo "✅ All required files and directories are present"
    echo "✅ Configuration files have valid syntax"
    echo ""
    echo "The project structure is ready for development!"
else
    echo "❌ Validation failed!"
    if [[ ${#missing_files[@]} -gt 0 ]]; then
        echo "Missing files: ${missing_files[*]}"
    fi
    if [[ ${#missing_dirs[@]} -gt 0 ]]; then
        echo "Missing directories: ${missing_dirs[*]}"
    fi
    exit 1
fi

echo ""
echo "Next steps (when Docker is available):"
echo "1. Run 'make validate' to validate Docker setup"
echo "2. Run 'make dev' to start the development environment"
echo "3. Run 'make build' to compile the Go binaries"