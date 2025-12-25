package websocket

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development
		// TODO: Restrict in production
		return true
	},
}

// Handler handles WebSocket upgrade requests
type Handler struct {
	hub *Hub
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

// ServeHTTP handles the HTTP request and upgrades to WebSocket
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get token from query parameter
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Token required", http.StatusUnauthorized)
		return
	}

	// TODO: Validate token and get user ID
	// For now, use a simple approach
	userID := h.validateToken(token)
	if userID == "" {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebSocket Handler] Failed to upgrade connection: %v", err)
		return
	}

	// Create new client
	client := NewClient(h.hub, conn, userID)

	// Register client with hub
	h.hub.register <- client

	// Start client goroutines
	client.Run()

	log.Printf("[WebSocket Handler] Client %s connected (user: %s)", client.id, userID)
}

// HandleBroadcast handles internal broadcast requests from the aggregator
func (h *Handler) HandleBroadcast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var msg struct {
		Channel string                 `json:"channel"`
		Data    map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if msg.Channel == "" {
		http.Error(w, "Channel is required", http.StatusBadRequest)
		return
	}

	// Broadcast to channel
	h.hub.BroadcastToChannel(msg.Channel, msg.Data)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// validateToken validates the authentication token and returns the user ID
// This is a placeholder - implement proper token validation
func (h *Handler) validateToken(token string) string {
	// TODO: Implement proper JWT validation or API key lookup
	// For now, accept any non-empty token and return a demo user ID
	if token != "" {
		return "user-" + token
	}
	return ""
}
