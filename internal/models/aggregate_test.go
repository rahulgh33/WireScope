package models

import (
	"testing"
	"time"
)

func TestCalculatePercentile(t *testing.T) {
	tests := []struct {
		name       string
		data       []float64
		percentile float64
		want       float64
	}{
		{
			name:       "empty data",
			data:       []float64{},
			percentile: 50,
			want:       0,
		},
		{
			name:       "single value",
			data:       []float64{42.0},
			percentile: 50,
			want:       42.0,
		},
		{
			name:       "P50 of sorted data",
			data:       []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			percentile: 50,
			want:       5.5, // 0.5 * (10-1) = 4.5, interpolate between index 4 (value 5) and 5 (value 6) = 5.5
		},
		{
			name:       "P95 of sorted data",
			data:       []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			percentile: 95,
			want:       9.55, // 0.95 * (10-1) = 8.55, interpolate between index 8 (value 9) and 9 (value 10)
		},
		{
			name:       "P50 of unsorted data",
			data:       []float64{10, 1, 5, 3, 8, 2, 7, 4, 9, 6},
			percentile: 50,
			want:       5.5, // Same data as sorted test
		},
		{
			name:       "P95 with duplicates",
			data:       []float64{5, 5, 5, 5, 5, 10, 10, 10, 10, 10},
			percentile: 95,
			want:       10.0,
		},
		{
			name:       "P99 of 100 values",
			data:       make100Values(),
			percentile: 99,
			want:       98.01, // 0.99 * (100-1) = 98.01, interpolate between index 98 (value 98) and 99 (value 99)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculatePercentile(tt.data, tt.percentile)
			// Use tolerance for floating point comparison
			tolerance := 0.01
			if diff := got - tt.want; diff < -tolerance || diff > tolerance {
				t.Errorf("calculatePercentile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDownsampleUniform(t *testing.T) {
	tests := []struct {
		name       string
		data       []float64
		targetSize int
		wantLen    int
	}{
		{
			name:       "no downsampling needed",
			data:       []float64{1, 2, 3, 4, 5},
			targetSize: 10,
			wantLen:    5,
		},
		{
			name:       "downsample to half",
			data:       []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			targetSize: 5,
			wantLen:    5,
		},
		{
			name:       "downsample large dataset",
			data:       make100Values(),
			targetSize: 10,
			wantLen:    10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := downsampleUniform(tt.data, tt.targetSize)
			if len(got) != tt.wantLen {
				t.Errorf("downsampleUniform() length = %v, want %v", len(got), tt.wantLen)
			}
		})
	}
}

func TestInMemoryAggregator(t *testing.T) {
	key := AggregateKey{
		ClientID:      "test-client",
		Target:        "https://example.com",
		WindowStartTs: parseTime("2024-01-01T00:00:00Z"),
	}

	agg := NewInMemoryAggregator(key)

	// Add successful event
	event1 := &TelemetryEvent{
		EventID:     "event-1",
		ClientID:    "test-client",
		TimestampMs: 1704067200000, // 2024-01-01T00:00:00Z
		Target:      "https://example.com",
		Timings: TimingMeasurements{
			DNSMs:      10.0,
			TCPMs:      20.0,
			TLSMs:      30.0,
			HTTPTTFBMs: 40.0,
		},
		ThroughputKbps: 5000.0,
	}
	agg.AddEvent(event1)

	if agg.CountTotal != 1 {
		t.Errorf("CountTotal = %v, want 1", agg.CountTotal)
	}
	if agg.CountSuccess != 1 {
		t.Errorf("CountSuccess = %v, want 1", agg.CountSuccess)
	}
	if agg.CountError != 0 {
		t.Errorf("CountError = %v, want 0", agg.CountError)
	}

	// Add error event
	errorStage := "DNS"
	event2 := &TelemetryEvent{
		EventID:     "event-2",
		ClientID:    "test-client",
		TimestampMs: 1704067210000,
		Target:      "https://example.com",
		ErrorStage:  &errorStage,
	}
	agg.AddEvent(event2)

	if agg.CountTotal != 2 {
		t.Errorf("CountTotal = %v, want 2", agg.CountTotal)
	}
	if agg.CountSuccess != 1 {
		t.Errorf("CountSuccess = %v, want 1", agg.CountSuccess)
	}
	if agg.CountError != 1 {
		t.Errorf("CountError = %v, want 1", agg.CountError)
	}
	if agg.ErrorStageCounts["DNS"] != 1 {
		t.Errorf("ErrorStageCounts[DNS] = %v, want 1", agg.ErrorStageCounts["DNS"])
	}

	// Convert to WindowedAggregate
	wa := agg.ToWindowedAggregate()
	if wa.CountTotal != 2 {
		t.Errorf("WindowedAggregate.CountTotal = %v, want 2", wa.CountTotal)
	}
	if wa.DNSP50 != 10.0 {
		t.Errorf("WindowedAggregate.DNSP50 = %v, want 10.0", wa.DNSP50)
	}
}

func TestWindowedAggregateRates(t *testing.T) {
	wa := &WindowedAggregate{
		CountTotal:   100,
		CountSuccess: 95,
		CountError:   5,
	}

	successRate := wa.SuccessRate()
	if successRate != 0.95 {
		t.Errorf("SuccessRate() = %v, want 0.95", successRate)
	}

	errorRate := wa.ErrorRate()
	if errorRate != 0.05 {
		t.Errorf("ErrorRate() = %v, want 0.05", errorRate)
	}
}

func TestWindowedAggregateWithZeroCount(t *testing.T) {
	wa := &WindowedAggregate{
		CountTotal:   0,
		CountSuccess: 0,
		CountError:   0,
	}

	successRate := wa.SuccessRate()
	if successRate != 0 {
		t.Errorf("SuccessRate() = %v, want 0", successRate)
	}

	errorRate := wa.ErrorRate()
	if errorRate != 0 {
		t.Errorf("ErrorRate() = %v, want 0", errorRate)
	}
}

// Helper functions
func make100Values() []float64 {
	values := make([]float64, 100)
	for i := 0; i < 100; i++ {
		values[i] = float64(i)
	}
	return values
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}
