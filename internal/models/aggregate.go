package models

import (
	"time"
)

// WindowedAggregate represents time-windowed aggregated metrics for a specific
// client and target combination.
//
// Requirements: 4.1, 4.2, 4.4, 5.1, 5.2, 5.3, 5.4, 5.5
type WindowedAggregate struct {
	// ClientID identifies the probe agent
	ClientID string

	// Target is the endpoint being measured
	Target string

	// WindowStartTs is the start of the 1-minute aggregation window
	WindowStartTs time.Time

	// CountTotal is the total number of measurement events in this window
	CountTotal int64

	// CountSuccess is the number of successful measurements
	CountSuccess int64

	// CountError is the number of failed measurements
	CountError int64

	// ErrorStageCounts tracks errors by stage (DNS, TCP, TLS, HTTP, throughput)
	ErrorStageCounts map[string]int64

	// DNS timing percentiles (milliseconds)
	DNSP50 float64
	DNSP95 float64

	// TCP timing percentiles (milliseconds)
	TCPP50 float64
	TCPP95 float64

	// TLS timing percentiles (milliseconds)
	TLSP50 float64
	TLSP95 float64

	// HTTP TTFB percentiles (milliseconds)
	TTFBP50 float64
	TTFBP95 float64

	// Throughput percentiles (kilobits per second)
	ThroughputP50 float64
	ThroughputP95 float64

	// DiagnosisLabel indicates the identified performance bottleneck type
	// Possible values: "DNS-bound", "Handshake-bound", "Server-bound", "Throughput-bound"
	DiagnosisLabel *string

	// UpdatedAt is the last time this aggregate was updated
	UpdatedAt time.Time
}

// ErrorStage constants for tracking which stage errors occurred
const (
	ErrorStageDNS        = "DNS"
	ErrorStageTCP        = "TCP"
	ErrorStageTLS        = "TLS"
	ErrorStageHTTP       = "HTTP"
	ErrorStageThroughput = "throughput"
)

// DiagnosisLabel constants for bottleneck classification
const (
	DiagnosisDNSBound        = "DNS-bound"
	DiagnosisHandshakeBound  = "Handshake-bound"
	DiagnosisServerBound     = "Server-bound"
	DiagnosisThroughputBound = "Throughput-bound"
)

// NewWindowedAggregate creates a new WindowedAggregate with initialized fields.
func NewWindowedAggregate(clientID, target string, windowStartTs time.Time) *WindowedAggregate {
	return &WindowedAggregate{
		ClientID:         clientID,
		Target:           target,
		WindowStartTs:    windowStartTs,
		ErrorStageCounts: make(map[string]int64),
		UpdatedAt:        time.Now(),
	}
}

// AggregateKey uniquely identifies an aggregate window
type AggregateKey struct {
	ClientID      string
	Target        string
	WindowStartTs time.Time
}

// Key returns the AggregateKey for this WindowedAggregate
func (wa *WindowedAggregate) Key() AggregateKey {
	return AggregateKey{
		ClientID:      wa.ClientID,
		Target:        wa.Target,
		WindowStartTs: wa.WindowStartTs,
	}
}

// SuccessRate returns the success rate as a value between 0 and 1
func (wa *WindowedAggregate) SuccessRate() float64 {
	if wa.CountTotal == 0 {
		return 0
	}
	return float64(wa.CountSuccess) / float64(wa.CountTotal)
}

// ErrorRate returns the error rate as a value between 0 and 1
func (wa *WindowedAggregate) ErrorRate() float64 {
	if wa.CountTotal == 0 {
		return 0
	}
	return float64(wa.CountError) / float64(wa.CountTotal)
}

// GetTotalLatencyP95 returns the sum of all timing components at P95
func (wa *WindowedAggregate) GetTotalLatencyP95() float64 {
	return wa.DNSP95 + wa.TCPP95 + wa.TLSP95 + wa.TTFBP95
}

// InMemoryAggregator holds raw samples in memory for exact percentile calculation
// This is used during the aggregation window before flushing to the database.
//
// Requirement: 4.2 - MVP uses exact percentiles from full sample set
type InMemoryAggregator struct {
	Key AggregateKey

	// Raw samples for percentile calculation (up to 10,000 samples for MVP)
	DNSSamples        []float64
	TCPSamples        []float64
	TLSSamples        []float64
	TTFBSamples       []float64
	ThroughputSamples []float64

	// Counters
	CountTotal   int64
	CountSuccess int64
	CountError   int64

	// Error stage tracking
	ErrorStageCounts map[string]int64

	// Last update time
	UpdatedAt time.Time
}

// NewInMemoryAggregator creates a new in-memory aggregator for a window
func NewInMemoryAggregator(key AggregateKey) *InMemoryAggregator {
	return &InMemoryAggregator{
		Key:               key,
		DNSSamples:        make([]float64, 0, 100),
		TCPSamples:        make([]float64, 0, 100),
		TLSSamples:        make([]float64, 0, 100),
		TTFBSamples:       make([]float64, 0, 100),
		ThroughputSamples: make([]float64, 0, 100),
		ErrorStageCounts:  make(map[string]int64),
		UpdatedAt:         time.Now(),
	}
}

// AddEvent adds a telemetry event to the in-memory aggregator
func (ima *InMemoryAggregator) AddEvent(event *TelemetryEvent) {
	ima.CountTotal++

	if event.ErrorStage != nil && *event.ErrorStage != "" {
		// Track error
		ima.CountError++
		ima.ErrorStageCounts[*event.ErrorStage]++
	} else {
		// Track success and add samples
		ima.CountSuccess++
		ima.DNSSamples = append(ima.DNSSamples, event.Timings.DNSMs)
		ima.TCPSamples = append(ima.TCPSamples, event.Timings.TCPMs)
		ima.TLSSamples = append(ima.TLSSamples, event.Timings.TLSMs)
		ima.TTFBSamples = append(ima.TTFBSamples, event.Timings.HTTPTTFBMs)
		ima.ThroughputSamples = append(ima.ThroughputSamples, event.ThroughputKbps)
	}

	ima.UpdatedAt = time.Now()
}

// ToWindowedAggregate converts the in-memory aggregator to a WindowedAggregate
// by computing percentiles from the collected samples.
func (ima *InMemoryAggregator) ToWindowedAggregate() *WindowedAggregate {
	wa := &WindowedAggregate{
		ClientID:         ima.Key.ClientID,
		Target:           ima.Key.Target,
		WindowStartTs:    ima.Key.WindowStartTs,
		CountTotal:       ima.CountTotal,
		CountSuccess:     ima.CountSuccess,
		CountError:       ima.CountError,
		ErrorStageCounts: ima.ErrorStageCounts,
		UpdatedAt:        ima.UpdatedAt,
	}

	// Compute percentiles if we have samples
	if len(ima.DNSSamples) > 0 {
		wa.DNSP50 = calculatePercentile(ima.DNSSamples, 50)
		wa.DNSP95 = calculatePercentile(ima.DNSSamples, 95)
	}

	if len(ima.TCPSamples) > 0 {
		wa.TCPP50 = calculatePercentile(ima.TCPSamples, 50)
		wa.TCPP95 = calculatePercentile(ima.TCPSamples, 95)
	}

	if len(ima.TLSSamples) > 0 {
		wa.TLSP50 = calculatePercentile(ima.TLSSamples, 50)
		wa.TLSP95 = calculatePercentile(ima.TLSSamples, 95)
	}

	if len(ima.TTFBSamples) > 0 {
		wa.TTFBP50 = calculatePercentile(ima.TTFBSamples, 50)
		wa.TTFBP95 = calculatePercentile(ima.TTFBSamples, 95)
	}

	if len(ima.ThroughputSamples) > 0 {
		wa.ThroughputP50 = calculatePercentile(ima.ThroughputSamples, 50)
		wa.ThroughputP95 = calculatePercentile(ima.ThroughputSamples, 95)
	}

	return wa
}
