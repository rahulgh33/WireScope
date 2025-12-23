package models

// EventProcessor defines the interface for message queue operations.
// This abstraction allows for different message queue implementations
// (NATS JetStream, Kafka, etc.) while maintaining consistent behavior.
//
// Requirement: 3.1 - At-least-once delivery with message queue
type EventProcessor interface {
	// PublishEvent publishes a telemetry event to the message queue
	// Returns an error if publishing fails
	PublishEvent(event *TelemetryEvent) error

	// ConsumeEvents starts consuming events from the queue and processes them
	// with the provided handler function. The handler is called for each event.
	// Returns an error if consumption setup fails.
	ConsumeEvents(handler func(*TelemetryEvent) error) error

	// AckEvent acknowledges successful processing of an event
	// This should be called after the event has been durably processed
	// Returns an error if acknowledgment fails
	AckEvent(eventID string) error

	// NackEvent negatively acknowledges an event, indicating processing failure
	// The event may be redelivered based on the queue's retry policy
	// Returns an error if negative acknowledgment fails
	NackEvent(eventID string) error

	// Close gracefully shuts down the event processor
	Close() error
}

// EventHandler is a function type for processing telemetry events
type EventHandler func(*TelemetryEvent) error
