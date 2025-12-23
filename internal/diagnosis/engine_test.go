package diagnosis

import (
	"math"
	"testing"
	"time"
)

func TestCalculateBaseline(t *testing.T) {
	tests := []struct {
		name     string
		windows  []WindowMetrics
		expected *Baseline
	}{
		{
			name:     "empty windows",
			windows:  []WindowMetrics{},
			expected: nil,
		},
		{
			name: "single window",
			windows: []WindowMetrics{
				{
					DNSP95:          100,
					TCPP95:          50,
					TLSP95:          30,
					TTFBP95:         120,
					TotalLatencyP95: 300,
					ThroughputP50:   1000,
					CountSuccess:    10,
				},
			},
			expected: &Baseline{
				DNSP95Avg:          100,
				TCPP95Avg:          50,
				TLSP95Avg:          30,
				TTFBP95Avg:         120,
				TotalLatencyP95Avg: 300,
				ThroughputP50Avg:   1000,
				WindowCount:        1,
			},
		},
		{
			name: "multiple windows",
			windows: []WindowMetrics{
				{DNSP95: 100, TCPP95: 50, TLSP95: 30, TTFBP95: 120, TotalLatencyP95: 300, ThroughputP50: 1000, CountSuccess: 10},
				{DNSP95: 120, TCPP95: 60, TLSP95: 40, TTFBP95: 140, TotalLatencyP95: 360, ThroughputP50: 1100, CountSuccess: 12},
				{DNSP95: 110, TCPP95: 55, TLSP95: 35, TTFBP95: 130, TotalLatencyP95: 330, ThroughputP50: 1050, CountSuccess: 11},
			},
			expected: &Baseline{
				DNSP95Avg:          110,
				TCPP95Avg:          55,
				TLSP95Avg:          35,
				TTFBP95Avg:         130,
				TotalLatencyP95Avg: 330,
				ThroughputP50Avg:   1050,
				WindowCount:        3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateBaseline(tt.windows)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil baseline, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatalf("Expected baseline, got nil")
			}

			if result.WindowCount != tt.expected.WindowCount {
				t.Errorf("WindowCount: expected %d, got %d", tt.expected.WindowCount, result.WindowCount)
			}

			const tolerance = 0.01
			if diff := abs(result.DNSP95Avg - tt.expected.DNSP95Avg); diff > tolerance {
				t.Errorf("DNSP95Avg: expected %.2f, got %.2f", tt.expected.DNSP95Avg, result.DNSP95Avg)
			}
			if diff := abs(result.ThroughputP50Avg - tt.expected.ThroughputP50Avg); diff > tolerance {
				t.Errorf("ThroughputP50Avg: expected %.2f, got %.2f", tt.expected.ThroughputP50Avg, result.ThroughputP50Avg)
			}
		})
	}
}

func TestDiagnoseDNSBound(t *testing.T) {
	baseline := &Baseline{
		DNSP95Avg:          100,
		TotalLatencyP95Avg: 300,
	}

	tests := []struct {
		name     string
		current  WindowMetrics
		baseline *Baseline
		expected bool
	}{
		{
			name:     "nil baseline",
			current:  WindowMetrics{DNSP95: 200, TotalLatencyP95: 300},
			baseline: nil,
			expected: false,
		},
		{
			name:     "DNS is 70% and exceeds baseline by 60%",
			current:  WindowMetrics{DNSP95: 160, TotalLatencyP95: 220},
			baseline: baseline,
			expected: true,
		},
		{
			name:     "DNS is 50% - not enough",
			current:  WindowMetrics{DNSP95: 150, TotalLatencyP95: 300},
			baseline: baseline,
			expected: false,
		},
		{
			name:     "DNS is 70% but only 40% increase",
			current:  WindowMetrics{DNSP95: 140, TotalLatencyP95: 200},
			baseline: baseline,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DiagnoseDNSBound(tt.current, tt.baseline)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDiagnoseHandshakeBound(t *testing.T) {
	baseline := &Baseline{
		TCPP95Avg:    50,
		TLSP95Avg:    30,
		TCPP95StdDev: 5,
		TLSP95StdDev: 3,
	}

	tests := []struct {
		name     string
		current  WindowMetrics
		baseline *Baseline
		expected bool
	}{
		{
			name:     "nil baseline",
			current:  WindowMetrics{TCPP95: 100, TLSP95: 60},
			baseline: nil,
			expected: false,
		},
		{
			name:     "exceeds by 100%",
			current:  WindowMetrics{TCPP95: 100, TLSP95: 60},
			baseline: baseline,
			expected: true,
		},
		{
			name:     "exceeds by 2 sigma",
			current:  WindowMetrics{TCPP95: 65, TLSP95: 40},
			baseline: baseline,
			expected: true,
		},
		{
			name:     "within normal range",
			current:  WindowMetrics{TCPP95: 52, TLSP95: 32},
			baseline: baseline,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DiagnoseHandshakeBound(tt.current, tt.baseline)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDiagnoseServerBound(t *testing.T) {
	baseline := &Baseline{
		TTFBP95Avg:    100,
		TTFBP95StdDev: 10,
		TCPP95Avg:     50,
		TLSP95Avg:     30,
		TCPP95StdDev:  5,
		TLSP95StdDev:  3,
	}

	tests := []struct {
		name     string
		current  WindowMetrics
		baseline *Baseline
		expected bool
	}{
		{
			name:     "nil baseline",
			current:  WindowMetrics{TTFBP95: 150, TCPP95: 52, TLSP95: 32},
			baseline: nil,
			expected: false,
		},
		{
			name:     "TTFB high but handshake also high",
			current:  WindowMetrics{TTFBP95: 150, TCPP95: 100, TLSP95: 50},
			baseline: baseline,
			expected: false,
		},
		{
			name:     "TTFB high and handshake normal",
			current:  WindowMetrics{TTFBP95: 150, TCPP95: 52, TLSP95: 32},
			baseline: baseline,
			expected: true,
		},
		{
			name:     "TTFB within range",
			current:  WindowMetrics{TTFBP95: 110, TCPP95: 52, TLSP95: 32},
			baseline: baseline,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DiagnoseServerBound(tt.current, tt.baseline)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDiagnoseThroughputBound(t *testing.T) {
	baseline := &Baseline{
		ThroughputP50Avg: 1000,
	}

	tests := []struct {
		name     string
		current  WindowMetrics
		baseline *Baseline
		expected bool
	}{
		{
			name:     "nil baseline",
			current:  WindowMetrics{ThroughputP50: 600},
			baseline: nil,
			expected: false,
		},
		{
			name:     "throughput dropped by 35%",
			current:  WindowMetrics{ThroughputP50: 650},
			baseline: baseline,
			expected: true,
		},
		{
			name:     "throughput dropped by exactly 30%",
			current:  WindowMetrics{ThroughputP50: 700},
			baseline: baseline,
			expected: true,
		},
		{
			name:     "throughput dropped by 20%",
			current:  WindowMetrics{ThroughputP50: 800},
			baseline: baseline,
			expected: false,
		},
		{
			name:     "throughput increased",
			current:  WindowMetrics{ThroughputP50: 1100},
			baseline: baseline,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DiagnoseThroughputBound(tt.current, tt.baseline)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDiagnose(t *testing.T) {
	now := time.Now()
	baseline := &Baseline{
		DNSP95Avg:          100,
		TotalLatencyP95Avg: 300,
		TCPP95Avg:          50,
		TLSP95Avg:          30,
		TCPP95StdDev:       5,
		TLSP95StdDev:       3,
		TTFBP95Avg:         100,
		TTFBP95StdDev:      10,
		ThroughputP50Avg:   1000,
	}

	tests := []struct {
		name     string
		current  WindowMetrics
		baseline *Baseline
		expected DiagnosisLabel
	}{
		{
			name:     "nil baseline",
			current:  WindowMetrics{CountSuccess: 10},
			baseline: nil,
			expected: DiagnosisNone,
		},
		{
			name:     "insufficient data",
			current:  WindowMetrics{CountSuccess: 3},
			baseline: baseline,
			expected: DiagnosisNone,
		},
		{
			name: "DNS-bound issue",
			current: WindowMetrics{
				WindowStartTs:   now,
				DNSP95:          160,
				TotalLatencyP95: 220,
				TCPP95:          52,
				TLSP95:          32,
				TTFBP95:         110,
				ThroughputP50:   900,
				CountSuccess:    10,
			},
			baseline: baseline,
			expected: DiagnosisDNSBound,
		},
		{
			name: "Handshake-bound issue",
			current: WindowMetrics{
				WindowStartTs:   now,
				DNSP95:          105,
				TotalLatencyP95: 300,
				TCPP95:          100,
				TLSP95:          60,
				TTFBP95:         110,
				ThroughputP50:   900,
				CountSuccess:    10,
			},
			baseline: baseline,
			expected: DiagnosisHandshake,
		},
		{
			name: "Server-bound issue",
			current: WindowMetrics{
				WindowStartTs:   now,
				DNSP95:          105,
				TotalLatencyP95: 300,
				TCPP95:          52,
				TLSP95:          32,
				TTFBP95:         150,
				ThroughputP50:   900,
				CountSuccess:    10,
			},
			baseline: baseline,
			expected: DiagnosisServerBound,
		},
		{
			name: "Throughput-bound issue",
			current: WindowMetrics{
				WindowStartTs:   now,
				DNSP95:          105,
				TotalLatencyP95: 300,
				TCPP95:          52,
				TLSP95:          32,
				TTFBP95:         110,
				ThroughputP50:   650,
				CountSuccess:    10,
			},
			baseline: baseline,
			expected: DiagnosisThroughput,
		},
		{
			name: "No issues detected",
			current: WindowMetrics{
				WindowStartTs:   now,
				DNSP95:          105,
				TotalLatencyP95: 300,
				TCPP95:          52,
				TLSP95:          32,
				TTFBP95:         110,
				ThroughputP50:   950,
				CountSuccess:    10,
			},
			baseline: baseline,
			expected: DiagnosisNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Diagnose(tt.current, tt.baseline)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Property 9: Baseline calculation always produces valid averages
// For any set of windows, baseline averages should be within the range of input values
func TestProperty_BaselineAveragesInRange(t *testing.T) {
	tests := []struct {
		name    string
		windows []WindowMetrics
	}{
		{
			name: "uniform values",
			windows: []WindowMetrics{
				{DNSP95: 100, TCPP95: 50, TLSP95: 30, TTFBP95: 120, TotalLatencyP95: 300, ThroughputP50: 1000, CountSuccess: 10},
				{DNSP95: 100, TCPP95: 50, TLSP95: 30, TTFBP95: 120, TotalLatencyP95: 300, ThroughputP50: 1000, CountSuccess: 10},
				{DNSP95: 100, TCPP95: 50, TLSP95: 30, TTFBP95: 120, TotalLatencyP95: 300, ThroughputP50: 1000, CountSuccess: 10},
			},
		},
		{
			name: "increasing values",
			windows: []WindowMetrics{
				{DNSP95: 100, TCPP95: 50, TLSP95: 30, TTFBP95: 120, TotalLatencyP95: 300, ThroughputP50: 1000, CountSuccess: 10},
				{DNSP95: 110, TCPP95: 55, TLSP95: 35, TTFBP95: 130, TotalLatencyP95: 330, ThroughputP50: 1050, CountSuccess: 11},
				{DNSP95: 120, TCPP95: 60, TLSP95: 40, TTFBP95: 140, TotalLatencyP95: 360, ThroughputP50: 1100, CountSuccess: 12},
			},
		},
		{
			name: "decreasing values",
			windows: []WindowMetrics{
				{DNSP95: 120, TCPP95: 60, TLSP95: 40, TTFBP95: 140, TotalLatencyP95: 360, ThroughputP50: 1100, CountSuccess: 12},
				{DNSP95: 110, TCPP95: 55, TLSP95: 35, TTFBP95: 130, TotalLatencyP95: 330, ThroughputP50: 1050, CountSuccess: 11},
				{DNSP95: 100, TCPP95: 50, TLSP95: 30, TTFBP95: 120, TotalLatencyP95: 300, ThroughputP50: 1000, CountSuccess: 10},
			},
		},
		{
			name: "mixed values",
			windows: []WindowMetrics{
				{DNSP95: 95, TCPP95: 48, TLSP95: 28, TTFBP95: 115, TotalLatencyP95: 286, ThroughputP50: 980, CountSuccess: 9},
				{DNSP95: 125, TCPP95: 62, TLSP95: 42, TTFBP95: 145, TotalLatencyP95: 374, ThroughputP50: 1120, CountSuccess: 13},
				{DNSP95: 105, TCPP95: 52, TLSP95: 32, TTFBP95: 125, TotalLatencyP95: 314, ThroughputP50: 1020, CountSuccess: 10},
				{DNSP95: 115, TCPP95: 58, TLSP95: 38, TTFBP95: 135, TotalLatencyP95: 346, ThroughputP50: 1080, CountSuccess: 12},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseline := CalculateBaseline(tt.windows)
			if baseline == nil {
				t.Fatal("Expected non-nil baseline")
			}

			// Find min/max for each metric
			minDNS, maxDNS := tt.windows[0].DNSP95, tt.windows[0].DNSP95
			minTCP, maxTCP := tt.windows[0].TCPP95, tt.windows[0].TCPP95
			minTLS, maxTLS := tt.windows[0].TLSP95, tt.windows[0].TLSP95
			minTTFB, maxTTFB := tt.windows[0].TTFBP95, tt.windows[0].TTFBP95
			minThroughput, maxThroughput := tt.windows[0].ThroughputP50, tt.windows[0].ThroughputP50

			for _, w := range tt.windows {
				if w.DNSP95 < minDNS {
					minDNS = w.DNSP95
				}
				if w.DNSP95 > maxDNS {
					maxDNS = w.DNSP95
				}
				if w.TCPP95 < minTCP {
					minTCP = w.TCPP95
				}
				if w.TCPP95 > maxTCP {
					maxTCP = w.TCPP95
				}
				if w.TLSP95 < minTLS {
					minTLS = w.TLSP95
				}
				if w.TLSP95 > maxTLS {
					maxTLS = w.TLSP95
				}
				if w.TTFBP95 < minTTFB {
					minTTFB = w.TTFBP95
				}
				if w.TTFBP95 > maxTTFB {
					maxTTFB = w.TTFBP95
				}
				if w.ThroughputP50 < minThroughput {
					minThroughput = w.ThroughputP50
				}
				if w.ThroughputP50 > maxThroughput {
					maxThroughput = w.ThroughputP50
				}
			}

			// Property: Averages must be within [min, max] range
			if baseline.DNSP95Avg < minDNS || baseline.DNSP95Avg > maxDNS {
				t.Errorf("DNSP95Avg %.2f not in range [%.2f, %.2f]", baseline.DNSP95Avg, minDNS, maxDNS)
			}
			if baseline.TCPP95Avg < minTCP || baseline.TCPP95Avg > maxTCP {
				t.Errorf("TCPP95Avg %.2f not in range [%.2f, %.2f]", baseline.TCPP95Avg, minTCP, maxTCP)
			}
			if baseline.TLSP95Avg < minTLS || baseline.TLSP95Avg > maxTLS {
				t.Errorf("TLSP95Avg %.2f not in range [%.2f, %.2f]", baseline.TLSP95Avg, minTLS, maxTLS)
			}
			if baseline.TTFBP95Avg < minTTFB || baseline.TTFBP95Avg > maxTTFB {
				t.Errorf("TTFBP95Avg %.2f not in range [%.2f, %.2f]", baseline.TTFBP95Avg, minTTFB, maxTTFB)
			}
			if baseline.ThroughputP50Avg < minThroughput || baseline.ThroughputP50Avg > maxThroughput {
				t.Errorf("ThroughputP50Avg %.2f not in range [%.2f, %.2f]", baseline.ThroughputP50Avg, minThroughput, maxThroughput)
			}

			// Property: Standard deviations must be non-negative
			if baseline.DNSP95StdDev < 0 {
				t.Errorf("DNSP95StdDev must be non-negative, got %.2f", baseline.DNSP95StdDev)
			}
			if baseline.TCPP95StdDev < 0 {
				t.Errorf("TCPP95StdDev must be non-negative, got %.2f", baseline.TCPP95StdDev)
			}
			if baseline.TLSP95StdDev < 0 {
				t.Errorf("TLSP95StdDev must be non-negative, got %.2f", baseline.TLSP95StdDev)
			}
		})
	}
}

// Property 10: DNS-bound diagnosis is monotonic with DNS latency
// If DNS increases while other metrics stay constant, diagnosis should remain DNS-bound or become DNS-bound
func TestProperty_DNSBoundMonotonic(t *testing.T) {
	baseline := &Baseline{
		DNSP95Avg:          100,
		TotalLatencyP95Avg: 300,
	}

	// Start with DNS at 70% and 60% increase
	current1 := WindowMetrics{
		DNSP95:          160,
		TCPP95:          30,
		TLSP95:          20,
		TTFBP95:         10,
		TotalLatencyP95: 220,
		CountSuccess:    10,
	}

	// Increase DNS further
	current2 := WindowMetrics{
		DNSP95:          180,
		TCPP95:          30,
		TLSP95:          20,
		TTFBP95:         10,
		TotalLatencyP95: 240,
		CountSuccess:    10,
	}

	result1 := DiagnoseDNSBound(current1, baseline)
	result2 := DiagnoseDNSBound(current2, baseline)

	// Property: If DNS-bound at lower DNS, must be DNS-bound at higher DNS
	if result1 && !result2 {
		t.Errorf("DNS-bound diagnosis not monotonic: true at DNS=160, false at DNS=180")
	}
}

// Property 11: Handshake-bound diagnosis respects 2σ threshold
// If handshake latency is exactly at threshold, behavior is well-defined
func TestProperty_HandshakeBoundThreshold(t *testing.T) {
	baseline := &Baseline{
		TCPP95Avg:    50,
		TLSP95Avg:    30,
		TCPP95StdDev: 5,
		TLSP95StdDev: 3,
	}

	// Calculate exact threshold (baseline + 2σ)
	// Note: standard deviations combine using sqrt(σ1² + σ2²)
	baselineHandshake := baseline.TCPP95Avg + baseline.TLSP95Avg // 80
	stdDev := math.Sqrt(baseline.TCPP95StdDev*baseline.TCPP95StdDev +
		baseline.TLSP95StdDev*baseline.TLSP95StdDev) // sqrt(25 + 9) = sqrt(34) ≈ 5.83
	twoSigmaThreshold := baselineHandshake + 2*stdDev // 80 + 11.66 = 91.66

	tests := []struct {
		name         string
		handshakeP95 float64
		expectBound  bool
	}{
		{
			name:         "well below threshold",
			handshakeP95: twoSigmaThreshold - 10,
			expectBound:  false,
		},
		{
			name:         "just below threshold",
			handshakeP95: twoSigmaThreshold - 0.1,
			expectBound:  false,
		},
		{
			name:         "at threshold",
			handshakeP95: twoSigmaThreshold,
			expectBound:  false, // <= not >, so false
		},
		{
			name:         "just above threshold",
			handshakeP95: twoSigmaThreshold + 0.1,
			expectBound:  true,
		},
		{
			name:         "100% increase",
			handshakeP95: baselineHandshake * 2,
			expectBound:  true,
		},
		{
			name: "50% increase but under 2σ",
			// At 50% increase: 80 * 1.5 = 120, which is > 91.66 (2σ), so should be true
			handshakeP95: baselineHandshake * 1.5,
			expectBound:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current := WindowMetrics{
				TCPP95:       tt.handshakeP95 * 0.625, // 50/80 ratio from baseline
				TLSP95:       tt.handshakeP95 * 0.375, // 30/80 ratio from baseline
				CountSuccess: 10,
			}

			result := DiagnoseHandshakeBound(current, baseline)
			if result != tt.expectBound {
				t.Errorf("Expected %v for handshake=%.2f (2σ threshold=%.2f, 100%% threshold=%.2f), got %v",
					tt.expectBound, tt.handshakeP95, twoSigmaThreshold, baselineHandshake*2, result)
			}
		})
	}
}

// Property 12: Server-bound diagnosis only when connections are normal
// If both TTFB and handshake are high, should NOT be server-bound
func TestProperty_ServerBoundRequiresNormalConnections(t *testing.T) {
	baseline := &Baseline{
		TTFBP95Avg:    100,
		TTFBP95StdDev: 10,
		TCPP95Avg:     50,
		TLSP95Avg:     30,
		TCPP95StdDev:  5,
		TLSP95StdDev:  3,
	}

	// High TTFB (exceeds 2σ)
	highTTFB := baseline.TTFBP95Avg + 2*baseline.TTFBP95StdDev + 10 // 130

	// Calculate connection threshold (1σ)
	baselineHandshake := baseline.TCPP95Avg + baseline.TLSP95Avg // 80
	handshakeStdDev := math.Sqrt(baseline.TCPP95StdDev*baseline.TCPP95StdDev +
		baseline.TLSP95StdDev*baseline.TLSP95StdDev) // sqrt(34) ≈ 5.83
	connectionThreshold := baselineHandshake + handshakeStdDev // 80 + 5.83 = 85.83

	tests := []struct {
		name        string
		tcpP95      float64
		tlsP95      float64
		expectBound bool
		description string
	}{
		{
			name:        "well below threshold",
			tcpP95:      52,
			tlsP95:      32,
			expectBound: true,
			description: "connections well within normal range",
		},
		{
			name:        "high connections",
			tcpP95:      100,
			tlsP95:      60,
			expectBound: false,
			description: "connections also elevated",
		},
		{
			name:        "just below threshold",
			tcpP95:      baseline.TCPP95Avg + handshakeStdDev*0.6,
			tlsP95:      baseline.TLSP95Avg + handshakeStdDev*0.4,
			expectBound: true,
			description: "connections just below 1σ threshold",
		},
		{
			name:        "just above threshold",
			tcpP95:      baseline.TCPP95Avg + handshakeStdDev*0.65,
			tlsP95:      baseline.TLSP95Avg + handshakeStdDev*0.45,
			expectBound: false,
			description: "connections just above 1σ threshold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current := WindowMetrics{
				TTFBP95:      highTTFB,
				TCPP95:       tt.tcpP95,
				TLSP95:       tt.tlsP95,
				CountSuccess: 10,
			}

			result := DiagnoseServerBound(current, baseline)
			totalHandshake := tt.tcpP95 + tt.tlsP95
			if result != tt.expectBound {
				t.Errorf("%s: Expected %v, got %v (handshake=%.2f, threshold=%.2f)",
					tt.description, tt.expectBound, result, totalHandshake, connectionThreshold)
			}
		})
	}
}

// Property 13: Throughput-bound diagnosis is monotonic with throughput decrease
// As throughput decreases, diagnosis should remain throughput-bound or become throughput-bound
func TestProperty_ThroughputBoundMonotonic(t *testing.T) {
	baseline := &Baseline{
		ThroughputP50Avg: 1000,
	}

	tests := []struct {
		name        string
		throughput  float64
		expectBound bool
	}{
		{
			name:        "20% decrease - not bound",
			throughput:  800,
			expectBound: false,
		},
		{
			name:        "30% decrease - exactly at threshold",
			throughput:  700,
			expectBound: true,
		},
		{
			name:        "40% decrease - bound",
			throughput:  600,
			expectBound: true,
		},
		{
			name:        "50% decrease - still bound",
			throughput:  500,
			expectBound: true,
		},
	}

	prevBound := false
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current := WindowMetrics{
				ThroughputP50: tt.throughput,
				CountSuccess:  10,
			}

			result := DiagnoseThroughputBound(current, baseline)
			if result != tt.expectBound {
				t.Errorf("Expected %v at throughput=%.0f, got %v", tt.expectBound, tt.throughput, result)
			}

			// Property: Monotonicity - if bound at higher throughput, must be bound at lower
			if prevBound && !result && tt.throughput < 700 {
				t.Errorf("Throughput diagnosis not monotonic: was bound, now not bound at lower throughput")
			}
			prevBound = result
		})
	}
}

// Property 14: Diagnosis priority order is consistent
// DNS > Handshake > Server > Throughput
func TestProperty_DiagnosisPriorityOrder(t *testing.T) {
	baseline := &Baseline{
		DNSP95Avg:          100,
		TotalLatencyP95Avg: 300,
		TCPP95Avg:          50,
		TLSP95Avg:          30,
		TCPP95StdDev:       5,
		TLSP95StdDev:       3,
		TTFBP95Avg:         100,
		TTFBP95StdDev:      10,
		ThroughputP50Avg:   1000,
	}

	// Create a window that triggers ALL diagnosis conditions
	current := WindowMetrics{
		WindowStartTs:   time.Now(),
		DNSP95:          160, // DNS-bound: 70% of 220 and 60% increase
		TCPP95:          100, // Handshake-bound: 100% increase
		TLSP95:          60,  // Handshake-bound: 100% increase
		TTFBP95:         150, // Server-bound: exceeds 2σ
		TotalLatencyP95: 220, // Total for DNS ratio calculation
		ThroughputP50:   650, // Throughput-bound: 35% decrease
		CountSuccess:    10,
	}

	// Verify all individual conditions are true
	if !DiagnoseDNSBound(current, baseline) {
		t.Error("Expected DNS-bound to be true")
	}
	if !DiagnoseHandshakeBound(current, baseline) {
		t.Error("Expected Handshake-bound to be true")
	}
	// Note: Server-bound requires connections to be normal, so it won't be true when handshake is high
	if !DiagnoseThroughputBound(current, baseline) {
		t.Error("Expected Throughput-bound to be true")
	}

	// Property: When multiple conditions are true, DNS has highest priority
	result := Diagnose(current, baseline)
	if result != DiagnosisDNSBound {
		t.Errorf("Expected DNS-bound due to priority, got %v", result)
	}

	// Test Handshake priority over Throughput
	current2 := WindowMetrics{
		WindowStartTs:   time.Now(),
		DNSP95:          105, // Normal DNS
		TCPP95:          100, // Handshake-bound
		TLSP95:          60,  // Handshake-bound
		TTFBP95:         110, // Normal TTFB
		TotalLatencyP95: 375,
		ThroughputP50:   650, // Throughput-bound
		CountSuccess:    10,
	}

	result2 := Diagnose(current2, baseline)
	if result2 != DiagnosisHandshake {
		t.Errorf("Expected Handshake-bound due to priority, got %v", result2)
	}
}
