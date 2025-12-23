package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/network-qoe-telemetry-platform/internal/database"
	"github.com/network-qoe-telemetry-platform/internal/models"
	"github.com/network-qoe-telemetry-platform/internal/queue"
)

var (
	natsURL       = flag.String("nats-url", "nats://localhost:4222", "NATS server URL")
	dbHost        = flag.String("db-host", "localhost", "PostgreSQL host")
	dbPort        = flag.Int("db-port", 5432, "PostgreSQL port")
	dbName        = flag.String("db-name", "telemetry", "PostgreSQL database name")
	dbUser        = flag.String("db-user", "postgres", "PostgreSQL user")
	dbPassword    = flag.String("db-password", "postgres", "PostgreSQL password")
	windowSize    = flag.Duration("window-size", 60*time.Second, "Aggregation window size")
	flushDelay    = flag.Duration("flush-delay", 10*time.Second, "Delay before flushing closed windows")
	lateTolerance = flag.Duration("late-tolerance", 2*time.Minute, "Tolerance for late event handling")
	consumerName  = flag.String("consumer-name", "aggregator-1", "Unique consumer name for this instance")
)

// Aggregator consumes events from NATS and produces windowed aggregates
type Aggregator struct {
	processor      models.EventProcessor
	eventsSeenRepo *database.EventsSeenRepository
	aggregatesRepo *database.AggregatesRepository

	mu               sync.RWMutex
	aggregators      map[string]*models.InMemoryAggregator
	windowStartTimes map[int64]bool

	windowSize    time.Duration
	flushDelay    time.Duration
	lateTolerance time.Duration // Tolerance for late events

	// Metrics
	lateEventCount    int64
	droppedEventCount int64

	ctx    context.Context
	cancel context.CancelFunc
}

func NewAggregator(
	processor models.EventProcessor,
	eventsSeenRepo *database.EventsSeenRepository,
	aggregatesRepo *database.AggregatesRepository,
	windowSize, flushDelay, lateTolerance time.Duration,
) *Aggregator {
	ctx, cancel := context.WithCancel(context.Background())

	return &Aggregator{
		processor:        processor,
		eventsSeenRepo:   eventsSeenRepo,
		aggregatesRepo:   aggregatesRepo,
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
	log.Printf("Metrics: late_events=%d, dropped_events=%d", a.lateEventCount, a.droppedEventCount)
	a.cancel()

	a.flushAllWindows()

	if err := a.processor.Close(); err != nil {
		log.Printf("Error closing processor: %v", err)
	}
}

func (a *Aggregator) handleEvent(event *models.TelemetryEvent) error {
	if err := a.processEventWithDedup(event); err != nil {
		return fmt.Errorf("failed to process event %s: %w", event.EventID, err)
	}

	if err := a.processor.AckEvent(event.EventID); err != nil {
		log.Printf("Warning: Failed to explicitly ACK event %s: %v", event.EventID, err)
	}

	return nil
}

func (a *Aggregator) processEventWithDedup(event *models.TelemetryEvent) error {
	ctx := context.Background()

	// Check if event is too late (processing time > recv_ts_ms + tolerance)
	// Requirement: 3.4 - Late event handling with 2-minute tolerance
	if event.RecvTimestampMs != nil && *event.RecvTimestampMs > 0 {
		processingTime := time.Now().UnixMilli()
		latencyMs := processingTime - *event.RecvTimestampMs

		if latencyMs > a.lateTolerance.Milliseconds() {
			a.lateEventCount++
			log.Printf("Late event detected: %s (latency: %dms > %dms tolerance)",
				event.EventID, latencyMs, a.lateTolerance.Milliseconds())
			// Continue processing late events, just log them for monitoring
		}
	}

	return a.eventsSeenRepo.WithTransaction(ctx, func(tx *sql.Tx) error {
		isNew, err := a.eventsSeenRepo.InsertEventSeen(ctx, event.EventID, event.ClientID, event.TimestampMs)
		if err != nil {
			return err
		}

		if !isNew {
			log.Printf("Duplicate event detected: %s (client: %s)", event.EventID, event.ClientID)
			return nil
		}

		windowStartMs := event.GetWindowStartMs()
		windowStartTime := time.UnixMilli(windowStartMs)
		aggregatorKey := getAggregatorKey(event.ClientID, event.Target, windowStartMs)

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
		}
		a.mu.Unlock()

		aggregator.AddEvent(event)

		log.Printf("Processed event %s (client: %s, target: %s, window: %d)",
			event.EventID, event.ClientID, event.Target, windowStartMs)

		return nil
	})
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

		dbAgg := convertToDBAggregate(windowedAgg)

		if err := a.aggregatesRepo.UpsertAggregate(ctx, dbAgg); err != nil {
			log.Printf("Failed to upsert aggregate for client %s, target %s, window %s: %v",
				windowedAgg.ClientID, windowedAgg.Target, windowedAgg.WindowStartTs.Format(time.RFC3339), err)
			continue
		}

		log.Printf("Flushed aggregate: client=%s, target=%s, window=%s, total=%d, success=%d, error=%d",
			windowedAgg.ClientID, windowedAgg.Target, windowedAgg.WindowStartTs.Format(time.RFC3339),
			windowedAgg.CountTotal, windowedAgg.CountSuccess, windowedAgg.CountError)
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

	log.Printf("Starting Network QoE Aggregator")
	log.Printf("NATS URL: %s", *natsURL)
	log.Printf("Database: %s@%s:%d/%s", *dbUser, *dbHost, *dbPort, *dbName)
	log.Printf("Consumer name: %s", *consumerName)

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

	aggregator := NewAggregator(
		processor,
		eventsSeenRepo,
		aggregatesRepo,
		*windowSize,
		*flushDelay,
		*lateTolerance,
	)

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
