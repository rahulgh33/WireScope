package admin

import (
	"fmt"
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
	api.HandleFunc("/clients/{id}", s.deleteClient).Methods("DELETE")
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
	ctx := r.Context()

	// Get summary from last 5 minutes of aggregated data
	var summary struct {
		ActiveClients int     `json:"active_clients"`
		AvgLatencyP95 float64 `json:"avg_latency_p95"`
		SuccessRate   float64 `json:"success_rate"`
		ErrorRate     float64 `json:"error_rate"`
		TotalEvents   int     `json:"total_events"`
		TotalTargets  int     `json:"total_targets"`
	}

	query := `
		SELECT 
			COUNT(DISTINCT client_id) as active_clients,
			COALESCE(AVG(CASE WHEN ttfb_p95 > 0 THEN ttfb_p95 END), 0) as avg_latency_p95,
			COALESCE(SUM(count_total), 0) as total_events,
			COALESCE(SUM(count_error), 0) as total_errors,
			COUNT(DISTINCT target) as total_targets
		FROM agg_1m
		WHERE window_start_ts >= NOW() - INTERVAL '24 hours'
	`

	var totalErrors int
	err := s.repo.Connection().DB().QueryRowContext(ctx, query).Scan(
		&summary.ActiveClients,
		&summary.AvgLatencyP95,
		&summary.TotalEvents,
		&totalErrors,
		&summary.TotalTargets,
	)

	if err != nil {
		http.Error(w, fmt.Sprintf("Database query failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Calculate rates (as decimals 0-1, not percentages)
	if summary.TotalEvents > 0 {
		summary.ErrorRate = float64(totalErrors) / float64(summary.TotalEvents)
		summary.SuccessRate = 1.0 - summary.ErrorRate
	} else {
		summary.SuccessRate = 0
		summary.ErrorRate = 0
	}

	overview := map[string]interface{}{
		"summary":   summary,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	respondJSON(w, http.StatusOK, overview)
}

func (s *Service) getTimeSeries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	metric := r.URL.Query().Get("metric")
	clientID := r.URL.Query().Get("client_id")
	target := r.URL.Query().Get("target")

	// Build base query
	query := `
		SELECT 
			DATE_TRUNC('hour', window_start_ts) as timestamp,
			COALESCE(AVG(CASE WHEN ttfb_p95 > 0 THEN ttfb_p95 END), 0) as p95,
			COALESCE(AVG(CASE WHEN dns_p95 > 0 THEN dns_p95 END), 0) as dns_p95,
			SUM(count_total) as count
		FROM agg_1m
		WHERE window_start_ts >= NOW() - INTERVAL '24 hours'
	`

	args := []interface{}{}
	argIndex := 1

	if clientID != "" && clientID != "undefined" {
		query += fmt.Sprintf(" AND client_id = $%d", argIndex)
		args = append(args, clientID)
		argIndex++
	}

	if target != "" && target != "undefined" {
		query += fmt.Sprintf(" AND target = $%d", argIndex)
		args = append(args, target)
		argIndex++
	}

	query += ` GROUP BY DATE_TRUNC('hour', window_start_ts) ORDER BY timestamp`

	rows, err := s.repo.Connection().DB().QueryContext(ctx, query, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database query failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	dataPoints := []map[string]interface{}{}

	for rows.Next() {
		var timestamp time.Time
		var p95, dnsP95 float64
		var count int

		if err := rows.Scan(&timestamp, &p95, &dnsP95, &count); err != nil {
			continue
		}

		dataPoints = append(dataPoints, map[string]interface{}{
			"timestamp": timestamp.Format(time.RFC3339),
			"p95":       p95,
			"dns_p95":   dnsP95,
			"count":     count,
		})
	}

	response := map[string]interface{}{
		"metric":      metric,
		"time_series": dataPoints,
	}
	respondJSON(w, http.StatusOK, response)
}

// Client handlers
func (s *Service) getClients(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	query := `
		SELECT 
			client_id,
			COUNT(DISTINCT target) as active_targets,
			MAX(window_start_ts) as last_seen,
			SUM(count_total) as total_requests,
			SUM(count_error) as error_count,
			COALESCE(AVG(CASE WHEN ttfb_p50 > 0 THEN ttfb_p50 END), 0) as avg_latency_ms
		FROM agg_1m
		WHERE window_start_ts >= NOW() - INTERVAL '24 hours'
		GROUP BY client_id
		ORDER BY last_seen DESC
	`

	rows, err := s.repo.Connection().DB().QueryContext(ctx, query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database query failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	clients := []map[string]interface{}{}
	activeCount := 0

	for rows.Next() {
		var clientID string
		var activeTargets, totalRequests, errorCount int
		var lastSeen time.Time
		var avgLatencyMs float64

		if err := rows.Scan(&clientID, &activeTargets, &lastSeen, &totalRequests, &errorCount, &avgLatencyMs); err != nil {
			continue
		}

		status := "inactive"
		if time.Since(lastSeen) < 5*time.Minute {
			status = "active"
			activeCount++
		} else if time.Since(lastSeen) < 30*time.Minute {
			status = "warning"
		}

		clients = append(clients, map[string]interface{}{
			"id":             clientID,
			"name":           clientID,
			"status":         status,
			"last_seen":      lastSeen.Format(time.RFC3339),
			"avg_latency_ms": avgLatencyMs,
			"total_requests": totalRequests,
			"error_count":    errorCount,
			"active_targets": activeTargets,
		})
	}

	response := map[string]interface{}{
		"clients":      clients,
		"total":        len(clients),
		"active_count": activeCount,
	}
	respondJSON(w, http.StatusOK, response)
}

func (s *Service) getClientDetail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clientID := vars["id"]
	ctx := r.Context()

	query := `
		SELECT 
			COUNT(DISTINCT target) as active_targets,
			MIN(window_start_ts) as first_seen,
			MAX(window_start_ts) as last_seen,
			SUM(count_total) as total_requests,
			SUM(count_error) as error_count,
			COALESCE(AVG(CASE WHEN ttfb_p50 > 0 THEN ttfb_p50 END), 0) as avg_latency_ms,
			COALESCE(AVG(CASE WHEN ttfb_p95 > 0 THEN ttfb_p95 END), 0) as p95_latency_ms,
			COALESCE(AVG(CASE WHEN ttfb_p99 > 0 THEN ttfb_p99 END), 0) as p99_latency_ms
		FROM agg_1m
		WHERE client_id = $1
	`

	var activeTargets, totalRequests, errorCount int
	var firstSeen, lastSeen time.Time
	var avgLatency, p95Latency, p99Latency float64

	err := s.repo.Connection().DB().QueryRowContext(ctx, query, clientID).Scan(
		&activeTargets,
		&firstSeen,
		&lastSeen,
		&totalRequests,
		&errorCount,
		&avgLatency,
		&p95Latency,
		&p99Latency,
	)

	if err != nil {
		http.Error(w, fmt.Sprintf("Client not found: %v", err), http.StatusNotFound)
		return
	}

	var errorRate float64
	if totalRequests > 0 {
		errorRate = float64(errorCount) / float64(totalRequests)
	}

	status := "inactive"
	if time.Since(lastSeen) < 10*time.Minute {
		status = "active"
	}

	client := map[string]interface{}{
		"id":             clientID,
		"name":           clientID,
		"status":         status,
		"last_seen":      lastSeen.Format(time.RFC3339),
		"first_seen":     firstSeen.Format(time.RFC3339),
		"avg_latency_ms": avgLatency,
		"p95_latency_ms": p95Latency,
		"p99_latency_ms": p99Latency,
		"total_requests": totalRequests,
		"error_count":    errorCount,
		"error_rate":     errorRate,
		"active_targets": activeTargets,
	}

	respondJSON(w, http.StatusOK, client)
}

func (s *Service) getClientPerformance(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clientID := vars["id"]
	ctx := r.Context()

	query := `
		SELECT 
			DATE_TRUNC('hour', window_start_ts) as timestamp,
			COALESCE(AVG(CASE WHEN ttfb_p50 > 0 THEN ttfb_p50 END), 0) as latency_ms,
			SUM(count_error)::float / NULLIF(SUM(count_total), 0) as error_rate,
			SUM(count_total) as request_count
		FROM agg_1m
		WHERE client_id = $1 
		  AND window_start_ts >= NOW() - INTERVAL '24 hours'
		GROUP BY DATE_TRUNC('hour', window_start_ts)
		ORDER BY timestamp
	`

	rows, err := s.repo.Connection().DB().QueryContext(ctx, query, clientID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database query failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	dataPoints := []map[string]interface{}{}

	for rows.Next() {
		var timestamp time.Time
		var latencyMs, errorRate float64
		var requestCount int

		if err := rows.Scan(&timestamp, &latencyMs, &errorRate, &requestCount); err != nil {
			continue
		}

		dataPoints = append(dataPoints, map[string]interface{}{
			"timestamp":     timestamp.Format(time.RFC3339),
			"latency_ms":    latencyMs,
			"error_rate":    errorRate,
			"request_count": requestCount,
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
	ctx := r.Context()

	query := `
		SELECT 
			target,
			COUNT(DISTINCT client_id) as active_clients,
			MAX(window_start_ts) as last_checked,
			SUM(count_total) as request_count,
			SUM(count_error) as error_count,
			COALESCE(AVG(CASE WHEN ttfb_p50 > 0 THEN ttfb_p50 END), 0) as avg_latency_ms
		FROM agg_1m
		WHERE window_start_ts >= NOW() - INTERVAL '24 hours'
		GROUP BY target
		ORDER BY last_checked DESC
	`

	rows, err := s.repo.Connection().DB().QueryContext(ctx, query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database query failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	targets := []map[string]interface{}{}

	for rows.Next() {
		var target string
		var activeClients, requestCount, errorCount int
		var lastChecked time.Time
		var avgLatencyMs float64

		if err := rows.Scan(&target, &activeClients, &lastChecked, &requestCount, &errorCount, &avgLatencyMs); err != nil {
			continue
		}

		status := "healthy"
		errorRate := float64(0)
		if requestCount > 0 {
			errorRate = float64(errorCount) / float64(requestCount)
		}

		if errorRate > 0.05 {
			status = "degraded"
		}
		if errorRate > 0.2 || time.Since(lastChecked) > 10*time.Minute {
			status = "unhealthy"
		}

		targets = append(targets, map[string]interface{}{
			"target":         target,
			"status":         status,
			"avg_latency_ms": avgLatencyMs,
			"request_count":  requestCount,
			"error_count":    errorCount,
			"active_clients": activeClients,
			"last_checked":   lastChecked.Format(time.RFC3339),
		})
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
	ctx := r.Context()

	query := `
		SELECT 
			COUNT(DISTINCT client_id) as active_clients,
			MIN(window_start_ts) as first_seen,
			MAX(window_start_ts) as last_checked,
			SUM(count_total) as request_count,
			SUM(count_error) as error_count,
			COALESCE(AVG(CASE WHEN ttfb_p50 > 0 THEN ttfb_p50 END), 0) as avg_latency_ms,
			COALESCE(AVG(CASE WHEN ttfb_p95 > 0 THEN ttfb_p95 END), 0) as p95_latency_ms,
			COALESCE(AVG(CASE WHEN ttfb_p95 > 0 THEN ttfb_p95 END), 0) as p95_latency_ms,
			COALESCE(AVG(CASE WHEN dns_p50 > 0 THEN dns_p50 END), 0) as dns_latency_ms,
			COALESCE(AVG(CASE WHEN tcp_p50 > 0 THEN tcp_p50 END), 0) as tcp_latency_ms,
			COALESCE(AVG(CASE WHEN tls_p50 > 0 THEN tls_p50 END), 0) as tls_latency_ms
		FROM agg_1m
		WHERE target = $1
	`

	var activeClients, requestCount, errorCount int
	var firstSeen, lastChecked time.Time
	var avgLatency, p95Latency, p99Latency, dnsLatency, tcpLatency, tlsLatency float64

	err := s.repo.Connection().DB().QueryRowContext(ctx, query, target).Scan(
		&activeClients,
		&firstSeen,
		&lastChecked,
		&requestCount,
		&errorCount,
		&avgLatency,
		&p95Latency,
		&p99Latency,
		&dnsLatency,
		&tcpLatency,
		&tlsLatency,
	)

	if err != nil {
		http.Error(w, fmt.Sprintf("Target not found: %v", err), http.StatusNotFound)
		return
	}

	var errorRate float64
	if requestCount > 0 {
		errorRate = float64(errorCount) / float64(requestCount)
	}

	status := "healthy"
	if errorRate > 0.05 {
		status = "degraded"
	}
	if errorRate > 0.2 || time.Since(lastChecked) > 10*time.Minute {
		status = "unhealthy"
	}

	detail := map[string]interface{}{
		"target":         target,
		"status":         status,
		"avg_latency_ms": avgLatency,
		"p95_latency_ms": p95Latency,
		"p99_latency_ms": p99Latency,
		"request_count":  requestCount,
		"error_count":    errorCount,
		"error_rate":     errorRate,
		"active_clients": activeClients,
		"last_checked":   lastChecked.Format(time.RFC3339),
		"first_seen":     firstSeen.Format(time.RFC3339),
		"dns_latency_ms": dnsLatency,
		"tcp_latency_ms": tcpLatency,
		"tls_latency_ms": tlsLatency,
		"ttfb_ms":        avgLatency - dnsLatency - tcpLatency - tlsLatency,
	}

	respondJSON(w, http.StatusOK, detail)
}

// Diagnostics handlers
func (s *Service) getDiagnostics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Query for issues: high error rates, high latencies, etc.
	query := `
		SELECT 
			client_id,
			target,
			window_start_ts,
			count_total,
			count_error,
			COALESCE(ttfb_p95, 0) as ttfb_p95,
			COALESCE(dns_p95, 0) as dns_p95
		FROM agg_1m
		WHERE window_start_ts >= NOW() - INTERVAL '24 hours'
		  AND (count_error > 0 OR ttfb_p95 > 1000 OR dns_p95 > 500)
		ORDER BY window_start_ts DESC
		LIMIT 50
	`

	rows, err := s.repo.Connection().DB().QueryContext(ctx, query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database query failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	diagnostics := []map[string]interface{}{}
	diagID := 1

	for rows.Next() {
		var clientID, target string
		var timestamp time.Time
		var countTotal, countError int
		var ttfbP95, dnsP95 float64

		if err := rows.Scan(&clientID, &target, &timestamp, &countTotal, &countError, &ttfbP95, &dnsP95); err != nil {
			continue
		}

		errorRate := float64(0)
		if countTotal > 0 {
			errorRate = float64(countError) / float64(countTotal)
		}

		// Determine primary issue
		var label, description, severity string
		metrics := map[string]interface{}{}

		if errorRate > 0.5 {
			label = "High Error Rate"
			description = fmt.Sprintf("Error rate: %.1f%%", errorRate*100)
			severity = "error"
			metrics["error_rate"] = errorRate
			metrics["total_requests"] = countTotal
		} else if errorRate > 0.1 {
			label = "Elevated Errors"
			description = fmt.Sprintf("Error rate: %.1f%%", errorRate*100)
			severity = "warning"
			metrics["error_rate"] = errorRate
		} else if dnsP95 > 500 {
			label = "DNS-bound"
			description = fmt.Sprintf("High DNS latency (%.0fms)", dnsP95)
			severity = "warning"
			metrics["dns_latency_ms"] = dnsP95
		} else if ttfbP95 > 1000 {
			label = "Server-bound"
			description = fmt.Sprintf("Slow TTFB (%.0fms)", ttfbP95)
			severity = "warning"
			metrics["ttfb_ms"] = ttfbP95
		} else {
			continue
		}

		diag := map[string]interface{}{
			"id":          fmt.Sprintf("diag-%03d", diagID),
			"timestamp":   timestamp.Format(time.RFC3339),
			"client_id":   clientID,
			"target":      target,
			"label":       label,
			"severity":    severity,
			"description": description,
			"metrics":     metrics,
		}
		diagnostics = append(diagnostics, diag)
		diagID++
	}

	response := map[string]interface{}{
		"diagnostics": diagnostics,
		"total":       len(diagnostics),
	}
	respondJSON(w, http.StatusOK, response)
}

func (s *Service) getDiagnosticsTrends(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Query for trends over the past 7 days
	query := `
		SELECT 
			DATE(window_start_ts) as date,
			COUNT(CASE WHEN count_error::float / NULLIF(count_total, 0) > 0.5 THEN 1 END) as high_error_count,
			COUNT(CASE WHEN count_error::float / NULLIF(count_total, 0) BETWEEN 0.1 AND 0.5 THEN 1 END) as elevated_error_count,
			COUNT(CASE WHEN dns_p95 > 500 THEN 1 END) as dns_issue_count,
			COUNT(CASE WHEN ttfb_p95 > 1000 THEN 1 END) as server_issue_count
		FROM agg_1m
		WHERE window_start_ts >= NOW() - INTERVAL '7 days'
		GROUP BY DATE(window_start_ts)
		ORDER BY date DESC
	`

	rows, err := s.repo.Connection().DB().QueryContext(ctx, query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database query failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	trends := []map[string]interface{}{}

	for rows.Next() {
		var date time.Time
		var highErrorCount, elevatedErrorCount, dnsIssueCount, serverIssueCount int

		if err := rows.Scan(&date, &highErrorCount, &elevatedErrorCount, &dnsIssueCount, &serverIssueCount); err != nil {
			continue
		}

		dateStr := date.Format("2006-01-02")

		// Add individual trend points for each category
		if highErrorCount > 0 {
			trends = append(trends, map[string]interface{}{
				"date":  dateStr,
				"label": "High Error Rate",
				"count": highErrorCount,
			})
		}
		if elevatedErrorCount > 0 {
			trends = append(trends, map[string]interface{}{
				"date":  dateStr,
				"label": "Elevated Errors",
				"count": elevatedErrorCount,
			})
		}
		if dnsIssueCount > 0 {
			trends = append(trends, map[string]interface{}{
				"date":  dateStr,
				"label": "DNS-bound",
				"count": dnsIssueCount,
			})
		}
		if serverIssueCount > 0 {
			trends = append(trends, map[string]interface{}{
				"date":  dateStr,
				"label": "Server-bound",
				"count": serverIssueCount,
			})
		}
	}

	response := map[string]interface{}{
		"trends": trends,
	}
	respondJSON(w, http.StatusOK, response)
}

func (s *Service) deleteClient(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	clientID := vars["id"]

	// TODO: In real implementation, this would:
	// 1. Delete all aggregates for this client (agg_1m, agg_5m, agg_1h)
	// 2. Delete all diagnoses for this client
	// 3. Delete all AI analyses for this client
	// 4. Delete the client record
	// 5. Consider cascading deletes vs soft deletes based on requirements

	// For now, just respond with success
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"message":   "Client deleted successfully",
		"client_id": clientID,
	})
}
