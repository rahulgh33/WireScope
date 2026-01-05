package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rahulgh33/wirescope/internal/auth"
	"github.com/rahulgh33/wirescope/internal/database"
)

// Service provides admin operations
type Service struct {
	repo      *database.Repository
	probes    map[string]*ProbeConfig
	tokens    map[string]*APIToken
	users     map[string]*User
	settings  *SystemSettings
	userStore auth.UserStore
	config    *Config
}

// NewService creates a new admin service
func NewService(config *Config, repo *database.Repository) *Service {
	userStore := auth.NewInMemoryUserStore()
	// Initialize default users from environment or defaults
	if err := auth.InitializeDefaultUsers(userStore); err != nil {
		// Log error but don't fail - users can be created via API
		fmt.Printf("Warning: Failed to initialize default users: %v\n", err)
	}

	return &Service{
		repo:      repo,
		probes:    make(map[string]*ProbeConfig),
		tokens:    make(map[string]*APIToken),
		users:     make(map[string]*User),
		userStore: userStore,
		config:    config,
		settings: &SystemSettings{
			EventsRetentionDays:     7,
			AggregatesRetentionDays: 90,
			LatencyThresholdMs:      500,
			ThroughputThresholdMb:   10,
			ErrorRateThreshold:      0.05,
			DNSBoundThreshold:       0.6,
			HandshakeBoundSigma:     2.0,
			ServerBoundSigma:        2.0,
			ThroughputDropThreshold: 0.3,
			MaxProbesPerClient:      10,
			MaxTargetsPerProbe:      50,
			MaxSamplesPerWindow:     10000,
			RateLimitPerClientRPS:   100,
			UpdatedAt:               time.Now(),
			UpdatedBy:               "system",
		},
	}
}

// RegisterRoutes registers admin API routes
func (s *Service) RegisterRoutes(router *mux.Router) {
	// Register authentication routes first
	s.RegisterAuthRoutes(router)

	// Register dashboard and data API routes
	s.RegisterDashboardRoutes(router)

	// Register user management routes
	s.RegisterUserManagementRoutes(router)

	adminRouter := router.PathPrefix("/api/v1/admin").Subrouter()

	// Probe Management
	adminRouter.HandleFunc("/probes", s.listProbes).Methods("GET")
	adminRouter.HandleFunc("/probes", s.createProbe).Methods("POST")
	adminRouter.HandleFunc("/probes/{id}", s.getProbe).Methods("GET")
	adminRouter.HandleFunc("/probes/{id}", s.updateProbe).Methods("PUT")
	adminRouter.HandleFunc("/probes/{id}", s.deleteProbe).Methods("DELETE")

	// API Token Management
	adminRouter.HandleFunc("/tokens", s.listTokens).Methods("GET")
	adminRouter.HandleFunc("/tokens", s.createToken).Methods("POST")
	adminRouter.HandleFunc("/tokens/{id}", s.getToken).Methods("GET")
	adminRouter.HandleFunc("/tokens/{id}", s.revokeToken).Methods("DELETE")

	// User Management
	adminRouter.HandleFunc("/users", s.listUsers).Methods("GET")
	adminRouter.HandleFunc("/users", s.createUser).Methods("POST")
	adminRouter.HandleFunc("/users/{id}", s.getUser).Methods("GET")
	adminRouter.HandleFunc("/users/{id}", s.updateUser).Methods("PUT")
	adminRouter.HandleFunc("/users/{id}", s.deleteUser).Methods("DELETE")

	// System Settings
	adminRouter.HandleFunc("/settings", s.getSettings).Methods("GET")
	adminRouter.HandleFunc("/settings", s.updateSettings).Methods("PUT")

	// Database Maintenance
	adminRouter.HandleFunc("/database/stats", s.getDatabaseStats).Methods("GET")
	adminRouter.HandleFunc("/database/maintenance", s.runMaintenance).Methods("POST")
	adminRouter.HandleFunc("/database/maintenance/{id}", s.getMaintenanceTask).Methods("GET")

	// System Health
	adminRouter.HandleFunc("/health", s.getSystemHealth).Methods("GET")
}

// Probe handlers
func (s *Service) listProbes(w http.ResponseWriter, r *http.Request) {
	probes := make([]*ProbeConfig, 0, len(s.probes))
	for _, probe := range s.probes {
		// Mask API token
		maskedProbe := *probe
		if maskedProbe.APIToken != "" {
			maskedProbe.APIToken = "***masked***"
		}
		probes = append(probes, &maskedProbe)
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"probes": probes})
}

func (s *Service) createProbe(w http.ResponseWriter, r *http.Request) {
	var req CreateProbeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Generate API token for probe
	token := "tok_" + uuid.New().String()

	probe := &ProbeConfig{
		ID:          uuid.New().String(),
		ClientID:    req.ClientID,
		Name:        req.Name,
		Description: req.Description,
		Targets:     req.Targets,
		Interval:    req.Interval,
		Enabled:     req.Enabled,
		APIEndpoint: "http://localhost:8080/api/v1/ingest",
		APIToken:    token,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	s.probes[probe.ID] = probe
	respondJSON(w, http.StatusCreated, probe)
}

func (s *Service) getProbe(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	probe, ok := s.probes[id]
	if !ok {
		respondError(w, http.StatusNotFound, "Probe not found")
		return
	}

	respondJSON(w, http.StatusOK, probe)
}

func (s *Service) updateProbe(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	probe, ok := s.probes[id]
	if !ok {
		respondError(w, http.StatusNotFound, "Probe not found")
		return
	}

	var req UpdateProbeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name != nil {
		probe.Name = *req.Name
	}
	if req.Description != nil {
		probe.Description = *req.Description
	}
	if req.Targets != nil {
		probe.Targets = *req.Targets
	}
	if req.Interval != nil {
		probe.Interval = *req.Interval
	}
	if req.Enabled != nil {
		probe.Enabled = *req.Enabled
	}
	probe.UpdatedAt = time.Now()

	respondJSON(w, http.StatusOK, probe)
}

func (s *Service) deleteProbe(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if _, ok := s.probes[id]; !ok {
		respondError(w, http.StatusNotFound, "Probe not found")
		return
	}

	delete(s.probes, id)
	w.WriteHeader(http.StatusNoContent)
}

// Token handlers
func (s *Service) listTokens(w http.ResponseWriter, r *http.Request) {
	tokens := make([]*APIToken, 0, len(s.tokens))
	for _, token := range s.tokens {
		// Mask full token
		maskedToken := *token
		maskedToken.Token = ""
		tokens = append(tokens, &maskedToken)
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"tokens": tokens})
}

func (s *Service) createToken(w http.ResponseWriter, r *http.Request) {
	var req CreateAPITokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	fullToken := "tok_" + uuid.New().String()
	token := &APIToken{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Token:       fullToken, // Only shown once on creation
		TokenPrefix: fullToken[:12] + "...",
		Enabled:     true,
		ExpiresAt:   req.ExpiresAt,
		CreatedAt:   time.Now(),
		CreatedBy:   "admin", // TODO: Get from auth context
		UsageCount:  0,
	}

	s.tokens[token.ID] = token
	respondJSON(w, http.StatusCreated, token)
}

func (s *Service) getToken(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	token, ok := s.tokens[id]
	if !ok {
		respondError(w, http.StatusNotFound, "Token not found")
		return
	}

	// Don't return full token
	maskedToken := *token
	maskedToken.Token = ""
	respondJSON(w, http.StatusOK, maskedToken)
}

func (s *Service) revokeToken(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if _, ok := s.tokens[id]; !ok {
		respondError(w, http.StatusNotFound, "Token not found")
		return
	}

	delete(s.tokens, id)
	w.WriteHeader(http.StatusNoContent)
}

// User handlers
func (s *Service) listUsers(w http.ResponseWriter, r *http.Request) {
	users := make([]*User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{"users": users})
}

func (s *Service) createUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user := &User{
		ID:        uuid.New().String(),
		Username:  req.Username,
		Email:     req.Email,
		FullName:  req.FullName,
		Role:      req.Role,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.users[user.ID] = user
	respondJSON(w, http.StatusCreated, user)
}

func (s *Service) getUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	user, ok := s.users[id]
	if !ok {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	respondJSON(w, http.StatusOK, user)
}

func (s *Service) updateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	user, ok := s.users[id]
	if !ok {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.FullName != nil {
		user.FullName = *req.FullName
	}
	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.Enabled != nil {
		user.Enabled = *req.Enabled
	}
	user.UpdatedAt = time.Now()

	respondJSON(w, http.StatusOK, user)
}

func (s *Service) deleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if _, ok := s.users[id]; !ok {
		respondError(w, http.StatusNotFound, "User not found")
		return
	}

	delete(s.users, id)
	w.WriteHeader(http.StatusNoContent)
}

// Settings handlers
func (s *Service) getSettings(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, s.settings)
}

func (s *Service) updateSettings(w http.ResponseWriter, r *http.Request) {
	var req UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.EventsRetentionDays != nil {
		s.settings.EventsRetentionDays = *req.EventsRetentionDays
	}
	if req.AggregatesRetentionDays != nil {
		s.settings.AggregatesRetentionDays = *req.AggregatesRetentionDays
	}
	if req.LatencyThresholdMs != nil {
		s.settings.LatencyThresholdMs = *req.LatencyThresholdMs
	}
	if req.ThroughputThresholdMb != nil {
		s.settings.ThroughputThresholdMb = *req.ThroughputThresholdMb
	}
	if req.ErrorRateThreshold != nil {
		s.settings.ErrorRateThreshold = *req.ErrorRateThreshold
	}
	// ... update other fields similarly

	s.settings.UpdatedAt = time.Now()
	s.settings.UpdatedBy = "admin" // TODO: Get from auth context

	respondJSON(w, http.StatusOK, s.settings)
}

// Database handlers
func (s *Service) getDatabaseStats(w http.ResponseWriter, r *http.Request) {
	// Mock data - in production, query actual database stats
	stats := &DatabaseStats{
		ActiveConnections:   5,
		IdleConnections:     10,
		MaxConnections:      20,
		EventsTableSize:     1024 * 1024 * 100,  // 100 MB
		AggregatesTableSize: 1024 * 1024 * 500,  // 500 MB
		TotalDatabaseSize:   1024 * 1024 * 1000, // 1 GB
		EventsRowCount:      150000,
		AggregatesRowCount:  50000,
		AvgQueryTimeMs:      25.5,
		SlowQueriesCount:    12,
		Timestamp:           time.Now(),
	}

	respondJSON(w, http.StatusOK, stats)
}

func (s *Service) runMaintenance(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type string `json:"type"` // vacuum, analyze, cleanup
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	task := &MaintenanceTask{
		ID:          uuid.New().String(),
		Type:        req.Type,
		Description: fmt.Sprintf("Manual %s operation", req.Type),
		Status:      "pending",
	}

	// In production, this would queue the task and run it asynchronously
	go func() {
		task.Status = "running"
		task.StartedAt = timePtr(time.Now())
		time.Sleep(2 * time.Second) // Simulate work
		task.Status = "completed"
		task.CompletedAt = timePtr(time.Now())
		task.RowsAffected = 1234
		task.DurationMs = 2000
	}()

	respondJSON(w, http.StatusAccepted, task)
}

func (s *Service) getMaintenanceTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Mock response
	task := &MaintenanceTask{
		ID:           id,
		Type:         "cleanup",
		Description:  "Cleanup old events",
		Status:       "completed",
		StartedAt:    timePtr(time.Now().Add(-5 * time.Minute)),
		CompletedAt:  timePtr(time.Now()),
		RowsAffected: 5432,
		DurationMs:   120000,
	}

	respondJSON(w, http.StatusOK, task)
}

// Health handler
func (s *Service) getSystemHealth(w http.ResponseWriter, r *http.Request) {
	health := &SystemHealth{
		Status:    "healthy",
		Timestamp: time.Now(),
		Database: ComponentHealth{
			Status:    "healthy",
			Latency:   15.3,
			LastCheck: time.Now(),
		},
		NATS: ComponentHealth{
			Status:    "healthy",
			Latency:   2.1,
			LastCheck: time.Now(),
		},
		Aggregator: ComponentHealth{
			Status:    "healthy",
			Latency:   45.2,
			ErrorRate: 0.001,
			LastCheck: time.Now(),
		},
		IngestAPI: ComponentHealth{
			Status:    "healthy",
			Latency:   12.5,
			ErrorRate: 0.002,
			LastCheck: time.Now(),
		},
		WebSocket: ComponentHealth{
			Status:    "healthy",
			Message:   "125 active connections",
			LastCheck: time.Now(),
		},
		ActiveProbes:      47,
		ActiveClients:     23,
		EventsPerSecond:   156.8,
		QueueLag:          125,
		ErrorRate:         0.015,
		AvgProcessingTime: 45.2,
	}

	respondJSON(w, http.StatusOK, health)
}

// Helper functions
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func timePtr(t time.Time) *time.Time {
	return &t
}
