package admin

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// RegisterDashboardRoutes registers dashboard and data API routes
func (s *Service) RegisterDashboardRoutes(router *mux.Router) {
	api := router.PathPrefix("/api/v1").Subrouter()

	// Dashboard
	api.HandleFunc("/dashboard/overview", s.getDashboardOverview).Methods("GET")
	api.HandleFunc("/dashboard/timeseries", s.getTimeSeries).Methods("GET")

	// Clients
	api.HandleFunc("/clients", s.getClients).Methods("GET")
	api.HandleFunc("/clients/{id}", s.getClientDetail).Methods("GET")
	api.HandleFunc("/clients/{id}/performance", s.getClientPerformance).Methods("GET")

	// Targets
	api.HandleFunc("/targets", s.getTargets).Methods("GET")
	api.HandleFunc("/targets/{target}", s.getTargetDetail).Methods("GET")

	// Diagnostics
	api.HandleFunc("/diagnostics", s.getDiagnostics).Methods("GET")
	api.HandleFunc("/diagnostics/trends", s.getDiagnosticsTrends).Methods("GET")
}

// Dashboard handlers
func (s *Service) getDashboardOverview(w http.ResponseWriter, r *http.Request) {
	overview := map[string]interface{}{
		"total_events":      125837,
		"active_clients":    47,
		"avg_latency_ms":    123.5,
		"error_rate":        0.023,
		"timestamp":         time.Now().Format(time.RFC3339),
		"events_per_second": 156.8,
		"p95_latency_ms":    287.3,
		"p99_latency_ms":    445.2,
		"healthy_targets":   23,
		"degraded_targets":  2,
		"unhealthy_targets": 1,
	}
	respondJSON(w, http.StatusOK, overview)
}

func (s *Service) getTimeSeries(w http.ResponseWriter, r *http.Request) {
	// Generate sample time series data
	now := time.Now()
	dataPoints := make([]map[string]interface{}, 0, 24)

	for i := 23; i >= 0; i-- {
		timestamp := now.Add(time.Duration(-i) * time.Hour)
		dataPoints = append(dataPoints, map[string]interface{}{
			"timestamp": timestamp.Format(time.RFC3339),
			"p50":       100 + float64(i*2),
			"p95":       200 + float64(i*5),
			"p99":       300 + float64(i*8),
			"count":     1000 + i*50,
		})
	}

	response := map[string]interface{}{
		"metric":      "latency_ms",
		"time_series": dataPoints,
	}
	respondJSON(w, http.StatusOK, response)
}

// Client handlers
func (s *Service) getClients(w http.ResponseWriter, r *http.Request) {
	clients := []map[string]interface{}{
		{
			"id":             "client-001",
			"name":           "client-001",
			"status":         "active",
			"last_seen":      time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
			"avg_latency_ms": 134.2,
			"total_requests": 5432,
			"error_count":    12,
			"active_targets": 3,
		},
		{
			"id":             "client-002",
			"name":           "client-002",
			"status":         "active",
			"last_seen":      time.Now().Add(-2 * time.Minute).Format(time.RFC3339),
			"avg_latency_ms": 98.7,
			"total_requests": 8765,
			"error_count":    5,
			"active_targets": 5,
		},
		{
			"id":             "client-003",
			"name":           "client-003",
			"status":         "warning",
			"last_seen":      time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
			"avg_latency_ms": 245.8,
			"total_requests": 3210,
			"error_count":    45,
			"active_targets": 2,
		},
		{
			"id":             "client-004",
			"name":           "client-004",
			"status":         "inactive",
			"last_seen":      time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			"avg_latency_ms": 187.3,
			"total_requests": 1890,
			"error_count":    8,
			"active_targets": 1,
		},
	}

	response := map[string]interface{}{
		"clients":      clients,
		"total":        len(clients),
		"active_count": 2,
	}
	respondJSON(w, http.StatusOK, response)
}

func (s *Service) getClientDetail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clientID := vars["id"]

	client := map[string]interface{}{
		"id":             clientID,
		"name":           clientID,
		"status":         "active",
		"last_seen":      time.Now().Add(-5 * time.Minute).Format(time.RFC3339),
		"first_seen":     time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339),
		"avg_latency_ms": 134.2,
		"p95_latency_ms": 287.3,
		"p99_latency_ms": 445.2,
		"total_requests": 5432,
		"error_count":    12,
		"error_rate":     0.022,
		"active_targets": 3,
		"version":        "1.0.0",
		"location":       "US-West",
	}

	respondJSON(w, http.StatusOK, client)
}

func (s *Service) getClientPerformance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clientID := vars["id"]

	// Generate sample performance data
	now := time.Now()
	dataPoints := make([]map[string]interface{}, 0, 24)

	for i := 23; i >= 0; i-- {
		timestamp := now.Add(time.Duration(-i) * time.Hour)
		dataPoints = append(dataPoints, map[string]interface{}{
			"timestamp":     timestamp.Format(time.RFC3339),
			"latency_ms":    100 + float64(i*3),
			"error_rate":    0.01 + float64(i)*0.001,
			"request_count": 200 + i*10,
		})
	}

	response := map[string]interface{}{
		"client_id":   clientID,
		"time_series": dataPoints,
	}
	respondJSON(w, http.StatusOK, response)
}

// Target handlers
func (s *Service) getTargets(w http.ResponseWriter, r *http.Request) {
	targets := []map[string]interface{}{
		{
			"target":         "api.example.com",
			"status":         "healthy",
			"avg_latency_ms": 98.7,
			"request_count":  15432,
			"error_count":    23,
			"active_clients": 12,
			"last_checked":   time.Now().Add(-1 * time.Minute).Format(time.RFC3339),
		},
		{
			"target":         "cdn.example.com",
			"status":         "healthy",
			"avg_latency_ms": 45.2,
			"request_count":  28765,
			"error_count":    8,
			"active_clients": 18,
			"last_checked":   time.Now().Add(-30 * time.Second).Format(time.RFC3339),
		},
		{
			"target":         "db.example.com",
			"status":         "degraded",
			"avg_latency_ms": 234.8,
			"request_count":  9876,
			"error_count":    156,
			"active_clients": 8,
			"last_checked":   time.Now().Add(-2 * time.Minute).Format(time.RFC3339),
		},
	}

	response := map[string]interface{}{
		"targets": targets,
		"total":   len(targets),
	}
	respondJSON(w, http.StatusOK, response)
}

func (s *Service) getTargetDetail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	target := vars["target"]

	detail := map[string]interface{}{
		"target":          target,
		"status":          "healthy",
		"avg_latency_ms":  98.7,
		"p95_latency_ms":  187.3,
		"p99_latency_ms":  298.5,
		"request_count":   15432,
		"error_count":     23,
		"error_rate":      0.015,
		"active_clients":  12,
		"last_checked":    time.Now().Add(-1 * time.Minute).Format(time.RFC3339),
		"first_seen":      time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339),
		"dns_latency_ms":  12.3,
		"tcp_latency_ms":  45.6,
		"tls_latency_ms":  23.4,
		"ttfb_ms":         67.8,
		"throughput_mbps": 125.6,
	}

	respondJSON(w, http.StatusOK, detail)
}

// Diagnostics handlers
func (s *Service) getDiagnostics(w http.ResponseWriter, r *http.Request) {
	diagnostics := []map[string]interface{}{
		{
			"id":          "diag-001",
			"timestamp":   time.Now().Add(-6 * time.Minute).Format(time.RFC3339),
			"client_id":   "client-001",
			"target":      "api.example.com",
			"label":       "DNS-bound",
			"severity":    "warning",
			"description": "High DNS latency (450ms)",
			"metrics": map[string]interface{}{
				"dns_latency_ms":   450,
				"total_latency_ms": 520,
			},
		},
		{
			"id":          "diag-002",
			"timestamp":   time.Now().Add(-16 * time.Minute).Format(time.RFC3339),
			"client_id":   "client-002",
			"target":      "cdn.example.com",
			"label":       "Server-bound",
			"severity":    "warning",
			"description": "Slow TTFB (280ms)",
			"metrics": map[string]interface{}{
				"ttfb_ms":          280,
				"total_latency_ms": 320,
			},
		},
		{
			"id":          "diag-003",
			"timestamp":   time.Now().Add(-31 * time.Minute).Format(time.RFC3339),
			"client_id":   "client-003",
			"target":      "api.example.com",
			"label":       "Throughput",
			"severity":    "error",
			"description": "Low throughput detected",
			"metrics": map[string]interface{}{
				"throughput_mbps": 2.3,
				"expected_mbps":   10.0,
			},
		},
	}

	response := map[string]interface{}{
		"diagnostics": diagnostics,
		"total":       len(diagnostics),
	}
	respondJSON(w, http.StatusOK, response)
}

func (s *Service) getDiagnosticsTrends(w http.ResponseWriter, r *http.Request) {
	// Generate sample trend data
	now := time.Now()
	trends := make([]map[string]interface{}, 0)

	labels := []string{"DNS-bound", "Server-bound", "Throughput", "Handshake-bound"}
	for i := 6; i >= 0; i-- {
		date := now.Add(time.Duration(-i) * 24 * time.Hour)
		for _, label := range labels {
			trends = append(trends, map[string]interface{}{
				"date":  date.Format("2006-01-02"),
				"label": label,
				"count": 10 + i*2,
			})
		}
	}

	response := map[string]interface{}{
		"trends": trends,
	}
	respondJSON(w, http.StatusOK, response)
}
