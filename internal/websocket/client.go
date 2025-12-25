package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = 30 * time.Second

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512KB
)

// Client represents a WebSocket client connection
type Client struct {
	id string

	hub *Hub

	// WebSocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan *Message

	// Client metadata
	userID string
}

// SubscribeRequest represents a subscription request from the client
type SubscribeRequest struct {
	SchemaVersion string   `json:"schema_version"`
	Type          string   `json:"type"`
	Channels      []string `json:"channels"`
	Timestamp     string   `json:"timestamp"`
	LastEventID   string   `json:"last_event_id,omitempty"`
}

// UnsubscribeRequest represents an unsubscribe request
type UnsubscribeRequest struct {
	SchemaVersion string                 `json:"schema_version"`
	Type          string                 `json:"type"`
	Timestamp     string                 `json:"timestamp"`
	Data          map[string]interface{} `json:"data"`
}

// NewClient creates a new Client
func NewClient(hub *Hub, conn *websocket.Conn, userID string) *Client {
	return &Client{
		id:     uuid.New().String(),
		hub:    hub,
		conn:   conn,
		send:   make(chan *Message, 256),
		userID: userID,
	}
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WebSocket Client %s] Unexpected close error: %v", c.id, err)
			}
			break
		}

		// Parse message
		var rawMsg map[string]interface{}
		if err := json.Unmarshal(messageBytes, &rawMsg); err != nil {
			log.Printf("[WebSocket Client %s] Failed to parse message: %v", c.id, err)
			c.sendError("INVALID_MESSAGE", "Failed to parse message")
			continue
		}

		msgType, ok := rawMsg["type"].(string)
		if !ok {
			c.sendError("INVALID_MESSAGE", "Message type is required")
			continue
		}

		switch msgType {
		case "subscribe":
			c.handleSubscribe(messageBytes)
		case "unsubscribe":
			c.handleUnsubscribe(messageBytes)
		case "pong":
			// Client responded to ping, already handled by pong handler
		default:
			log.Printf("[WebSocket Client %s] Unknown message type: %s", c.id, msgType)
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			// Write message as JSON
			jsonData, err := json.Marshal(message)
			if err != nil {
				log.Printf("[WebSocket Client %s] Failed to marshal message: %v", c.id, err)
				w.Close()
				continue
			}

			w.Write(jsonData)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				nextMsg := <-c.send
				jsonData, err := json.Marshal(nextMsg)
				if err != nil {
					log.Printf("[WebSocket Client %s] Failed to marshal queued message: %v", c.id, err)
					continue
				}
				w.Write(jsonData)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleSubscribe handles subscription requests from the client
func (c *Client) handleSubscribe(messageBytes []byte) {
	var req SubscribeRequest
	if err := json.Unmarshal(messageBytes, &req); err != nil {
		log.Printf("[WebSocket Client %s] Failed to parse subscribe request: %v", c.id, err)
		c.sendError("INVALID_SUBSCRIBE", "Failed to parse subscribe request")
		return
	}

	if len(req.Channels) == 0 {
		c.sendError("INVALID_SUBSCRIBE", "At least one channel is required")
		return
	}

	// Subscribe to channels
	c.hub.Subscribe(c, req.Channels)

	// Send acknowledgment
	ackMsg := &Message{
		SchemaVersion: "1.0",
		Type:          "ack",
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Data: map[string]interface{}{
			"subscribed_channels": req.Channels,
		},
	}

	select {
	case c.send <- ackMsg:
	default:
		log.Printf("[WebSocket Client %s] Failed to send ack, buffer full", c.id)
	}
}

// handleUnsubscribe handles unsubscribe requests from the client
func (c *Client) handleUnsubscribe(messageBytes []byte) {
	var req UnsubscribeRequest
	if err := json.Unmarshal(messageBytes, &req); err != nil {
		log.Printf("[WebSocket Client %s] Failed to parse unsubscribe request: %v", c.id, err)
		c.sendError("INVALID_UNSUBSCRIBE", "Failed to parse unsubscribe request")
		return
	}

	channels, ok := req.Data["channels"].([]interface{})
	if !ok || len(channels) == 0 {
		c.sendError("INVALID_UNSUBSCRIBE", "Channels are required")
		return
	}

	channelStrings := make([]string, 0, len(channels))
	for _, ch := range channels {
		if chStr, ok := ch.(string); ok {
			channelStrings = append(channelStrings, chStr)
		}
	}

	c.hub.Unsubscribe(c, channelStrings)
}

// sendError sends an error message to the client
func (c *Client) sendError(code, message string) {
	errMsg := &Message{
		SchemaVersion: "1.0",
		Type:          "error",
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Error: &ErrorDetails{
			Code:    code,
			Message: message,
		},
	}

	select {
	case c.send <- errMsg:
	default:
		log.Printf("[WebSocket Client %s] Failed to send error, buffer full", c.id)
	}
}

// Run starts the client's read and write pumps
func (c *Client) Run() {
	go c.writePump()
	go c.readPump()
}
