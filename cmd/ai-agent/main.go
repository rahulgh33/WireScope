package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rahulgh33/wirescope/internal/admin"
	"github.com/rahulgh33/wirescope/internal/ai"
	"github.com/rahulgh33/wirescope/internal/database"
	"github.com/rahulgh33/wirescope/internal/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
)

func main() {
	config := loadConfig()

	conn, err := database.NewConnection(config.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close()

	repo := database.NewRepository(conn)
	dal := ai.NewDataAccessLayer(repo)

	var llm ai.LLMProvider
	if config.AIAgent.Provider == "mock" {
		llm = ai.NewMockLLMProvider()
		log.Println("Using Mock LLM Provider")
	} else {
		llm = ai.NewOpenAIProvider(config.AIAgent.APIKey, config.AIAgent.Model)
		log.Printf("Using OpenAI LLM Provider with model: %s", config.AIAgent.Model)
	}

	agent := ai.NewAgent(llm, dal, config.AIAgent)
	sessionManager := ai.NewSessionManager(config.SessionMaxAge)

	// Create WebSocket hub
	wsHub := websocket.NewHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start WebSocket hub
	go wsHub.Run(ctx)

	server := NewAIAgentServer(agent, sessionManager, wsHub, repo, config)

	log.Printf("Starting AI Agent API server on %s", config.ServerAddr)
	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

type AIAgentServer struct {
	agent          *ai.Agent
	sessionManager *ai.SessionManager
	wsHub          *websocket.Hub
	repo           *database.Repository
	config         Config
	httpServer     *http.Server
}

func NewAIAgentServer(agent *ai.Agent, sessionManager *ai.SessionManager, wsHub *websocket.Hub, repo *database.Repository, config Config) *AIAgentServer {
	return &AIAgentServer{
		agent:          agent,
		sessionManager: sessionManager,
		wsHub:          wsHub,
		repo:           repo,
		config:         config,
	}
}

func (s *AIAgentServer) Start() error {
	router := mux.NewRouter()

	// AI Agent API routes
	api := router.PathPrefix("/api/v1/ai").Subrouter()
	api.Use(s.authMiddleware)
	api.HandleFunc("/query", s.handleQuery).Methods("POST")
	api.HandleFunc("/sessions", s.handleListSessions).Methods("GET")
	api.HandleFunc("/sessions", s.handleCreateSession).Methods("POST")
	api.HandleFunc("/sessions/{id}", s.handleGetSession).Methods("GET")
	api.HandleFunc("/sessions/{id}", s.handleDeleteSession).Methods("DELETE")
	api.HandleFunc("/capabilities", s.handleGetCapabilities).Methods("GET")

	// Admin API routes
	adminConfig := &admin.Config{
		TLS: admin.TLSConfig{
			Enabled: false, // Will be configured via environment/config file
		},
	}
	adminService := admin.NewService(adminConfig, s.repo)
	adminService.RegisterRoutes(router)

	// WebSocket endpoint for real-time metrics
	wsHandler := websocket.NewHandler(s.wsHub)
	router.HandleFunc("/api/v1/ws/metrics", wsHandler.ServeHTTP)
	router.HandleFunc("/api/v1/ws/broadcast", wsHandler.HandleBroadcast).Methods("POST")

	router.HandleFunc("/health", s.handleHealth).Methods("GET")
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:5174"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-CSRF-Token"},
		AllowCredentials: true,
	})

	handler := corsHandler.Handler(router)

	s.httpServer = &http.Server{
		Addr:         s.config.ServerAddr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	return s.httpServer.ListenAndServe()
}

func (s *AIAgentServer) handleQuery(w http.ResponseWriter, r *http.Request) {
	var req ai.QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.TimeRange.Start.IsZero() {
		req.TimeRange = ai.TimeRange{
			Start: time.Now().Add(-24 * time.Hour),
			End:   time.Now(),
		}
	}

	var session *ai.Session
	if req.SessionID != "" {
		var err error
		session, err = s.sessionManager.GetSession(r.Context(), req.SessionID)
		if err != nil {
			userID := r.Header.Get("X-User-ID")
			if userID == "" {
				userID = "anonymous"
			}
			session, _ = s.sessionManager.CreateSession(r.Context(), userID)
		}
	} else {
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			userID = "anonymous"
		}
		session, _ = s.sessionManager.CreateSession(r.Context(), userID)
	}

	if session != nil {
		session.AddMessage("user", req.Query, nil)
	}

	startTime := time.Now()
	response, err := s.agent.Query(r.Context(), req)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, fmt.Sprintf("Query failed: %v", err))
		return
	}

	response.Metadata.QueryTimeMs = time.Since(startTime).Milliseconds()
	response.SessionID = req.SessionID

	if session != nil {
		session.AddMessage("assistant", response.Response.Text, response.Response.Data)
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *AIAgentServer) handleListSessions(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "anonymous"
	}

	sessions, err := s.sessionManager.ListUserSessions(r.Context(), userID)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to list sessions")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"sessions": sessions,
	})
}

func (s *AIAgentServer) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		userID = "anonymous"
	}

	session, err := s.sessionManager.CreateSession(r.Context(), userID)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	s.respondJSON(w, http.StatusCreated, session)
}

func (s *AIAgentServer) handleGetSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	session, err := s.sessionManager.GetSession(r.Context(), sessionID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "Session not found")
		return
	}

	s.respondJSON(w, http.StatusOK, session)
}

func (s *AIAgentServer) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	if err := s.sessionManager.DeleteSession(r.Context(), sessionID); err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to delete session")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *AIAgentServer) handleGetCapabilities(w http.ResponseWriter, r *http.Request) {
	capabilities := map[string]interface{}{
		"version": "1.0.0",
		"capabilities": []string{
			"natural_language_query",
			"performance_analysis",
			"client_comparison",
			"trend_analysis",
		},
		"supported_metrics": []string{
			"latency_p95",
			"throughput_p50",
			"error_rate",
			"success_rate",
		},
	}

	s.respondJSON(w, http.StatusOK, capabilities)
}

func (s *AIAgentServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

func (s *AIAgentServer) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("Authorization")
		if apiKey == "" {
			apiKey = r.URL.Query().Get("api_key")
		}

		if apiKey == "" {
			s.respondError(w, http.StatusUnauthorized, "Missing API key")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *AIAgentServer) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *AIAgentServer) respondError(w http.ResponseWriter, status int, message string) {
	s.respondJSON(w, status, map[string]string{
		"error": message,
	})
}

type Config struct {
	ServerAddr    string
	Database      *database.ConnectionConfig
	AIAgent       ai.AgentConfig
	SessionMaxAge time.Duration
}

func loadConfig() Config {
	return Config{
		ServerAddr: getEnv("SERVER_ADDR", ":8080"),
		Database: &database.ConnectionConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     5432,
			Database: getEnv("DB_NAME", "telemetry"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			SSLMode:  "disable",
		},
		AIAgent: ai.AgentConfig{
			Provider:         getEnv("AI_PROVIDER", "mock"),
			Model:            getEnv("AI_MODEL", "gpt-4"),
			APIKey:           getEnv("OPENAI_API_KEY", ""),
			Temperature:      0.2,
			MaxTokens:        2000,
			MaxContextTokens: 8000,
			EnableCaching:    true,
		},
		SessionMaxAge: 24 * time.Hour,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
