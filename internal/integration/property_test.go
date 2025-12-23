package integration

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/network-qoe-telemetry-platform/internal/database"
	"github.com/network-qoe-telemetry-platform/internal/models"
)

// Property 3: Exactly-once aggregate effects via deduplication
func TestProperty_ExactlyOnceAggregateEffects(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	eventsSeenRepo := database.NewEventsSeenRepository(db)
	aggregatesRepo := database.NewAggregatesRepository(db)

	ctx := context.Background()
	clientID := "test-client-" + uuid.New().String()
	target := "http://test.example.com"
	eventID := uuid.New().String()
	timestampMs := time.Now().UnixMilli()
	windowStartMs := (timestampMs / 60000) * 60000

	for i := 0; i < 2; i++ {
		err := eventsSeenRepo.WithTransaction(ctx, func(tx *sql.Tx) error {
			isNew, err := eventsSeenRepo.InsertEventSeen(ctx, eventID, clientID, timestampMs)
			if err != nil {
				return err
			}

			if isNew {
				agg := &database.WindowedAggregate{
					ClientID:      clientID,
					Target:        target,
					WindowStartTs: time.UnixMilli(windowStartMs),
					CountTotal:    1,
					CountSuccess:  1,
					CountError:    0,
				}
				return aggregatesRepo.UpsertAggregate(ctx, agg)
			}
			return nil
		})

		if err != nil {
			t.Fatalf("Failed to process event (iteration %d): %v", i, err)
		}
	}

	var countTotal int
	err := db.DB().QueryRow(`
		SELECT count_total FROM agg_1m 
		WHERE client_id = $1 AND target = $2 AND window_start_ts = $3
	`, clientID, target, time.UnixMilli(windowStartMs)).Scan(&countTotal)

	if err != nil {
		t.Fatalf("Failed to query aggregate: %v", err)
	}

	if countTotal != 1 {
		t.Errorf("Expected count_total = 1, got %d", countTotal)
	}

	t.Logf("Property 3 verified: count_total = %d", countTotal)
}

// Property 4: Transactional consistency
func TestProperty_TransactionalConsistency(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	eventsSeenRepo := database.NewEventsSeenRepository(db)
	ctx := context.Background()

	clientID := "test-client-" + uuid.New().String()
	eventID := uuid.New().String()
	timestampMs := time.Now().UnixMilli()

	// Test 1: Successful transaction - should commit
	err := eventsSeenRepo.WithTransaction(ctx, func(tx *sql.Tx) error {
		query := `INSERT INTO events_seen (event_id, client_id, ts_ms) VALUES ($1, $2, $3)`
		_, err := tx.ExecContext(ctx, query, eventID, clientID, timestampMs)
		return err
	})

	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	var count int
	err = db.DB().QueryRow("SELECT COUNT(*) FROM events_seen WHERE event_id = $1", eventID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected event persisted, got count = %d", count)
	}

	// Test 2: Failed transaction - should rollback
	eventID2 := uuid.New().String()
	err = eventsSeenRepo.WithTransaction(ctx, func(tx *sql.Tx) error {
		query := `INSERT INTO events_seen (event_id, client_id, ts_ms) VALUES ($1, $2, $3)`
		_, err := tx.ExecContext(ctx, query, eventID2, clientID, timestampMs)
		if err != nil {
			return err
		}
		return fmt.Errorf("simulated error")
	})

	if err == nil {
		t.Fatal("Expected transaction to fail")
	}

	err = db.DB().QueryRow("SELECT COUNT(*) FROM events_seen WHERE event_id = $1", eventID2).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected rollback, got count = %d", count)
	}

	t.Logf("Property 4 verified")
}

// Property 5: Late event handling
func TestProperty_LateEventHandling(t *testing.T) {
	lateTolerance := 2 * time.Minute

	testCases := []struct {
		name         string
		recvTsAge    time.Duration
		expectedLate bool
	}{
		{"Recent event (30s)", 30 * time.Second, false},
		{"On-time (1m)", 1 * time.Minute, false},
		{"Borderline (2m)", 2 * time.Minute, false},
		{"Late (3m)", 3 * time.Minute, true},
		{"Very late (10m)", 10 * time.Minute, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			now := time.Now()
			recvTsMs := now.Add(-tc.recvTsAge).UnixMilli()
			processingTimeMs := now.UnixMilli()

			latencyMs := processingTimeMs - recvTsMs
			isLate := latencyMs > lateTolerance.Milliseconds()

			if isLate != tc.expectedLate {
				t.Errorf("Expected %v, got %v", tc.expectedLate, isLate)
			}
		})
	}

	t.Logf("Property 5 verified")
}

// Property 6: Window assignment accuracy
func TestProperty_WindowAssignment(t *testing.T) {
	testCases := []struct {
		name        string
		timestampMs int64
		expectedMs  int64
	}{
		{"Start of minute", 1703297100000, 1703297100000},
		{"Middle", 1703297130500, 1703297100000},
		{"End", 1703297159999, 1703297100000},
		{"Next boundary", 1703297160000, 1703297160000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event := &models.TelemetryEvent{TimestampMs: tc.timestampMs}
			windowMs := event.GetWindowStartMs()

			if windowMs != tc.expectedMs {
				t.Errorf("Expected %d, got %d", tc.expectedMs, windowMs)
			}
		})
	}

	t.Logf("Property 6 verified")
}

// Property 7: Percentile calculation
func TestProperty_PercentileCalculation(t *testing.T) {
	testCases := []struct {
		name   string
		values []float64
		p50    float64
		p95    float64
	}{
		{"Single", []float64{10.0}, 10.0, 10.0},
		{"Two", []float64{10.0, 20.0}, 15.0, 19.5},
		{"Sequence 1-100", makeSequence(1.0, 100.0), 50.5, 95.05},
		{"All same", []float64{5.0, 5.0, 5.0, 5.0}, 5.0, 5.0},
		{"Large (1000)", makeSequence(1.0, 1000.0), 500.5, 950.05},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p50, p95 := models.CalculatePercentiles(tc.values)

			if !floatEquals(p50, tc.p50, 0.1) {
				t.Errorf("P50: expected %.2f, got %.2f", tc.p50, p50)
			}
			if !floatEquals(p95, tc.p95, 0.1) {
				t.Errorf("P95: expected %.2f, got %.2f", tc.p95, p95)
			}
		})
	}

	t.Logf("Property 7 verified")
}

func setupTestDB(t *testing.T) *database.Connection {
	t.Helper()

	config := database.DefaultConnectionConfig()
	config.Host = "localhost"
	config.Port = 5432
	config.Database = "telemetry"
	config.User = "telemetry"
	config.Password = "telemetry"

	db, err := database.NewConnection(config)
	if err != nil {
		t.Skipf("Skipping - database not available: %v", err)
	}

	cleanupTestData(t, db)
	return db
}

func cleanupTestData(t *testing.T, db *database.Connection) {
	t.Helper()
	db.DB().Exec("DELETE FROM events_seen WHERE client_id LIKE 'test-client-%'")
	db.DB().Exec("DELETE FROM agg_1m WHERE client_id LIKE 'test-client-%'")
}

func makeSequence(start, end float64) []float64 {
	size := int(end - start + 1)
	values := make([]float64, size)
	for i := 0; i < size; i++ {
		values[i] = start + float64(i)
	}
	return values
}

func floatEquals(a, b, tolerance float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}
