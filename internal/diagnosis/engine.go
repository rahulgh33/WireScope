package diagnosis

import (
	"math"
	"time"
)

// DiagnosisLabel represents the type of network performance issue detected
type DiagnosisLabel string

const (
	DiagnosisNone        DiagnosisLabel = ""
	DiagnosisDNSBound    DiagnosisLabel = "dns-bound"
	DiagnosisHandshake   DiagnosisLabel = "handshake-bound"
	DiagnosisServerBound DiagnosisLabel = "server-bound"
	DiagnosisThroughput  DiagnosisLabel = "throughput-bound"
)

// WindowMetrics represents metrics for a single time window
type WindowMetrics struct {
	WindowStartTs   time.Time
	DNSP95          float64
	TCPP95          float64
	TLSP95          float64
	TTFBP95         float64
	TotalLatencyP95 float64 // Sum of DNS + TCP + TLS + TTFB
	ThroughputP50   float64
	CountSuccess    int
}

// Baseline represents baseline metrics calculated from historical windows
type Baseline struct {
	DNSP95Avg          float64
	TCPP95Avg          float64
	TLSP95Avg          float64
	TTFBP95Avg         float64
	TotalLatencyP95Avg float64
	ThroughputP50Avg   float64

	// Standard deviations for statistical thresholds
	DNSP95StdDev          float64
	TCPP95StdDev          float64
	TLSP95StdDev          float64
	TTFBP95StdDev         float64
	TotalLatencyP95StdDev float64
	ThroughputP50StdDev   float64

	WindowCount int
}

// CalculateBaseline computes baseline metrics from historical windows
// Uses simple moving average over the last N windows
//
// Requirement: 5.1 - Baseline calculation using simple moving average
func CalculateBaseline(windows []WindowMetrics) *Baseline {
	if len(windows) == 0 {
		return nil
	}

	baseline := &Baseline{
		WindowCount: len(windows),
	}

	// Calculate sums for averages
	for _, w := range windows {
		baseline.DNSP95Avg += w.DNSP95
		baseline.TCPP95Avg += w.TCPP95
		baseline.TLSP95Avg += w.TLSP95
		baseline.TTFBP95Avg += w.TTFBP95
		baseline.TotalLatencyP95Avg += w.TotalLatencyP95
		baseline.ThroughputP50Avg += w.ThroughputP50
	}

	// Compute averages
	n := float64(len(windows))
	baseline.DNSP95Avg /= n
	baseline.TCPP95Avg /= n
	baseline.TLSP95Avg /= n
	baseline.TTFBP95Avg /= n
	baseline.TotalLatencyP95Avg /= n
	baseline.ThroughputP50Avg /= n

	// Calculate standard deviations (for 2σ thresholds)
	for _, w := range windows {
		baseline.DNSP95StdDev += math.Pow(w.DNSP95-baseline.DNSP95Avg, 2)
		baseline.TCPP95StdDev += math.Pow(w.TCPP95-baseline.TCPP95Avg, 2)
		baseline.TLSP95StdDev += math.Pow(w.TLSP95-baseline.TLSP95Avg, 2)
		baseline.TTFBP95StdDev += math.Pow(w.TTFBP95-baseline.TTFBP95Avg, 2)
		baseline.TotalLatencyP95StdDev += math.Pow(w.TotalLatencyP95-baseline.TotalLatencyP95Avg, 2)
		baseline.ThroughputP50StdDev += math.Pow(w.ThroughputP50-baseline.ThroughputP50Avg, 2)
	}

	baseline.DNSP95StdDev = math.Sqrt(baseline.DNSP95StdDev / n)
	baseline.TCPP95StdDev = math.Sqrt(baseline.TCPP95StdDev / n)
	baseline.TLSP95StdDev = math.Sqrt(baseline.TLSP95StdDev / n)
	baseline.TTFBP95StdDev = math.Sqrt(baseline.TTFBP95StdDev / n)
	baseline.TotalLatencyP95StdDev = math.Sqrt(baseline.TotalLatencyP95StdDev / n)
	baseline.ThroughputP50StdDev = math.Sqrt(baseline.ThroughputP50StdDev / n)

	return baseline
}

// DiagnoseDNSBound checks if DNS resolution is the primary bottleneck
//
// Criteria:
// - DNS p95 ≥ 60% of total latency p95
// - DNS p95 exceeds baseline by ≥ 50%
//
// Requirement: 5.2 - DNS-bound diagnosis
func DiagnoseDNSBound(current WindowMetrics, baseline *Baseline) bool {
	if baseline == nil || current.TotalLatencyP95 == 0 || baseline.DNSP95Avg == 0 {
		return false
	}

	// Check if DNS is ≥60% of total latency
	dnsRatio := current.DNSP95 / current.TotalLatencyP95
	if dnsRatio < 0.60 {
		return false
	}

	// Check if DNS exceeds baseline by ≥50%
	increase := (current.DNSP95 - baseline.DNSP95Avg) / baseline.DNSP95Avg
	return increase >= 0.50
}

// DiagnoseHandshakeBound checks if TCP/TLS handshake is the bottleneck
//
// Criteria:
// - TCP/TLS p95 exceeds baseline by 2σ OR 100%
//
// Requirement: 5.3 - Handshake-bound diagnosis
func DiagnoseHandshakeBound(current WindowMetrics, baseline *Baseline) bool {
	if baseline == nil || baseline.TCPP95Avg == 0 {
		return false
	}

	handshakeP95 := current.TCPP95 + current.TLSP95
	baselineHandshake := baseline.TCPP95Avg + baseline.TLSP95Avg
	baselineStdDev := math.Sqrt(baseline.TCPP95StdDev*baseline.TCPP95StdDev +
		baseline.TLSP95StdDev*baseline.TLSP95StdDev)

	// Check 2σ threshold
	twoSigmaThreshold := baselineHandshake + 2*baselineStdDev
	if handshakeP95 > twoSigmaThreshold {
		return true
	}

	// Check 100% increase threshold
	increase := (handshakeP95 - baselineHandshake) / baselineHandshake
	return increase >= 1.0
}

// DiagnoseServerBound checks if server processing (TTFB) is the bottleneck
//
// Criteria:
// - TTFB p95 exceeds baseline by 2σ
// - TCP/TLS connections are normal (not exceeding baseline significantly)
//
// Requirement: 5.4 - Server-bound diagnosis
func DiagnoseServerBound(current WindowMetrics, baseline *Baseline) bool {
	if baseline == nil || baseline.TTFBP95Avg == 0 {
		return false
	}

	// Check if TTFB exceeds baseline by 2σ
	twoSigmaThreshold := baseline.TTFBP95Avg + 2*baseline.TTFBP95StdDev
	if current.TTFBP95 <= twoSigmaThreshold {
		return false
	}

	// Verify connections are normal (not also experiencing handshake issues)
	handshakeP95 := current.TCPP95 + current.TLSP95
	baselineHandshake := baseline.TCPP95Avg + baseline.TLSP95Avg
	baselineHandshakeStdDev := math.Sqrt(baseline.TCPP95StdDev*baseline.TCPP95StdDev +
		baseline.TLSP95StdDev*baseline.TLSP95StdDev)

	// Connections are "normal" if within 1σ of baseline
	connectionThreshold := baselineHandshake + baselineHandshakeStdDev
	return handshakeP95 <= connectionThreshold
}

// DiagnoseThroughputBound checks if throughput is degraded
//
// Criteria:
// - Throughput p50 drops ≥30% below baseline
//
// Requirement: 5.5 - Throughput-bound diagnosis
func DiagnoseThroughputBound(current WindowMetrics, baseline *Baseline) bool {
	if baseline == nil || baseline.ThroughputP50Avg == 0 {
		return false
	}

	// Check if throughput dropped by ≥30%
	decrease := (baseline.ThroughputP50Avg - current.ThroughputP50) / baseline.ThroughputP50Avg
	return decrease >= 0.30
}

// Diagnose runs all diagnosis checks and returns the primary issue
// Priority order: DNS > Handshake > Server > Throughput
//
// Requirements: 5.1, 5.2, 5.3, 5.4, 5.5
func Diagnose(current WindowMetrics, baseline *Baseline) DiagnosisLabel {
	// Skip diagnosis if insufficient data
	if baseline == nil || current.CountSuccess < 5 {
		return DiagnosisNone
	}

	// Check in priority order
	if DiagnoseDNSBound(current, baseline) {
		return DiagnosisDNSBound
	}

	if DiagnoseHandshakeBound(current, baseline) {
		return DiagnosisHandshake
	}

	if DiagnoseServerBound(current, baseline) {
		return DiagnosisServerBound
	}

	if DiagnoseThroughputBound(current, baseline) {
		return DiagnosisThroughput
	}

	return DiagnosisNone
}
