package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/network-qoe-telemetry-platform/internal/models"
	"github.com/network-qoe-telemetry-platform/internal/queue"
)

var (
	port      = flag.String("port", "8080", "HTTP server port")
	natsURL   = flag.String("nats-url", "nats://localhost:4222", "NATS server URL")
	apiTokens = flag.String("api-tokens", "", "Comma-separated list of valid API tokens")
)

// IngestAPI handles HTTP requests for telemetry event ingestion
type IngestAPI struct {
	processor   models.EventProcessor
	validTokens map[string]bool
}

// NewIngestAPI creates a new ingest API server
func NewIngestAPI(processor models.EventProcessor, tokens []string) *IngestAPI {
	validTokens := make(map[string]bool)
	for _, token := range tokens {
		if token != "" {
			validTokens[token] = true
		}
	}

	return &IngestAPI{
		processor:   processor,
		validTokens: validTokens,
	}
}

// authMiddleware validates API tokens
//
// Requirement: 10.1 - HTTP server with basic authentication middleware
func (api *IngestAPI) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Check for Bearer token format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid Authorization header format. Expected: Bearer <token>", http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// Validate token
		if len(api.validTokens) > 0 && !api.validTokens[token] {
			http.Error(w, "Invalid API token", http.StatusUnauthorized)
			return
		}

		// Token valid, proceed to next handler
		next(w, r)
	}
}

// handleIngestEvent handles POST /events for telemetry event ingestion
//
// Requirements: 10.2, 10.3, 10.4, 10.5
func (api *IngestAPI) handleIngestEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse JSON request body
	var event models.TelemetryEvent
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // Strict parsing for now

	if err := decoder.Decode(&event); err != nil {
		log.Printf("Failed to decode event: %v", err)
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Inject recv_ts_ms timestamp for clock skew debugging
	// Requirement: 10.4 - recv_ts_ms timestamp injection
	recvTs := time.Now().UnixMilli()
	event.RecvTimestampMs = &recvTs

	// Validate schema version
	// Requirement: 10.2 - Request validation with schema version checking
	if event.SchemaVersion == "" {
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
	}

	// Validate event structure
	if err := event.Validate(); err != nil {
		log.Printf("Event validation failed: %v", err)
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusBadRequest)
		return
	}

	// Publish event to NATS JetStream
	// Requirement: 10.3 - Event publishing to NATS JetStream
	if err := api.processor.PublishEvent(&event); err != nil {
		log.Printf("Failed to publish event %s: %v", event.EventID, err)
		http.Error(w, "Failed to publish event", http.StatusInternalServerError)
		return
	}

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

	// Create ingest API
	api := NewIngestAPI(processor, tokens)

	// Set up HTTP routes
	http.HandleFunc("/health", api.handleHealth)
	http.HandleFunc("/events", api.authMiddleware(api.handleIngestEvent))

	// Start HTTP server
	addr := ":" + *port
	log.Printf("Ingest API listening on %s", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
