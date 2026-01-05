package ai

import (
	"context"
	"fmt"

	"github.com/rahulgh33/wirescope/internal/database"
)

// DataAccessLayer provides optimized data access for AI agent queries
type DataAccessLayer struct {
	repo *database.Repository
}

// NewDataAccessLayer creates a new data access layer instance
func NewDataAccessLayer(repo *database.Repository) *DataAccessLayer {
	return &DataAccessLayer{
		repo: repo,
	}
}

// GetOverallMetrics returns high-level summary statistics
func (dal *DataAccessLayer) GetOverallMetrics(ctx context.Context, timeRange TimeRange) (*OverallMetrics, error) {
	query := `
		SELECT 
			COUNT(DISTINCT client_id) as total_clients,
			COUNT(DISTINCT CASE WHEN count_total > 0 THEN client_id END) as active_clients,
			COUNT(DISTINCT target) as total_targets,
			COALESCE(AVG(COALESCE(dns_p95, 0) + COALESCE(tcp_p95, 0) + COALESCE(tls_p95, 0) + COALESCE(ttfb_p95, 0)), 0) as avg_latency_p95,
			COALESCE(AVG(throughput_p50), 0) as avg_throughput_p50,
			COALESCE(SUM(count_total), 0) as total_measurements,
			COALESCE(SUM(count_success)::float / NULLIF(SUM(count_total), 0), 0) as success_rate,
			COALESCE(SUM(count_error)::float / NULLIF(SUM(count_total), 0), 0) as error_rate
		FROM agg_1m
		WHERE window_start_ts >= $1
			AND window_start_ts <= $2
	`

	var metrics OverallMetrics
	metrics.TimeRange = timeRange

	err := dal.repo.Connection().DB().QueryRowContext(ctx, query, timeRange.Start, timeRange.End).Scan(
		&metrics.TotalClients,
		&metrics.ActiveClients,
		&metrics.TotalTargets,
		&metrics.AvgLatencyP95,
		&metrics.AvgThroughputP50,
		&metrics.TotalMeasurements,
		&metrics.SuccessRate,
		&metrics.ErrorRate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get overall metrics: %w", err)
	}

	return &metrics, nil
}

// CompareClientPerformance compares performance across multiple clients
func (dal *DataAccessLayer) CompareClientPerformance(ctx context.Context, timeRange TimeRange, target *string, limit int) ([]ClientPerformance, error) {
	query := `
		WITH performance AS (
			SELECT 
				client_id,
				target,
				COALESCE(AVG(COALESCE(dns_p95, 0) + COALESCE(tcp_p95, 0) + COALESCE(tls_p95, 0) + COALESCE(ttfb_p95, 0)), 0) as avg_latency_p95,
				COALESCE(AVG(throughput_p50), 0) as avg_throughput_p50,
				COALESCE(SUM(count_error)::float / NULLIF(SUM(count_total), 0), 0) as error_rate,
				COALESCE(SUM(count_total), 0) as total_measurements
			FROM agg_1m
			WHERE window_start_ts >= $1
				AND window_start_ts <= $2
	`

	args := []interface{}{timeRange.Start, timeRange.End}
	argIdx := 3

	if target != nil {
		query += fmt.Sprintf(" AND target = $%d", argIdx)
		args = append(args, *target)
		argIdx++
	}

	query += `
			GROUP BY client_id, target
		),
		diagnoses AS (
			SELECT 
				client_id,
				target,
				diagnosis_label,
				COUNT(*) as label_count,
				ROW_NUMBER() OVER (PARTITION BY client_id, target ORDER BY COUNT(*) DESC) as rn
			FROM agg_1m
			WHERE window_start_ts >= $1
				AND window_start_ts <= $2
				AND diagnosis_label IS NOT NULL
	`

	if target != nil {
		query += fmt.Sprintf(" AND target = $%d", argIdx)
	}

	query += `
			GROUP BY client_id, target, diagnosis_label
		)
		SELECT 
			p.client_id,
			p.target,
			p.avg_latency_p95,
			p.avg_throughput_p50,
			COALESCE(p.error_rate, 0) as error_rate,
			p.total_measurements,
			d.diagnosis_label
		FROM performance p
		LEFT JOIN diagnoses d ON p.client_id = d.client_id AND p.target = d.target AND d.rn = 1
		ORDER BY p.avg_latency_p95 DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := dal.repo.Connection().DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to compare client performance: %w", err)
	}
	defer rows.Close()

	var results []ClientPerformance
	for rows.Next() {
		var perf ClientPerformance
		err := rows.Scan(
			&perf.ClientID,
			&perf.Target,
			&perf.AvgLatencyP95,
			&perf.AvgThroughputP50,
			&perf.ErrorRate,
			&perf.TotalMeasurements,
			&perf.PrimaryIssue,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan client performance row: %w", err)
		}
		results = append(results, perf)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating client performance rows: %w", err)
	}

	return results, nil
}

// ListClients returns all unique client IDs in the time range
func (dal *DataAccessLayer) ListClients(ctx context.Context, timeRange TimeRange) ([]string, error) {
	query := `
		SELECT DISTINCT client_id
		FROM agg_1m
		WHERE window_start_ts >= $1
			AND window_start_ts <= $2
		ORDER BY client_id
	`

	rows, err := dal.repo.Connection().DB().QueryContext(ctx, query, timeRange.Start, timeRange.End)
	if err != nil {
		return nil, fmt.Errorf("failed to list clients: %w", err)
	}
	defer rows.Close()

	var clients []string
	for rows.Next() {
		var clientID string
		if err := rows.Scan(&clientID); err != nil {
			return nil, fmt.Errorf("failed to scan client ID: %w", err)
		}
		clients = append(clients, clientID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating client rows: %w", err)
	}

	return clients, nil
}

// ListTargets returns all unique targets in the time range
func (dal *DataAccessLayer) ListTargets(ctx context.Context, timeRange TimeRange) ([]string, error) {
	query := `
		SELECT DISTINCT target
		FROM agg_1m
		WHERE window_start_ts >= $1
			AND window_start_ts <= $2
		ORDER BY target
	`

	rows, err := dal.repo.Connection().DB().QueryContext(ctx, query, timeRange.Start, timeRange.End)
	if err != nil {
		return nil, fmt.Errorf("failed to list targets: %w", err)
	}
	defer rows.Close()

	var targets []string
	for rows.Next() {
		var target string
		if err := rows.Scan(&target); err != nil {
			return nil, fmt.Errorf("failed to scan target: %w", err)
		}
		targets = append(targets, target)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating target rows: %w", err)
	}

	return targets, nil
}
