module github.com/network-qoe-telemetry-platform

go 1.21

require (
	github.com/golang-migrate/migrate/v4 v4.16.2
	github.com/lib/pq v1.10.9
	github.com/nats-io/nats.go v1.31.0
	github.com/prometheus/client_golang v1.17.0
	go.opentelemetry.io/otel v1.21.0
	go.opentelemetry.io/otel/exporters/jaeger v1.17.0
	go.opentelemetry.io/otel/sdk v1.21.0
	go.opentelemetry.io/otel/trace v1.21.0
)