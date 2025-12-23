package tracing

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds OpenTelemetry configuration
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string // e.g., "localhost:4318" for Jaeger OTLP HTTP
	Enabled        bool
}

// DefaultConfig returns sensible defaults for OpenTelemetry
func DefaultConfig(serviceName string) *Config {
	return &Config{
		ServiceName:    serviceName,
		ServiceVersion: "1.0.0",
		Environment:    "development",
		OTLPEndpoint:   "localhost:4318",
		Enabled:        true,
	}
}

// InitTracer initializes the OpenTelemetry tracer with OTLP exporter
//
// Requirement: 6.4 - Distributed tracing with OpenTelemetry
func InitTracer(config *Config) (func(context.Context) error, error) {
	if !config.Enabled {
		log.Printf("Tracing disabled for service: %s", config.ServiceName)
		return func(ctx context.Context) error { return nil }, nil
	}

	// Create OTLP HTTP exporter
	exporter, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpoint(config.OTLPEndpoint),
		otlptracehttp.WithInsecure(), // Use HTTP instead of HTTPS for local development
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			attribute.String("environment", config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider with batch span processor
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // Sample all traces in development
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator for context propagation across services
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	log.Printf("OpenTelemetry tracing initialized for service: %s (endpoint: %s)", config.ServiceName, config.OTLPEndpoint)

	// Return cleanup function
	return func(ctx context.Context) error {
		// Give exporter time to flush pending spans
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return tp.Shutdown(ctx)
	}, nil
}

// GetTracer returns a tracer for the given name
//
// Requirement: 6.4 - Tracer creation for instrumentation
func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// SpanFromContext extracts the current span from context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddSpanAttributes adds attributes to the current span in context
//
// Requirement: 6.5 - Span attributes for debugging
func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

// AddSpanEvent adds an event to the current span in context
func AddSpanEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// RecordError records an error on the current span in context
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() && err != nil {
		span.RecordError(err)
	}
}
