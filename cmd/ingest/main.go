package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"

	"github.com/network-qoe-telemetry-platform/internal/metrics"
	"github.com/network-qoe-telemetry-platform/internal/models"
	"github.com/network-qoe-telemetry-platform/internal/queue"
	"github.com/network-qoe-telemetry-platform/internal/tracing"
)

var (
	port           = flag.String("port", "8080", "HTTP server port")
	natsURL        = flag.String("nats-url", "nats://localhost:4222", "NATS server URL")
	apiTokens      = flag.String("api-tokens", "", "Comma-separated list of valid API tokens")
	rateLimit      = flag.Int("rate-limit", 100, "Maximum requests per client per second")
	rateLimitBurst = flag.Int("rate-limit-burst", 20, "Maximum burst size for rate limiting")
	otlpEndpoint   = flag.String("otlp-endpoint", "localhost:4318", "OpenTelemetry OTLP HTTP endpoint")
	tracingEnabled = flag.Bool("tracing-enabled", true, "Enable distributed tracing")
)

// Prometheus metrics
// Requirements: 6.1, 6.2, 6.3 - Comprehensive ingest API metrics
var (
	ingestRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ingest_requests_total",
			Help: "Total number of ingest API requests",
		},
		[]string{"status"}, // success, validation_error, auth_error, rate_limited, publish_error
	)

	ingestRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ingest_request_duration_seconds",
			Help:    "Duration of ingest API requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"status"},
	)

	ingestPayloadSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ingest_payload_size_bytes",
			Help:    "Size of ingest request payloads in bytes",
			Buckets: []float64{100, 500, 1000, 5000, 10000, 50000},
		},
		[]string{"status"},
	)

	ingestAuthFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ingest_auth_failures_total",
			Help: "Total number of authentication failures",
		},
		[]string{"reason"}, // missing_token, invalid_token
	)

	ingestRateLimitHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ingest_rate_limit_hits_total",
			Help: "Total number of rate limit hits per client",
		},
		[]string{"client_id_hash"}, // hashed for cardinality management
	)

	ingestActiveConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "ingest_active_connections",
			Help: "Number of currently active HTTP connections",
		},
	)

	ingestEventsPerClient = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ingest_events_per_client_total",
			Help: "Total events ingested per client (hashed)",
		},
		[]string{"client_id_hash"},
	)
)

func init() {
	prometheus.MustRegister(ingestRequestsTotal)
	prometheus.MustRegister(ingestRequestDuration)
	prometheus.MustRegister(ingestPayloadSize)
	prometheus.MustRegister(ingestAuthFailures)
	prometheus.MustRegister(ingestRateLimitHits)
	prometheus.MustRegister(ingestActiveConnections)
	prometheus.MustRegister(ingestEventsPerClient)
}

// TokenBucket implements a simple token bucket rate limiter
// Requirement: 8.3 - Rate limiting per client_id
type TokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

func NewTokenBucket(rate, burst int) *TokenBucket {
	return &TokenBucket{
		tokens:     float64(burst),
		maxTokens:  float64(burst),
		refillRate: float64(rate),
		lastRefill: time.Now(),
	}
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()

	// Refill tokens based on elapsed time
	tb.tokens = tb.tokens + (elapsed * tb.refillRate)
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
	tb.lastRefill = now

	// Check if we have tokens available
	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		return true
	}

	return false
}

// IngestAPI handles HTTP requests for telemetry event ingestion
type IngestAPI struct {
	processor    models.EventProcessor
	validTokens  map[string]bool
	rateLimiters map[string]*TokenBucket
	limiterMu    sync.RWMutex
	rateLimit    int
	rateBurst    int
}

// NewIngestAPI creates a new ingest API server
func NewIngestAPI(processor models.EventProcessor, tokens []string, rateLimit, rateBurst int) *IngestAPI {
	validTokens := make(map[string]bool)
	for _, token := range tokens {
		if token != "" {
			validTokens[token] = true
		}
	}

	return &IngestAPI{
		processor:    processor,
		validTokens:  validTokens,
		rateLimiters: make(map[string]*TokenBucket),
		rateLimit:    rateLimit,
		rateBurst:    rateBurst,
	}
}

// getRateLimiter returns or creates a rate limiter for a client
func (api *IngestAPI) getRateLimiter(clientID string) *TokenBucket {
	api.limiterMu.RLock()
	limiter, exists := api.rateLimiters[clientID]
	api.limiterMu.RUnlock()

	if exists {
		return limiter
	}

	// Create new limiter
	api.limiterMu.Lock()
	defer api.limiterMu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists := api.rateLimiters[clientID]; exists {
		return limiter
	}

	limiter = NewTokenBucket(api.rateLimit, api.rateBurst)
	api.rateLimiters[clientID] = limiter
	return limiter
}

// authMiddleware validates API tokens
//
// Requirement: 10.1 - HTTP server with basic authentication middleware
func (api *IngestAPI) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ingestActiveConnections.Inc()
		defer ingestActiveConnections.Dec()

		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			ingestRequestsTotal.WithLabelValues("auth_error").Inc()
			ingestAuthFailures.WithLabelValues("missing_token").Inc()
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Check for Bearer token format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			ingestRequestsTotal.WithLabelValues("auth_error").Inc()
			ingestAuthFailures.WithLabelValues("invalid_format").Inc()
			http.Error(w, "Invalid Authorization header format. Expected: Bearer <token>", http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// Validate token
		if len(api.validTokens) > 0 && !api.validTokens[token] {
			ingestRequestsTotal.WithLabelValues("auth_error").Inc()
			ingestAuthFailures.WithLabelValues("invalid_token").Inc()
			http.Error(w, "Invalid API token", http.StatusUnauthorized)
			return
		}

		// Token valid, proceed to next handler
		next(w, r)
	}
}

// handleIngestEvent handles POST /events for telemetry event ingestion
//
// Requirements: 10.2, 10.3, 10.4, 10.5, 6.4, 6.5
func (api *IngestAPI) handleIngestEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tracer := tracing.GetTracer("ingest-api")
	ctx, span := tracer.Start(ctx, "ingest.handleEvent")
	defer span.End()

	start := time.Now()
	status := "success"
	var payloadSize int64
	var clientIDHash string

	defer func() {
		duration := time.Since(start).Seconds()
		ingestRequestsTotal.WithLabelValues(status).Inc()
		ingestRequestDuration.WithLabelValues(status).Observe(duration)
		ingestPayloadSize.WithLabelValues(status).Observe(float64(payloadSize))

		// Add status to span
		span.SetAttributes(attribute.String("http.status", status))
	}()

	if r.Method != http.MethodPost {
		status = "method_not_allowed"
		tracing.RecordError(ctx, fmt.Errorf("method not allowed: %s", r.Method))
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Track payload size
	payloadSize = r.ContentLength
	span.SetAttributes(attribute.Int64("http.request.body.size", payloadSize))

	// Parse JSON request body
	var event models.TelemetryEvent
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // Strict parsing for now

	if err := decoder.Decode(&event); err != nil {
		status = "validation_error"
		tracing.RecordError(ctx, err)
		log.Printf("Failed to decode event: %v", err)
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Add event attributes to span for debugging
	// Requirement: 6.5 - Span attributes for debugging
	span.SetAttributes(
		attribute.String("event.id", event.EventID),
		attribute.String("event.client_id", event.ClientID),
		attribute.String("event.target", event.Target),
		attribute.String("event.schema_version", event.SchemaVersion),
	)

	// Hash client_id for cardinality management
	clientIDHash = metrics.HashClientID(event.ClientID)

	// Inject recv_ts_ms timestamp for clock skew debugging
	// Requirement: 10.4 - recv_ts_ms timestamp injection
	recvTs := time.Now().UnixMilli()
	event.RecvTimestampMs = &recvTs

	// Validate schema version
	// Requirement: 10.2 - Request validation with schema version checking
	if event.SchemaVersion == "" {
		status = "validation_error"
		tracing.RecordError(ctx, fmt.Errorf("missing schema_version"))
		http.Error(w, "Missing schema_version", http.StatusBadRequest)
		return
	}

	// Forward compatibility check - accept all versions for now
	// In production, you might want to validate supported versions
	supportedVersions := map[string]bool{
		"1.0": true,
		// Add future versions here
	}

	if !supportedVersions[event.SchemaVersion] {
		log.Printf("Warning: Unknown schema version %s, accepting anyway for forward compatibility", event.SchemaVersion)
		tracing.AddSpanEvent(ctx, "schema_version.unknown", attribute.String("version", event.SchemaVersion))
	}

	// Validate event structure
	if err := event.Validate(); err != nil {
		status = "validation_error"
		tracing.RecordError(ctx, err)
		log.Printf("Event validation failed: %v", err)
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}

	tracing.AddSpanEvent(ctx, "event.validated")

	// Rate limiting per client_id
	// Requirement: 8.3 - Rate limiting per client_id in ingest API
	limiter := api.getRateLimiter(event.ClientID)
	if !limiter.Allow() {
		status = "rate_limited"
		ingestRateLimitHits.WithLabelValues(clientIDHash).Inc()
		tracing.AddSpanEvent(ctx, "rate_limit.exceeded", attribute.String("client_id_hash", clientIDHash))
		log.Printf("Rate limit exceeded for client %s", event.ClientID)
		http.Error(w, "Rate limit exceeded. Please slow down your requests.", http.StatusTooManyRequests)
		return
	}

	// Publish event to NATS JetStream
	// Requirement: 10.3 - Event publishing to NATS JetStream
	tracing.AddSpanEvent(ctx, "queue.publish.start")
	// Inject trace context for downstream propagation
	tracing.InjectContextIntoEvent(ctx, &event)
	if err := api.processor.PublishEvent(&event); err != nil {
		status = "publish_error"
		tracing.RecordError(ctx, err)
		log.Printf("Failed to publish event %s: %v", event.EventID, err)
		http.Error(w, "Failed to publish event", http.StatusInternalServerError)
		return
	}
	tracing.AddSpanEvent(ctx, "queue.publish.success")

	// Track successful events per client
	ingestEventsPerClient.WithLabelValues(clientIDHash).Inc()

	log.Printf("Successfully ingested event %s from client %s", event.EventID, event.ClientID)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "accepted",
		"event_id": event.EventID,
	})
}

// handleHealth handles GET /health for health checks
func (api *IngestAPI) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "ingest-api",
	})
}

func main() {
	flag.Parse()

	log.Printf("Starting Network QoE Ingest API")
	log.Printf("Port: %s", *port)
	log.Printf("NATS URL: %s", *natsURL)
	log.Printf("Rate limit: %d req/s per client (burst: %d)", *rateLimit, *rateLimitBurst)

	// Initialize OpenTelemetry tracing
	// Requirement: 6.4 - Distributed tracing setup
	tracingConfig := tracing.DefaultConfig("ingest-api")
	tracingConfig.OTLPEndpoint = *otlpEndpoint
	tracingConfig.Enabled = *tracingEnabled

	shutdownTracing, err := tracing.InitTracer(tracingConfig)
	if err != nil {
		log.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer func() {
		if err := shutdownTracing(context.Background()); err != nil {
			log.Printf("Error shutting down tracing: %v", err)
		}
	}()

	// Parse API tokens
	var tokens []string
	if *apiTokens != "" {
		tokens = strings.Split(*apiTokens, ",")
		log.Printf("Loaded %d API token(s)", len(tokens))
	} else {
		log.Printf("WARNING: No API tokens configured, authentication is disabled")
	}

	// Check for environment variable tokens as well
	if envTokens := os.Getenv("API_TOKENS"); envTokens != "" {
		envTokenList := strings.Split(envTokens, ",")
		tokens = append(tokens, envTokenList...)
		log.Printf("Loaded %d API token(s) from environment", len(envTokenList))
	}

	// Initialize NATS processor
	natsConfig := queue.DefaultNATSConfig()
	natsConfig.URL = *natsURL

	processor, err := queue.NewNATSEventProcessor(natsConfig)
	if err != nil {
		log.Fatalf("Failed to create NATS processor: %v", err)
	}
	defer processor.Close()

	log.Printf("Connected to NATS at %s", *natsURL)

	// Create ingest API with rate limiting
	api := NewIngestAPI(processor, tokens, *rateLimit, *rateLimitBurst)

	// Set up HTTP routes with OpenTelemetry instrumentation
	// Requirement: 6.4 - HTTP request tracing with context propagation
	http.Handle("/health", otelhttp.NewHandler(http.HandlerFunc(api.handleHealth), "health"))
	http.Handle("/events", otelhttp.NewHandler(http.HandlerFunc(api.authMiddleware(api.handleIngestEvent)), "ingest.events"))
	http.Handle("/metrics", promhttp.Handler())

	// Start HTTP server
	server := &http.Server{
		Addr:    ":" + *port,
		Handler: nil,
	}

	log.Printf("Ingest API listening on %s", server.Addr)
	log.Printf("Metrics available at http://localhost:%s/metrics", *port)

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-sigCh
	log.Printf("Shutting down ingest API...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Printf("Ingest API stopped")
}
