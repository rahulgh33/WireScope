package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"

	"github.com/rahulgh33/wirescope/internal/database"
	"github.com/rahulgh33/wirescope/internal/diagnosis"
	"github.com/rahulgh33/wirescope/internal/models"
	"github.com/rahulgh33/wirescope/internal/queue"
	"github.com/rahulgh33/wirescope/internal/tracing"
)

var (
	natsURL        = flag.String("nats-url", "nats://localhost:4222", "NATS server URL")
	dbHost         = flag.String("db-host", "localhost", "PostgreSQL host")
	dbPort         = flag.Int("db-port", 5432, "PostgreSQL port")
	dbName         = flag.String("db-name", "telemetry", "PostgreSQL database name")
	dbUser         = flag.String("db-user", "telemetry", "PostgreSQL user")
	dbPassword     = flag.String("db-password", "telemetry", "PostgreSQL password")
	windowSize     = flag.Duration("window-size", 60*time.Second, "Aggregation window size")
	flushDelay     = flag.Duration("flush-delay", 10*time.Second, "Delay before flushing closed windows")
	lateTolerance  = flag.Duration("late-tolerance", 2*time.Minute, "Tolerance for late event handling")
	consumerName   = flag.String("consumer-name", "aggregator-1", "Unique consumer name for this instance")
	metricsPort    = flag.String("metrics-port", "9090", "Prometheus metrics port")
	otlpEndpoint   = flag.String("otlp-endpoint", "localhost:4318", "OpenTelemetry OTLP HTTP endpoint")
	tracingEnabled = flag.Bool("tracing-enabled", true, "Enable distributed tracing")
)

// Prometheus metrics
// Requirements: 6.1, 6.2, 6.3
var (
	eventsProcessedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_processed_total",
			Help: "Total number of events processed",
		},
		[]string{"status"}, // success, duplicate, error
	)

	processingDelaySeconds = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "processing_delay_seconds",
			Help:    "End-to-end processing delay from event recv_ts_ms to aggregation completion",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60, 120},
		},
	)

	dedupRate = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "dedup_rate",
			Help: "Ratio of duplicate events to total events received (rolling window)",
		},
	)

	lateEventsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "late_events_total",
			Help: "Total number of late events detected",
		},
	)

	windowFlushDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "window_flush_duration_seconds",
			Help: "Time taken to flush aggregation windows to database",
			Buckets: []float64{
				0.001, // 1ms
				0.005, // 5ms
				0.010, // 10ms
				0.050, // 50ms
				0.100, // 100ms
				0.500, // 500ms
				1.0,   // 1s
				5.0,   // 5s
			},
		},
		[]string{"status"},
	)
)

func init() {
	prometheus.MustRegister(eventsProcessedTotal)
	prometheus.MustRegister(processingDelaySeconds)
	prometheus.MustRegister(dedupRate)
	prometheus.MustRegister(lateEventsTotal)
	prometheus.MustRegister(windowFlushDuration)
}

// Aggregator consumes events from NATS and produces windowed aggregates
type Aggregator struct {
	processor      models.EventProcessor
	eventsSeenRepo *database.EventsSeenRepository
	aggregatesRepo *database.AggregatesRepository
	repository     *database.Repository // For fetching historical data

	mu               sync.RWMutex
	aggregators      map[string]*models.InMemoryAggregator
	windowStartTimes map[int64]bool

	windowSize    time.Duration
	flushDelay    time.Duration
	lateTolerance time.Duration // Tolerance for late events

	// Dedup tracking for metrics
	totalProcessed int64
	duplicateCount int64

	ctx    context.Context
	cancel context.CancelFunc
}

func NewAggregator(
	processor models.EventProcessor,
	eventsSeenRepo *database.EventsSeenRepository,
	aggregatesRepo *database.AggregatesRepository,
	repository *database.Repository,
	windowSize, flushDelay, lateTolerance time.Duration,
) *Aggregator {
	ctx, cancel := context.WithCancel(context.Background())

	return &Aggregator{
		processor:        processor,
		eventsSeenRepo:   eventsSeenRepo,
		aggregatesRepo:   aggregatesRepo,
		repository:       repository,
		aggregators:      make(map[string]*models.InMemoryAggregator),
		windowStartTimes: make(map[int64]bool),
		windowSize:       windowSize,
		flushDelay:       flushDelay,
		lateTolerance:    lateTolerance,
		ctx:              ctx,
		cancel:           cancel,
	}
}

func (a *Aggregator) Start() error {
	log.Printf("Starting aggregator with window size: %v, flush delay: %v", a.windowSize, a.flushDelay)

	go a.periodicWindowFlusher()

	return a.processor.ConsumeEvents(a.handleEvent)
}

func (a *Aggregator) Stop() {
	log.Printf("Stopping aggregator...")
	a.cancel()

	a.flushAllWindows()

	if err := a.processor.Close(); err != nil {
		log.Printf("Error closing processor: %v", err)
	}
}

func (a *Aggregator) handleEvent(event *models.TelemetryEvent) error {
	// Create trace span for event processing
	// Requirement: 6.4 - Aggregator operation tracing
	tracer := tracing.GetTracer("aggregator")
	// Extract parent context from event's trace fields for propagation
	parentCtx := tracing.ExtractContextFromEvent(a.ctx, event)
	ctx, span := tracer.Start(parentCtx, "aggregator.processEvent")
	defer span.End()

	// Add event attributes to span
	// Requirement: 6.5 - Span attributes for debugging
	span.SetAttributes(
		attribute.String("event.id", event.EventID),
		attribute.String("event.client_id", event.ClientID),
		attribute.String("event.target", event.Target),
		attribute.Int64("event.timestamp_ms", event.TimestampMs),
	)

	if err := a.processEventWithDedup(ctx, event); err != nil {
		tracing.RecordError(ctx, err)
		return fmt.Errorf("failed to process event %s: %w", event.EventID, err)
	}

	if err := a.processor.AckEvent(event.EventID); err != nil {
		log.Printf("Warning: Failed to explicitly ACK event %s: %v", event.EventID, err)
	}

	tracing.AddSpanEvent(ctx, "event.processed")
	return nil
}

func (a *Aggregator) processEventWithDedup(ctx context.Context, event *models.TelemetryEvent) error {
	// Track processing delay if recv_ts_ms is available
	if event.RecvTimestampMs != nil && *event.RecvTimestampMs > 0 {
		defer func() {
			// Calculate end-to-end delay from recv_ts_ms to now
			delaySeconds := float64(time.Now().UnixMilli()-*event.RecvTimestampMs) / 1000.0
			processingDelaySeconds.Observe(delaySeconds)

			// Add delay to span
			tracing.AddSpanAttributes(ctx, attribute.Float64("processing.delay_seconds", delaySeconds))
		}()
	}

	// Check if event is too late (processing time > recv_ts_ms + tolerance)
	// Requirement: 3.4 - Late event handling with 2-minute tolerance
	if event.RecvTimestampMs != nil && *event.RecvTimestampMs > 0 {
		processingTime := time.Now().UnixMilli()
		latencyMs := processingTime - *event.RecvTimestampMs

		if latencyMs > a.lateTolerance.Milliseconds() {
			lateEventsTotal.Inc()
			tracing.AddSpanEvent(ctx, "event.late", attribute.Int64("latency_ms", latencyMs))
			log.Printf("Late event detected: %s (latency: %dms > %dms tolerance)",
				event.EventID, latencyMs, a.lateTolerance.Milliseconds())
			// Continue processing late events, just log them for monitoring
		}
	}

	var isNewEvent bool
	// Add database transaction span
	// Requirement: 6.4 - Database transaction tracing
	tracing.AddSpanEvent(ctx, "db.transaction.start")
	err := a.eventsSeenRepo.WithTransaction(ctx, func(tx *sql.Tx) error {
		isNew, err := a.eventsSeenRepo.InsertEventSeen(ctx, event.EventID, event.ClientID, event.TimestampMs)
		if err != nil {
			tracing.RecordError(ctx, err)
			return err
		}

		isNewEvent = isNew

		if !isNew {
			tracing.AddSpanEvent(ctx, "event.duplicate")
			log.Printf("Duplicate event detected: %s (client: %s)", event.EventID, event.ClientID)
			return nil
		}

		windowStartMs := event.GetWindowStartMs()
		windowStartTime := time.UnixMilli(windowStartMs)
		aggregatorKey := getAggregatorKey(event.ClientID, event.Target, windowStartMs)

		// Add window attributes to span
		tracing.AddSpanAttributes(ctx,
			attribute.Int64("window.start_ms", windowStartMs),
			attribute.String("window.start_time", windowStartTime.Format(time.RFC3339)),
		)

		a.mu.Lock()
		aggregator, exists := a.aggregators[aggregatorKey]
		if !exists {
			key := models.AggregateKey{
				ClientID:      event.ClientID,
				Target:        event.Target,
				WindowStartTs: windowStartTime,
			}
			aggregator = models.NewInMemoryAggregator(key)
			a.aggregators[aggregatorKey] = aggregator
			a.windowStartTimes[windowStartMs] = true
			tracing.AddSpanEvent(ctx, "aggregator.created")
		}
		a.mu.Unlock()

		aggregator.AddEvent(event)
		tracing.AddSpanEvent(ctx, "event.aggregated")

		log.Printf("Processed event %s (client: %s, target: %s, window: %d)",
			event.EventID, event.ClientID, event.Target, windowStartMs)

		return nil
	})

	// Update metrics after transaction completes
	if err != nil {
		eventsProcessedTotal.WithLabelValues("error").Inc()
		return err
	}

	// Track total and duplicate counts for dedup rate calculation
	a.totalProcessed++
	if !isNewEvent {
		a.duplicateCount++
		eventsProcessedTotal.WithLabelValues("duplicate").Inc()

		// Update dedup rate
		if a.totalProcessed > 0 {
			rate := float64(a.duplicateCount) / float64(a.totalProcessed)
			dedupRate.Set(rate)
		}
	} else {
		eventsProcessedTotal.WithLabelValues("success").Inc()
	}

	return nil
}

func (a *Aggregator) periodicWindowFlusher() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.flushClosedWindows()
		}
	}
}

func (a *Aggregator) flushClosedWindows() {
	now := time.Now()
	windowSizeMs := a.windowSize.Milliseconds()
	currentWindowStartMs := (now.UnixMilli() / windowSizeMs) * windowSizeMs

	flushBeforeMs := currentWindowStartMs - a.flushDelay.Milliseconds()

	a.mu.RLock()
	var windowsToFlush []int64
	for windowStartMs := range a.windowStartTimes {
		if windowStartMs < flushBeforeMs {
			windowsToFlush = append(windowsToFlush, windowStartMs)
		}
	}
	a.mu.RUnlock()

	for _, windowStartMs := range windowsToFlush {
		a.flushWindow(windowStartMs)
	}
}

func (a *Aggregator) flushWindow(windowStartMs int64) {
	start := time.Now()
	status := "success"
	defer func() {
		duration := time.Since(start).Seconds()
		windowFlushDuration.WithLabelValues(status).Observe(duration)
	}()

	a.mu.Lock()

	var aggregatorsToFlush []*models.InMemoryAggregator
	var keysToDelete []string

	for key, aggregator := range a.aggregators {
		if aggregator.Key.WindowStartTs.UnixMilli() == windowStartMs {
			aggregatorsToFlush = append(aggregatorsToFlush, aggregator)
			keysToDelete = append(keysToDelete, key)
		}
	}

	for _, key := range keysToDelete {
		delete(a.aggregators, key)
	}
	delete(a.windowStartTimes, windowStartMs)

	a.mu.Unlock()

	if len(aggregatorsToFlush) == 0 {
		return
	}

	log.Printf("Flushing window %d with %d aggregates", windowStartMs, len(aggregatorsToFlush))

	ctx := context.Background()
	for _, aggregator := range aggregatorsToFlush {
		windowedAgg := aggregator.ToWindowedAggregate()

		// Run diagnosis engine to determine issue type
		// Requirements: 5.1, 5.2, 5.3, 5.4, 5.5
		diagnosisLabel := a.runDiagnosis(ctx, windowedAgg)

		dbAgg := convertToDBAggregate(windowedAgg)
		// Convert string to *string for diagnosis_label
		if diagnosisLabel != "" {
			dbAgg.DiagnosisLabel = &diagnosisLabel
		}

		if err := a.aggregatesRepo.UpsertAggregate(ctx, dbAgg); err != nil {
			status = "error"
			log.Printf("Failed to upsert aggregate for client %s, target %s, window %s: %v",
				windowedAgg.ClientID, windowedAgg.Target, windowedAgg.WindowStartTs.Format(time.RFC3339), err)
			continue
		}

		log.Printf("Flushed aggregate: client=%s, target=%s, window=%s, total=%d, success=%d, error=%d, diagnosis=%s",
			windowedAgg.ClientID, windowedAgg.Target, windowedAgg.WindowStartTs.Format(time.RFC3339),
			windowedAgg.CountTotal, windowedAgg.CountSuccess, windowedAgg.CountError, diagnosisLabel)
	}
}

func (a *Aggregator) flushAllWindows() {
	a.mu.RLock()
	var allWindowStarts []int64
	for windowStartMs := range a.windowStartTimes {
		allWindowStarts = append(allWindowStarts, windowStartMs)
	}
	a.mu.RUnlock()

	log.Printf("Flushing all %d remaining windows on shutdown", len(allWindowStarts))
	for _, windowStartMs := range allWindowStarts {
		a.flushWindow(windowStartMs)
	}
}

// runDiagnosis performs automated diagnosis on the current window metrics
// Returns a diagnosis label based on explicit thresholds and historical baseline
//
// Requirements: 5.1, 5.2, 5.3, 5.4, 5.5
func (a *Aggregator) runDiagnosis(ctx context.Context, agg *models.WindowedAggregate) string {
	// Fetch last 10 windows for baseline calculation
	historicalAggs, err := a.repository.GetHistoricalAggregates(ctx, agg.ClientID, agg.Target, 10)
	if err != nil {
		log.Printf("Warning: Failed to fetch historical aggregates for diagnosis: %v", err)
		return ""
	}

	// Need at least 3 historical windows for meaningful baseline
	if len(historicalAggs) < 3 {
		return ""
	}

	// Convert historical aggregates to diagnosis.WindowMetrics
	var historicalWindows []diagnosis.WindowMetrics
	for _, h := range historicalAggs {
		// Skip windows with insufficient data
		if h.CountSuccess < 5 {
			continue
		}

		window := diagnosis.WindowMetrics{
			WindowStartTs:   h.WindowStartTs,
			DNSP95:          getFloatValue(h.DNSP95),
			TCPP95:          getFloatValue(h.TCPP95),
			TLSP95:          getFloatValue(h.TLSP95),
			TTFBP95:         getFloatValue(h.TTFBP95),
			TotalLatencyP95: getFloatValue(h.DNSP95) + getFloatValue(h.TCPP95) + getFloatValue(h.TLSP95) + getFloatValue(h.TTFBP95),
			ThroughputP50:   getFloatValue(h.ThroughputP50),
			CountSuccess:    int(h.CountSuccess),
		}
		historicalWindows = append(historicalWindows, window)
	}

	// Calculate baseline from historical data
	baseline := diagnosis.CalculateBaseline(historicalWindows)

	// Build current window metrics
	currentWindow := diagnosis.WindowMetrics{
		WindowStartTs:   agg.WindowStartTs,
		DNSP95:          agg.DNSP95,
		TCPP95:          agg.TCPP95,
		TLSP95:          agg.TLSP95,
		TTFBP95:         agg.TTFBP95,
		TotalLatencyP95: agg.DNSP95 + agg.TCPP95 + agg.TLSP95 + agg.TTFBP95,
		ThroughputP50:   agg.ThroughputP50,
		CountSuccess:    int(agg.CountSuccess),
	}

	// Run diagnosis
	label := diagnosis.Diagnose(currentWindow, baseline)
	return string(label)
}

// Helper function to safely extract float value from *float64
func getFloatValue(ptr *float64) float64 {
	if ptr == nil {
		return 0.0
	}
	return *ptr
}

func getAggregatorKey(clientID, target string, windowStartMs int64) string {
	return fmt.Sprintf("%s:%s:%d", clientID, target, windowStartMs)
}

func convertToDBAggregate(agg *models.WindowedAggregate) *database.WindowedAggregate {
	floatPtr := func(f float64) *float64 {
		if f == 0 {
			return nil
		}
		return &f
	}

	return &database.WindowedAggregate{
		ClientID:             agg.ClientID,
		Target:               agg.Target,
		WindowStartTs:        agg.WindowStartTs,
		CountTotal:           agg.CountTotal,
		CountSuccess:         agg.CountSuccess,
		CountError:           agg.CountError,
		DNSErrorCount:        agg.ErrorStageCounts["DNS"],
		TCPErrorCount:        agg.ErrorStageCounts["TCP"],
		TLSErrorCount:        agg.ErrorStageCounts["TLS"],
		HTTPErrorCount:       agg.ErrorStageCounts["HTTP"],
		ThroughputErrorCount: agg.ErrorStageCounts["throughput"],
		DNSP50:               floatPtr(agg.DNSP50),
		DNSP95:               floatPtr(agg.DNSP95),
		TCPP50:               floatPtr(agg.TCPP50),
		TCPP95:               floatPtr(agg.TCPP95),
		TLSP50:               floatPtr(agg.TLSP50),
		TLSP95:               floatPtr(agg.TLSP95),
		TTFBP50:              floatPtr(agg.TTFBP50),
		TTFBP95:              floatPtr(agg.TTFBP95),
		ThroughputP50:        floatPtr(agg.ThroughputP50),
		ThroughputP95:        floatPtr(agg.ThroughputP95),
		DiagnosisLabel:       nil,
		UpdatedAt:            time.Now(),
	}
}

func main() {
	flag.Parse()

	log.Printf("Starting WireScope Aggregator")
	log.Printf("NATS URL: %s", *natsURL)
	log.Printf("Database: %s@%s:%d/%s", *dbUser, *dbHost, *dbPort, *dbName)
	log.Printf("Consumer name: %s", *consumerName)

	// Initialize OpenTelemetry tracing
	// Requirement: 6.4 - Distributed tracing setup
	tracingConfig := tracing.DefaultConfig("aggregator")
	tracingConfig.OTLPEndpoint = *otlpEndpoint
	tracingConfig.Enabled = *tracingEnabled

	shutdownTracing, err := tracing.InitTracer(tracingConfig)
	if err != nil {
		log.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer func() {
		if err := shutdownTracing(context.Background()); err != nil {
			log.Printf("Error shutting down tracing: %v", err)
		}
	}()

	dbConfig := database.DefaultConnectionConfig()
	dbConfig.Host = *dbHost
	dbConfig.Port = *dbPort
	dbConfig.Database = *dbName
	dbConfig.User = *dbUser
	dbConfig.Password = *dbPassword

	dbConn, err := database.NewConnection(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	log.Printf("Connected to database")

	eventsSeenRepo := database.NewEventsSeenRepository(dbConn)
	aggregatesRepo := database.NewAggregatesRepository(dbConn)

	natsConfig := queue.DefaultNATSConfig()
	natsConfig.URL = *natsURL

	processor, err := queue.NewNATSEventProcessor(natsConfig)
	if err != nil {
		log.Fatalf("Failed to create NATS processor: %v", err)
	}
	defer processor.Close()

	log.Printf("Connected to NATS")

	// Create general repository for historical queries
	repo := database.NewRepository(dbConn)

	aggregator := NewAggregator(
		processor,
		eventsSeenRepo,
		aggregatesRepo,
		repo,
		*windowSize,
		*flushDelay,
		*lateTolerance,
	)

	// Start metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		addr := ":" + *metricsPort
		log.Printf("Metrics server listening on %s", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	// Periodically update queue lag metrics
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if err := processor.UpdateQueueMetrics(); err != nil {
				log.Printf("Failed to update queue metrics: %v", err)
			}
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		if err := aggregator.Start(); err != nil {
			errCh <- err
		}
	}()

	log.Printf("Aggregator started successfully")

	select {
	case <-sigCh:
		log.Printf("Received shutdown signal")
	case err := <-errCh:
		log.Printf("Aggregator error: %v", err)
	}

	aggregator.Stop()
	log.Printf("Aggregator stopped")
}
