package diagnosis

import (
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
		TTFBP95Avg:     100,
		TTFBP95StdDev:  10,
		TCPP95Avg:      50,
		TLSP95Avg:      30,
		TCPP95StdDev:   5,
		TLSP95StdDev:   3,
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
