package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"
)

// Hub maintains active WebSocket connections and broadcasts messages
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from clients
	broadcast chan *Message

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Client subscriptions
	subscriptions map[string]map[*Client]bool

	mu sync.RWMutex
}

// Message represents a WebSocket message
type Message struct {
	SchemaVersion string                 `json:"schema_version"`
	Type          string                 `json:"type"`
	Channel       string                 `json:"channel,omitempty"`
	EventID       string                 `json:"event_id,omitempty"`
	Timestamp     string                 `json:"timestamp"`
	Data          map[string]interface{} `json:"data,omitempty"`
	Error         *ErrorDetails          `json:"error,omitempty"`
}

// ErrorDetails represents error information in a WebSocket message
type ErrorDetails struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:       make(map[*Client]bool),
		broadcast:     make(chan *Message, 256),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
		subscriptions: make(map[string]map[*Client]bool),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[WebSocket Hub] Client registered: %s", client.id)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				// Remove from all subscriptions
				for channel := range h.subscriptions {
					delete(h.subscriptions[channel], client)
				}
			}
			h.mu.Unlock()
			log.Printf("[WebSocket Hub] Client unregistered: %s", client.id)

		case message := <-h.broadcast:
			h.broadcastToSubscribers(message)

		case <-ticker.C:
			// Send ping to all clients
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- &Message{
					SchemaVersion: "1.0",
					Type:          "ping",
					Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
				}:
				default:
					// Client send buffer full, skip ping
				}
			}
			h.mu.RUnlock()

		case <-ctx.Done():
			log.Println("[WebSocket Hub] Shutting down")
			h.mu.Lock()
			for client := range h.clients {
				close(client.send)
			}
			h.mu.Unlock()
			return
		}
	}
}

// broadcastToSubscribers sends a message to all subscribers of a channel
func (h *Hub) broadcastToSubscribers(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if message.Channel == "" {
		// Broadcast to all clients
		for client := range h.clients {
			select {
			case client.send <- message:
			default:
				// Client send buffer full, skip
				log.Printf("[WebSocket Hub] Client %s send buffer full, skipping message", client.id)
			}
		}
		return
	}

	// Broadcast to channel subscribers
	subscribers, ok := h.subscriptions[message.Channel]
	if !ok {
		return
	}

	for client := range subscribers {
		select {
		case client.send <- message:
		default:
			// Client send buffer full, skip
			log.Printf("[WebSocket Hub] Client %s send buffer full, skipping message", client.id)
		}
	}
}

// Subscribe adds a client to a channel's subscription list
func (h *Hub) Subscribe(client *Client, channels []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, channel := range channels {
		if h.subscriptions[channel] == nil {
			h.subscriptions[channel] = make(map[*Client]bool)
		}
		h.subscriptions[channel][client] = true
	}

	log.Printf("[WebSocket Hub] Client %s subscribed to: %v", client.id, channels)
}

// Unsubscribe removes a client from channel subscriptions
func (h *Hub) Unsubscribe(client *Client, channels []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, channel := range channels {
		if subscribers, ok := h.subscriptions[channel]; ok {
			delete(subscribers, client)
		}
	}

	log.Printf("[WebSocket Hub] Client %s unsubscribed from: %v", client.id, channels)
}

// BroadcastToChannel sends a message to all subscribers of a specific channel
func (h *Hub) BroadcastToChannel(channel string, data map[string]interface{}) {
	message := &Message{
		SchemaVersion: "1.0",
		Type:          "update",
		Channel:       channel,
		EventID:       generateEventID(),
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Data:          data,
	}

	select {
	case h.broadcast <- message:
	default:
		log.Printf("[WebSocket Hub] Broadcast buffer full, dropping message for channel: %s", channel)
	}
}

// BroadcastError sends an error message to a specific client or channel
func (h *Hub) BroadcastError(channel string, errDetails *ErrorDetails) {
	message := &Message{
		SchemaVersion: "1.0",
		Type:          "error",
		Channel:       channel,
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Error:         errDetails,
	}

	select {
	case h.broadcast <- message:
	default:
		log.Printf("[WebSocket Hub] Broadcast buffer full, dropping error message")
	}
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetSubscriptionCount returns the number of active subscriptions
func (h *Hub) GetSubscriptionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	count := 0
	for _, subscribers := range h.subscriptions {
		count += len(subscribers)
	}
	return count
}

// generateEventID generates a unique event ID for messages
func generateEventID() string {
	return time.Now().UTC().Format("20060102150405.000000000")
}

// Helper function to marshal message to JSON
func (m *Message) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}
