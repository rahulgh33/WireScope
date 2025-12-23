package queue

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/network-qoe-telemetry-platform/internal/models"
)

// TestDefaultNATSConfig tests the default configuration
func TestDefaultNATSConfig(t *testing.T) {
	config := DefaultNATSConfig()

	if config.URL != nats.DefaultURL {
		t.Errorf("Expected default URL %s, got %s", nats.DefaultURL, config.URL)
	}

	if config.MaxDeliver != DefaultMaxDeliver {
		t.Errorf("Expected MaxDeliver %d, got %d", DefaultMaxDeliver, config.MaxDeliver)
	}

	if config.AckWait != DefaultAckWait {
		t.Errorf("Expected AckWait %v, got %v", DefaultAckWait, config.AckWait)
	}

	if config.MaxAckPending != DefaultMaxAckPending {
		t.Errorf("Expected MaxAckPending %d, got %d", DefaultMaxAckPending, config.MaxAckPending)
	}

	if !config.EnableDLQ {
		t.Error("Expected DLQ to be enabled by default")
	}
}

// TestNATSEventProcessor_PublishAndConsume tests the basic publish/consume flow
// This test requires a running NATS server (skip if not available)
func TestNATSEventProcessor_PublishAndConsume(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Create processor with local NATS
	config := DefaultNATSConfig()
	config.URL = "nats://localhost:4222"

	processor, err := NewNATSEventProcessor(config)
	if err != nil {
		t.Skipf("NATS server not available: %v", err)
		return
	}
	defer processor.Close()

	// Create a test event
	event := &models.TelemetryEvent{
		EventID:       uuid.New().String(),
		ClientID:      "test-client",
		TimestampMs:   time.Now().UnixMilli(),
		SchemaVersion: "1.0",
		Target:        "https://example.com",
		NetworkContext: models.NetworkContext{
			InterfaceType: "wifi",
			VPNEnabled:    false,
		},
		Timings: models.TimingMeasurements{
			DNSMs:      10.0,
			TCPMs:      20.0,
			TLSMs:      30.0,
			HTTPTTFBMs: 40.0,
		},
		ThroughputKbps: 5000.0,
	}

	// Publish the event
	err = processor.PublishEvent(event)
	if err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}

	// Consume and verify
	received := make(chan bool, 1)
	handler := func(e *models.TelemetryEvent) error {
		if e.EventID == event.EventID {
			received <- true
			// Acknowledge the event
			return processor.AckEvent(e.EventID)
		}
		return nil
	}

	err = processor.ConsumeEvents(handler)
	if err != nil {
		t.Fatalf("Failed to start consuming: %v", err)
	}

	// Wait for message with timeout
	select {
	case <-received:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for message")
	}
}

// TestNATSEventProcessor_Validation tests event validation before publishing
func TestNATSEventProcessor_Validation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	config := DefaultNATSConfig()
	config.URL = "nats://localhost:4222"

	processor, err := NewNATSEventProcessor(config)
	if err != nil {
		t.Skipf("NATS server not available: %v", err)
		return
	}
	defer processor.Close()

	// Try to publish an invalid event (missing required fields)
	invalidEvent := &models.TelemetryEvent{
		EventID: "not-a-uuid",
	}

	err = processor.PublishEvent(invalidEvent)
	if err == nil {
		t.Error("Expected error for invalid event, got nil")
	}
}

// TestNATSEventProcessor_MaxDeliveries tests DLQ routing after max deliveries
func TestNATSEventProcessor_MaxDeliveries(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	config := DefaultNATSConfig()
	config.URL = "nats://localhost:4222"
	config.MaxDeliver = 2 // Set low for faster testing

	processor, err := NewNATSEventProcessor(config)
	if err != nil {
		t.Skipf("NATS server not available: %v", err)
		return
	}
	defer processor.Close()

	// Create a test event
	event := &models.TelemetryEvent{
		EventID:       uuid.New().String(),
		ClientID:      "test-client-dlq",
		TimestampMs:   time.Now().UnixMilli(),
		SchemaVersion: "1.0",
		Target:        "https://example.com",
		NetworkContext: models.NetworkContext{
			InterfaceType: "wifi",
		},
		Timings: models.TimingMeasurements{
			DNSMs:      10.0,
			TCPMs:      20.0,
			TLSMs:      30.0,
			HTTPTTFBMs: 40.0,
		},
		ThroughputKbps: 5000.0,
	}

	// Publish the event
	err = processor.PublishEvent(event)
	if err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}

	// Handler that always fails to trigger redelivery
	deliveryCount := 0
	handler := func(e *models.TelemetryEvent) error {
		deliveryCount++
		t.Logf("Delivery attempt %d for event %s", deliveryCount, e.EventID)
		// Always fail to trigger DLQ after max deliveries
		return processor.NackEvent(e.EventID)
	}

	err = processor.ConsumeEvents(handler)
	if err != nil {
		t.Fatalf("Failed to start consuming: %v", err)
	}

	// Wait for redeliveries to complete
	time.Sleep(3 * time.Second)

	if deliveryCount < 2 {
		t.Errorf("Expected at least 2 delivery attempts, got %d", deliveryCount)
	}
}
