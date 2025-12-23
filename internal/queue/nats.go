package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/network-qoe-telemetry-platform/internal/models"
)

const (
	// Stream names
	StreamNameEvents = "telemetry-events"
	StreamNameDLQ    = "telemetry-events-dlq"

	// Subject names
	SubjectEvents = "telemetry.events"
	SubjectDLQ    = "telemetry.dlq"

	// Consumer names
	ConsumerNameAggregator = "aggregator"

	// Configuration defaults
	DefaultMaxDeliver      = 5
	DefaultAckWait         = 30 * time.Second
	DefaultMaxAckPending   = 1000
	DefaultStreamRetention = 7 * 24 * time.Hour
)

// NATSConfig holds configuration for NATS JetStream connection
type NATSConfig struct {
	URL             string
	StreamRetention time.Duration
	MaxDeliver      int
	AckWait         time.Duration
	MaxAckPending   int
	EnableDLQ       bool
	ReconnectWait   time.Duration
	MaxReconnects   int
}

// DefaultNATSConfig returns a NATSConfig with sensible defaults
func DefaultNATSConfig() *NATSConfig {
	return &NATSConfig{
		URL:             nats.DefaultURL,
		StreamRetention: DefaultStreamRetention,
		MaxDeliver:      DefaultMaxDeliver,
		AckWait:         DefaultAckWait,
		MaxAckPending:   DefaultMaxAckPending,
		EnableDLQ:       true,
		ReconnectWait:   2 * time.Second,
		MaxReconnects:   -1, // Unlimited reconnects
	}
}

// NATSEventProcessor implements the EventProcessor interface using NATS JetStream
//
// Requirements: 3.1 (at-least-once delivery), 8.4 (DLQ for poison messages)
type NATSEventProcessor struct {
	config    *NATSConfig
	nc        *nats.Conn
	js        jetstream.JetStream
	ctx       context.Context
	ctxCancel context.CancelFunc

	// Subscription management
	consumerMu sync.Mutex
	consumer   jetstream.Consumer

	// Message tracking for acknowledgment
	msgMu    sync.RWMutex
	messages map[string]jetstream.Msg
}

// NewNATSEventProcessor creates a new NATS JetStream event processor
func NewNATSEventProcessor(config *NATSConfig) (*NATSEventProcessor, error) {
	if config == nil {
		config = DefaultNATSConfig()
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())

	processor := &NATSEventProcessor{
		config:    config,
		ctx:       ctx,
		ctxCancel: cancel,
		messages:  make(map[string]jetstream.Msg),
	}

	// Connect to NATS
	if err := processor.connect(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create streams
	if err := processor.createStreams(); err != nil {
		cancel()
		processor.nc.Close()
		return nil, fmt.Errorf("failed to create streams: %w", err)
	}

	return processor, nil
}

// connect establishes connection to NATS server
func (p *NATSEventProcessor) connect() error {
	opts := []nats.Option{
		nats.ReconnectWait(p.config.ReconnectWait),
		nats.MaxReconnects(p.config.MaxReconnects),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				log.Printf("NATS disconnected: %v", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("NATS reconnected to %s", nc.ConnectedUrl())
			natsReconnectsTotal.Inc()
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Println("NATS connection closed")
		}),
	}

	nc, err := nats.Connect(p.config.URL, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS at %s: %w", p.config.URL, err)
	}

	p.nc = nc

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return fmt.Errorf("failed to create JetStream context: %w", err)
	}

	p.js = js
	return nil
}

// createStreams creates the necessary JetStream streams
func (p *NATSEventProcessor) createStreams() error {
	// Create main telemetry events stream
	eventsStream := jetstream.StreamConfig{
		Name:        StreamNameEvents,
		Subjects:    []string{SubjectEvents},
		Storage:     jetstream.FileStorage,
		Retention:   jetstream.WorkQueuePolicy,
		MaxAge:      p.config.StreamRetention,
		Replicas:    1,
		Discard:     jetstream.DiscardOld,
		Description: "Telemetry events stream with at-least-once delivery",
	}

	_, err := p.js.CreateOrUpdateStream(p.ctx, eventsStream)
	if err != nil {
		return fmt.Errorf("failed to create events stream: %w", err)
	}

	// Create DLQ stream if enabled
	if p.config.EnableDLQ {
		dlqStream := jetstream.StreamConfig{
			Name:        StreamNameDLQ,
			Subjects:    []string{SubjectDLQ},
			Storage:     jetstream.FileStorage,
			Retention:   jetstream.LimitsPolicy,
			MaxAge:      30 * 24 * time.Hour, // Keep DLQ messages for 30 days
			Replicas:    1,
			Description: "Dead letter queue for poison messages",
		}

		_, err := p.js.CreateOrUpdateStream(p.ctx, dlqStream)
		if err != nil {
			return fmt.Errorf("failed to create DLQ stream: %w", err)
		}
	}

	return nil
}

// PublishEvent publishes a telemetry event to the message queue
//
// Requirement: 3.1 - At-least-once delivery with message queue
func (p *NATSEventProcessor) PublishEvent(event *models.TelemetryEvent) error {
	// Validate event before publishing
	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}

	// Serialize event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Publish to JetStream with acknowledgment
	_, err = p.js.Publish(p.ctx, SubjectEvents, data)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// ConsumeEvents starts consuming events from the queue and processes them with the handler
//
// Requirement: 3.1 - At-least-once delivery
// Requirement: 8.4 - DLQ for poison messages
func (p *NATSEventProcessor) ConsumeEvents(handler func(*models.TelemetryEvent) error) error {
	p.consumerMu.Lock()
	defer p.consumerMu.Unlock()

	// Create or get consumer
	consumerConfig := jetstream.ConsumerConfig{
		Name:          ConsumerNameAggregator,
		Durable:       ConsumerNameAggregator,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    p.config.MaxDeliver,
		AckWait:       p.config.AckWait,
		MaxAckPending: p.config.MaxAckPending,
		FilterSubject: SubjectEvents,
		Description:   "Aggregator consumer with explicit acknowledgment",
	}

	consumer, err := p.js.CreateOrUpdateConsumer(p.ctx, StreamNameEvents, consumerConfig)
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	p.consumer = consumer

	// Start consuming messages
	_, err = consumer.Consume(func(msg jetstream.Msg) {
		// Parse the telemetry event
		var event models.TelemetryEvent
		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			log.Printf("Failed to unmarshal event: %v", err)

			// Send to DLQ if this is the last delivery attempt
			metadata, _ := msg.Metadata()
			if metadata != nil && metadata.NumDelivered >= uint64(p.config.MaxDeliver) {
				p.sendToDLQ(msg.Data(), fmt.Sprintf("unmarshal error: %v", err))
			}

			msg.Nak()
			return
		}

		// Store message for later acknowledgment
		p.msgMu.Lock()
		p.messages[event.EventID] = msg
		p.msgMu.Unlock()

		// Process the event with the handler
		if err := handler(&event); err != nil {
			log.Printf("Failed to process event %s: %v", event.EventID, err)

			// Check if this is the last delivery attempt
			metadata, _ := msg.Metadata()
			if metadata != nil && metadata.NumDelivered >= uint64(p.config.MaxDeliver) {
				log.Printf("Event %s exceeded max deliveries, sending to DLQ", event.EventID)
				p.sendToDLQ(msg.Data(), fmt.Sprintf("processing failed after %d attempts: %v", p.config.MaxDeliver, err))
				msg.Ack() // Ack to remove from main stream
			} else {
				msg.Nak() // Negative ack for retry
			}

			// Remove from tracking
			p.msgMu.Lock()
			delete(p.messages, event.EventID)
			p.msgMu.Unlock()
			return
		}

		// Handler succeeded, but don't ack yet - let AckEvent do it
		// This allows the handler to complete any database transactions first
	})

	if err != nil {
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	return nil
}

// AckEvent acknowledges successful processing of an event
//
// Requirement: 3.3 - Transactional consistency (only ACK after DB commit)
func (p *NATSEventProcessor) AckEvent(eventID string) error {
	p.msgMu.Lock()
	msg, exists := p.messages[eventID]
	if !exists {
		p.msgMu.Unlock()
		return fmt.Errorf("message not found for event ID: %s", eventID)
	}
	delete(p.messages, eventID)
	p.msgMu.Unlock()

	return msg.Ack()
}

// NackEvent negatively acknowledges an event for redelivery
//
// Requirement: 3.1 - At-least-once delivery with retry
func (p *NATSEventProcessor) NackEvent(eventID string) error {
	p.msgMu.Lock()
	msg, exists := p.messages[eventID]
	if !exists {
		p.msgMu.Unlock()
		return fmt.Errorf("message not found for event ID: %s", eventID)
	}
	delete(p.messages, eventID)
	p.msgMu.Unlock()

	return msg.Nak()
}

// sendToDLQ sends a message to the dead letter queue
//
// Requirement: 8.4 - DLQ for poison messages
func (p *NATSEventProcessor) sendToDLQ(data []byte, reason string) {
	if !p.config.EnableDLQ {
		return
	}

	// Increment DLQ counter
	dlqMessagesTotal.Inc()

	// Create DLQ message with metadata
	dlqMsg := map[string]interface{}{
		"original_data": string(data),
		"reason":        reason,
		"timestamp":     time.Now().Unix(),
	}

	dlqData, err := json.Marshal(dlqMsg)
	if err != nil {
		log.Printf("Failed to marshal DLQ message: %v", err)
		return
	}

	// Publish to DLQ (fire and forget)
	_, err = p.js.Publish(p.ctx, SubjectDLQ, dlqData)
	if err != nil {
		log.Printf("Failed to publish to DLQ: %v", err)
	}
}

// Close gracefully shuts down the event processor
func (p *NATSEventProcessor) Close() error {
	p.ctxCancel()

	if p.nc != nil {
		p.nc.Close()
	}

	return nil
}

// GetConsumerInfo returns information about the consumer for monitoring
func (p *NATSEventProcessor) GetConsumerInfo() (*jetstream.ConsumerInfo, error) {
	p.consumerMu.Lock()
	defer p.consumerMu.Unlock()

	if p.consumer == nil {
		return nil, fmt.Errorf("consumer not initialized")
	}

	return p.consumer.Info(p.ctx)
}

// RepublishFromDLQ republishes a message from the DLQ back to the main stream
//
// Requirement: 8.4 - DLQ republish logic for manual replay
func (p *NATSEventProcessor) RepublishFromDLQ(dlqMessageID string) error {
	if !p.config.EnableDLQ {
		return fmt.Errorf("DLQ is not enabled")
	}

	// Get the DLQ stream
	stream, err := p.js.Stream(p.ctx, StreamNameDLQ)
	if err != nil {
		return fmt.Errorf("failed to get DLQ stream: %w", err)
	}

	// Create a temporary consumer to read from DLQ
	consumerConfig := jetstream.ConsumerConfig{
		FilterSubject: SubjectDLQ,
		AckPolicy:     jetstream.AckExplicitPolicy,
	}

	consumer, err := stream.CreateConsumer(p.ctx, consumerConfig)
	if err != nil {
		return fmt.Errorf("failed to create DLQ consumer: %w", err)
	}

	// Fetch messages and find the one to republish
	var found bool
	msgBatch, err := consumer.Fetch(100) // Fetch up to 100 messages
	if err != nil {
		return fmt.Errorf("failed to fetch from DLQ: %w", err)
	}

	for msg := range msgBatch.Messages() {
		// Parse DLQ message
		var dlqMsg map[string]interface{}
		if err := json.Unmarshal(msg.Data(), &dlqMsg); err != nil {
			msg.Ack()
			continue
		}

		// Extract original data
		originalData, ok := dlqMsg["original_data"].(string)
		if !ok {
			msg.Ack()
			continue
		}

		// Parse original event to get event ID
		var event models.TelemetryEvent
		if err := json.Unmarshal([]byte(originalData), &event); err != nil {
			msg.Ack()
			continue
		}

		// Check if this is the message we want to republish
		if event.EventID == dlqMessageID {
			// Republish to main stream
			_, err := p.js.Publish(p.ctx, SubjectEvents, []byte(originalData))
			if err != nil {
				return fmt.Errorf("failed to republish event: %w", err)
			}

			// Acknowledge the DLQ message
			msg.Ack()
			found = true
			log.Printf("Successfully republished event %s from DLQ", event.EventID)
			break
		}

		msg.Ack()
	}

	if !found {
		return fmt.Errorf("message with ID %s not found in DLQ", dlqMessageID)
	}

	return nil
}

// ListDLQMessages returns a list of messages currently in the DLQ
//
// Requirement: 8.4 - DLQ inspection for operational visibility
func (p *NATSEventProcessor) ListDLQMessages(limit int) ([]map[string]interface{}, error) {
	if !p.config.EnableDLQ {
		return nil, fmt.Errorf("DLQ is not enabled")
	}

	// Get the DLQ stream
	stream, err := p.js.Stream(p.ctx, StreamNameDLQ)
	if err != nil {
		return nil, fmt.Errorf("failed to get DLQ stream: %w", err)
	}

	// Create a temporary consumer
	consumerConfig := jetstream.ConsumerConfig{
		FilterSubject: SubjectDLQ,
		AckPolicy:     jetstream.AckNonePolicy, // Don't require ack for listing
	}

	consumer, err := stream.CreateConsumer(p.ctx, consumerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create DLQ consumer: %w", err)
	}

	// Fetch messages
	msgBatch, err := consumer.Fetch(limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from DLQ: %w", err)
	}

	var messages []map[string]interface{}
	for msg := range msgBatch.Messages() {
		var dlqMsg map[string]interface{}
		if err := json.Unmarshal(msg.Data(), &dlqMsg); err != nil {
			continue
		}
		messages = append(messages, dlqMsg)
	}

	return messages, nil
}

// UpdateQueueMetrics updates Prometheus metrics for queue status
// This should be called periodically to monitor queue health
//
// Requirement: 6.2 - Queue lag monitoring
func (p *NATSEventProcessor) UpdateQueueMetrics() error {
	info, err := p.GetConsumerInfo()
	if err != nil {
		return fmt.Errorf("failed to get consumer info: %w", err)
	}

	// Update queue lag (pending messages in stream)
	queueLagMessages.Set(float64(info.NumPending))

	// Update ack pending (messages delivered but not acknowledged)
	queueAckPendingMessages.Set(float64(info.NumAckPending))

	return nil
}
