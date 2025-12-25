package metrics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/network-qoe-telemetry-platform/internal/models"
)

// Broadcaster interface for broadcasting real-time metrics
type Broadcaster interface {
	BroadcastAggregate(aggregate *models.WindowedAggregate)
	BroadcastDiagnosis(clientID string, target string, diagnosis string, severity string)
	BroadcastProbeStatus(clientID string, status string, lastSeen time.Time)
	BroadcastDashboardUpdate(summary map[string]interface{})
	Close()
}

// WebSocketBroadcaster broadcasts metrics to WebSocket clients via HTTP
type WebSocketBroadcaster struct {
	endpoint   string
	httpClient *http.Client
	buffer     chan *BroadcastMessage
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// BroadcastMessage represents a message to be broadcast
type BroadcastMessage struct {
	Channel string                 `json:"channel"`
	Data    map[string]interface{} `json:"data"`
}

// NewWebSocketBroadcaster creates a new WebSocket broadcaster
func NewWebSocketBroadcaster(wsEndpoint string) *WebSocketBroadcaster {
	ctx, cancel := context.WithCancel(context.Background())

	b := &WebSocketBroadcaster{
		endpoint: wsEndpoint,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		buffer: make(chan *BroadcastMessage, 1000),
		ctx:    ctx,
		cancel: cancel,
	}

	// Start worker goroutines
	for i := 0; i < 4; i++ {
		b.wg.Add(1)
		go b.worker()
	}

	return b
}

// worker processes broadcast messages from the buffer
func (b *WebSocketBroadcaster) worker() {
	defer b.wg.Done()

	for {
		select {
		case msg := <-b.buffer:
			if err := b.send(msg); err != nil {
				log.Printf("[Broadcaster] Failed to send message: %v", err)
			}
		case <-b.ctx.Done():
			return
		}
	}
}

// send sends a broadcast message to the WebSocket server
func (b *WebSocketBroadcaster) send(msg *BroadcastMessage) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(b.ctx, "POST", b.endpoint+"/broadcast", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// BroadcastAggregate broadcasts a new aggregate to relevant subscribers
func (b *WebSocketBroadcaster) BroadcastAggregate(aggregate *models.WindowedAggregate) {
	// Broadcast to client-specific channel
	clientChannel := fmt.Sprintf("client:%s", aggregate.ClientID)

	data := map[string]interface{}{
		"client_id":       aggregate.ClientID,
		"target":          aggregate.Target,
		"window_start_ts": aggregate.WindowStartTs,
		"latency_p95":     aggregate.TTFBP95,
		"latency_p50":     aggregate.TTFBP50,
		"throughput_p50":  aggregate.ThroughputP50,
		"error_rate":      calculateErrorRate(aggregate),
		"count_total":     aggregate.CountTotal,
	}

	b.enqueue(&BroadcastMessage{
		Channel: clientChannel,
		Data:    data,
	})

	// Also broadcast to dashboard channel for overview updates
	b.enqueue(&BroadcastMessage{
		Channel: "dashboard",
		Data: map[string]interface{}{
			"type":   "aggregate",
			"update": data,
		},
	})
}

// BroadcastDiagnosis broadcasts a new diagnosis event
func (b *WebSocketBroadcaster) BroadcastDiagnosis(clientID string, target string, diagnosis string, severity string) {
	data := map[string]interface{}{
		"client_id": clientID,
		"target":    target,
		"diagnosis": diagnosis,
		"severity":  severity,
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	}

	// Broadcast to client-specific channel
	b.enqueue(&BroadcastMessage{
		Channel: fmt.Sprintf("client:%s", clientID),
		Data: map[string]interface{}{
			"type":      "diagnosis",
			"diagnosis": data,
		},
	})

	// Broadcast to diagnostics channel
	b.enqueue(&BroadcastMessage{
		Channel: "diagnostics",
		Data:    data,
	})

	// Broadcast to dashboard for critical issues
	if severity == "critical" || severity == "high" {
		b.enqueue(&BroadcastMessage{
			Channel: "dashboard",
			Data: map[string]interface{}{
				"type":  "alert",
				"alert": data,
			},
		})
	}
}

// BroadcastProbeStatus broadcasts probe status updates
func (b *WebSocketBroadcaster) BroadcastProbeStatus(clientID string, status string, lastSeen time.Time) {
	data := map[string]interface{}{
		"client_id": clientID,
		"status":    status,
		"last_seen": lastSeen.UTC().Format(time.RFC3339Nano),
	}

	b.enqueue(&BroadcastMessage{
		Channel: "probes",
		Data:    data,
	})

	// Also update dashboard
	b.enqueue(&BroadcastMessage{
		Channel: "dashboard",
		Data: map[string]interface{}{
			"type":         "probe_status",
			"probe_status": data,
		},
	})
}

// BroadcastDashboardUpdate broadcasts a general dashboard update
func (b *WebSocketBroadcaster) BroadcastDashboardUpdate(summary map[string]interface{}) {
	b.enqueue(&BroadcastMessage{
		Channel: "dashboard",
		Data: map[string]interface{}{
			"type":    "summary",
			"summary": summary,
		},
	})
}

// enqueue adds a message to the broadcast buffer
func (b *WebSocketBroadcaster) enqueue(msg *BroadcastMessage) {
	select {
	case b.buffer <- msg:
	default:
		// Buffer full, drop message
		log.Printf("[Broadcaster] Buffer full, dropping message for channel: %s", msg.Channel)
	}
}

// Close stops the broadcaster
func (b *WebSocketBroadcaster) Close() {
	b.cancel()
	b.wg.Wait()
	close(b.buffer)
}

// calculateErrorRate calculates the error rate from an aggregate
func calculateErrorRate(agg *models.WindowedAggregate) float64 {
	if agg.CountTotal == 0 {
		return 0.0
	}
	return float64(agg.CountError) / float64(agg.CountTotal) * 100.0
}

// NullBroadcaster is a no-op broadcaster for when real-time updates are disabled
type NullBroadcaster struct{}

func NewNullBroadcaster() *NullBroadcaster {
	return &NullBroadcaster{}
}

func (n *NullBroadcaster) BroadcastAggregate(aggregate *models.WindowedAggregate)                  {}
func (n *NullBroadcaster) BroadcastDiagnosis(clientID, target, diagnosis, severity string)         {}
func (n *NullBroadcaster) BroadcastProbeStatus(clientID string, status string, lastSeen time.Time) {}
func (n *NullBroadcaster) BroadcastDashboardUpdate(summary map[string]interface{})                 {}
func (n *NullBroadcaster) Close()                                                                  {}
