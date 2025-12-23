package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/network-qoe-telemetry-platform/internal/models"
	"github.com/network-qoe-telemetry-platform/internal/probe"
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
)

func main() {
	flag.Parse()

	// Get or create stable client ID
	clientID, err := probe.GetOrCreateClientID()
	if err != nil {
		log.Fatalf("Failed to get client ID: %v", err)
	}

	log.Printf("Network QoE Probe Agent")
	log.Printf("Client ID: %s", clientID)
	log.Printf("Target: %s", *target)
	log.Printf("Interval: %v", *interval)

	// Set throughput URL if not specified
	throughputEndpoint := *throughputURL
	if throughputEndpoint == "" {
		throughputEndpoint = *target + "/fixed/1mb.bin"
	}

	// Run measurement loop
	for {
		event := performMeasurement(clientID, *target, throughputEndpoint)
		
		// Print measurement results
		printMeasurement(event)

		// TODO: Send to ingest API when implemented
		// For now, just log the measurement
		
		if *once {
			break
		}

		time.Sleep(*interval)
	}
}

func performMeasurement(clientID, targetURL, throughputURL string) *models.TelemetryEvent {
	log.Printf("Performing measurement for %s", targetURL)

	// Perform the measurement
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
		if measurement != nil && measurement.ErrorStage != nil {
			event.ErrorStage = measurement.ErrorStage
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
		}
		log.Printf("Measurement error: %v", err)
	} else {
		// Successful measurement
		event.Timings = models.TimingMeasurements{
			DNSMs:      measurement.DNSMs,
			TCPMs:      measurement.TCPMs,
			TLSMs:      measurement.TLSMs,
			HTTPTTFBMs: measurement.HTTPTTFBMs,
		}
		event.ThroughputKbps = measurement.ThroughputKbps
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