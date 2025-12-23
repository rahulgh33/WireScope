package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/network-qoe-telemetry-platform/internal/models"
	"github.com/network-qoe-telemetry-platform/internal/probe"
	"github.com/network-qoe-telemetry-platform/internal/tracing"
)

var (
	target         = flag.String("target", "https://example.com", "Target URL to measure")
	throughputURL  = flag.String("throughput-url", "", "URL for throughput testing (defaults to target/fixed/1mb.bin)")
	interval       = flag.Duration("interval", 60*time.Second, "Measurement interval")
	ingestURL      = flag.String("ingest-url", "http://localhost:8080/events", "Ingest API URL")
	apiToken       = flag.String("api-token", "", "API token for authentication")
	once           = flag.Bool("once", false, "Run once and exit")
	interfaceType  = flag.String("interface", "ethernet", "Network interface type (wifi, ethernet, cellular)")
	vpnEnabled     = flag.Bool("vpn", false, "Whether VPN is enabled")
	userLabel      = flag.String("label", "", "Optional user-defined label")
	schemaVersion  = flag.String("schema", "1.0", "Event schema version")
	queueSize      = flag.Int("queue-size", 100, "Maximum number of events to buffer")
	maxBackoff     = flag.Duration("max-backoff", 60*time.Second, "Maximum backoff duration for retries")
	otlpEndpoint   = flag.String("otlp-endpoint", "localhost:4318", "OpenTelemetry OTLP HTTP endpoint")
	tracingEnabled = flag.Bool("tracing-enabled", true, "Enable distributed tracing")
)

// EventQueue implements a bounded queue with exponential backoff
// Requirement: 8.2 - Backpressure with bounded local queue
type EventQueue struct {
	queue      chan *models.TelemetryEvent
	droppedCnt int
}

func NewEventQueue(size int) *EventQueue {
	return &EventQueue{
		queue: make(chan *models.TelemetryEvent, size),
	}
}

func (q *EventQueue) Enqueue(event *models.TelemetryEvent) bool {
	select {
	case q.queue <- event:
		return true
	default:
		q.droppedCnt++
		log.Printf("Event queue full, dropping event %s (total dropped: %d)", event.EventID, q.droppedCnt)
		return false
	}
}

func (q *EventQueue) Dequeue() *models.TelemetryEvent {
	return <-q.queue
}

func (q *EventQueue) DroppedCount() int {
	return q.droppedCnt
}

func main() {
	flag.Parse()

	// Initialize OpenTelemetry tracing
	// Requirement: 6.4 - Distributed tracing setup for probe
	tracingConfig := tracing.DefaultConfig("probe")
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

	// Get or create stable client ID
	clientID, err := probe.GetOrCreateClientID()
	if err != nil {
		log.Fatalf("Failed to get client ID: %v", err)
	}

	log.Printf("Network QoE Probe Agent")
	log.Printf("Client ID: %s", clientID)
	log.Printf("Target: %s", *target)
	log.Printf("Interval: %v", *interval)
	log.Printf("Queue size: %d", *queueSize)

	// Set throughput URL if not specified
	throughputEndpoint := *throughputURL
	if throughputEndpoint == "" {
		throughputEndpoint = *target + "/fixed/1mb.bin"
	}

	// Create event queue
	eventQueue := NewEventQueue(*queueSize)

	// Start worker goroutine to send events with exponential backoff
	if *ingestURL != "" && *apiToken != "" {
		go eventSender(eventQueue, *ingestURL, *apiToken, *maxBackoff)
	}

	// Run measurement loop
	for {
		event := performMeasurement(clientID, *target, throughputEndpoint)

		// Enqueue event for sending
		if *ingestURL != "" && *apiToken != "" {
			eventQueue.Enqueue(event)
		}

		if *once {
			// Wait a bit for the event to be sent before exiting
			time.Sleep(2 * time.Second)
			log.Printf("Total dropped events: %d", eventQueue.DroppedCount())
			break
		}

		time.Sleep(*interval)
	}
}

func performMeasurement(clientID, targetURL, throughputURL string) *models.TelemetryEvent {
	// Create trace span for measurement
	// Requirement: 6.4 - Probe measurement tracing
	tracer := tracing.GetTracer("probe")
	ctx, span := tracer.Start(context.Background(), "probe.measure")
	defer span.End()

	// Add measurement attributes to span
	// Requirement: 6.5 - Span attributes for network operations
	span.SetAttributes(
		attribute.String("client.id", clientID),
		attribute.String("target.url", targetURL),
		attribute.String("throughput.url", throughputURL),
	)

	log.Printf("Performing measurement for %s", targetURL)

	// Perform the measurement
	tracing.AddSpanEvent(ctx, "measurement.start")
	measurement, err := probe.MeasureTargetWithThroughput(targetURL, throughputURL)

	// Create telemetry event
	event := &models.TelemetryEvent{
		EventID:       uuid.New().String(),
		ClientID:      clientID,
		TimestampMs:   time.Now().UnixMilli(),
		SchemaVersion: *schemaVersion,
		Target:        targetURL,
		NetworkContext: models.NetworkContext{
			InterfaceType: *interfaceType,
			VPNEnabled:    *vpnEnabled,
		},
	}

	// Set user label if provided
	if *userLabel != "" {
		event.NetworkContext.UserLabel = userLabel
	}

	if err != nil {
		// Measurement had an error
		tracing.RecordError(ctx, err)
		if measurement != nil && measurement.ErrorStage != nil {
			event.ErrorStage = measurement.ErrorStage
			tracing.AddSpanAttributes(ctx, attribute.String("error.stage", *measurement.ErrorStage))
			// Still include partial timing data
			event.Timings = models.TimingMeasurements{
				DNSMs:      measurement.DNSMs,
				TCPMs:      measurement.TCPMs,
				TLSMs:      measurement.TLSMs,
				HTTPTTFBMs: measurement.HTTPTTFBMs,
			}
			event.ThroughputKbps = measurement.ThroughputKbps
		} else {
			// Complete failure - set generic error
			errorStage := "unknown"
			event.ErrorStage = &errorStage
			tracing.AddSpanAttributes(ctx, attribute.String("error.stage", "unknown"))
		}
		log.Printf("Measurement error: %v", err)
	} else {
		// Successful measurement
		tracing.AddSpanEvent(ctx, "measurement.success")
		event.Timings = models.TimingMeasurements{
			DNSMs:      measurement.DNSMs,
			TCPMs:      measurement.TCPMs,
			TLSMs:      measurement.TLSMs,
			HTTPTTFBMs: measurement.HTTPTTFBMs,
		}
		event.ThroughputKbps = measurement.ThroughputKbps

		// Add timing attributes to span
		// Requirement: 6.5 - Network operation details in spans
		span.SetAttributes(
			attribute.Float64("timing.dns_ms", measurement.DNSMs),
			attribute.Float64("timing.tcp_ms", measurement.TCPMs),
			attribute.Float64("timing.tls_ms", measurement.TLSMs),
			attribute.Float64("timing.ttfb_ms", measurement.HTTPTTFBMs),
			attribute.Float64("throughput.kbps", measurement.ThroughputKbps),
		)
	}

	return event
}

func printMeasurement(event *models.TelemetryEvent) {
	fmt.Println("─────────────────────────────────────────────")
	fmt.Printf("Measurement at %s\n", time.UnixMilli(event.TimestampMs).Format(time.RFC3339))
	fmt.Printf("Event ID: %s\n", event.EventID)
	fmt.Printf("Target: %s\n", event.Target)

	if event.ErrorStage != nil {
		fmt.Printf("❌ Error Stage: %s\n", *event.ErrorStage)
	} else {
		fmt.Println("✓ Success")
	}

	fmt.Printf("DNS:   %.2f ms\n", event.Timings.DNSMs)
	fmt.Printf("TCP:   %.2f ms\n", event.Timings.TCPMs)
	fmt.Printf("TLS:   %.2f ms\n", event.Timings.TLSMs)
	fmt.Printf("TTFB:  %.2f ms\n", event.Timings.HTTPTTFBMs)
	fmt.Printf("Total: %.2f ms\n", event.Timings.DNSMs+event.Timings.TCPMs+event.Timings.TLSMs+event.Timings.HTTPTTFBMs)

	if event.ThroughputKbps > 0 {
		fmt.Printf("Throughput: %.2f kbps (%.2f Mbps)\n", event.ThroughputKbps, event.ThroughputKbps/1000.0)
	}

	fmt.Println("─────────────────────────────────────────────")
}

// eventSender processes events from the queue with exponential backoff
// Requirement: 8.2 - Exponential backoff for retry
func eventSender(queue *EventQueue, ingestURL, apiToken string, maxBackoff time.Duration) {
	for {
		event := queue.Dequeue()

		backoff := 1 * time.Second
		for {
			err := sendEventToIngest(event, ingestURL, apiToken)
			if err == nil {
				log.Printf("Successfully sent event %s to ingest API", event.EventID)
				break // Success, move to next event
			}

			log.Printf("Failed to send event %s: %v, retrying in %v", event.EventID, err, backoff)
			time.Sleep(backoff)

			// Exponential backoff with max limit
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

func sendEventToIngest(event *models.TelemetryEvent, ingestURL, apiToken string) error {
	// Serialize event to JSON
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", ingestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiToken)

	// Send request
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("ingest API returned status %d", resp.StatusCode)
	}

	return nil
}
