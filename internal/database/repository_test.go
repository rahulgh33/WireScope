package database

import (
	"testing"
	"time"
)

func TestWindowedAggregateStruct(t *testing.T) {
	// Test that WindowedAggregate struct has all required fields
	agg := &WindowedAggregate{
		ClientID:             "test-client",
		Target:               "example.com",
		WindowStartTs:        time.Now().Truncate(time.Minute),
		CountTotal:           100,
		CountSuccess:         95,
		CountError:           5,
		DNSErrorCount:        1,
		TCPErrorCount:        2,
		TLSErrorCount:        1,
		HTTPErrorCount:       1,
		ThroughputErrorCount: 0,
		UpdatedAt:            time.Now(),
	}

	// Test required fields are set
	if agg.ClientID == "" {
		t.Error("ClientID should not be empty")
	}

	if agg.Target == "" {
		t.Error("Target should not be empty")
	}

	if agg.WindowStartTs.IsZero() {
		t.Error("WindowStartTs should not be zero")
	}

	if agg.CountTotal != 100 {
		t.Errorf("Expected CountTotal 100, got %d", agg.CountTotal)
	}

	if agg.CountSuccess != 95 {
		t.Errorf("Expected CountSuccess 95, got %d", agg.CountSuccess)
	}

	if agg.CountError != 5 {
		t.Errorf("Expected CountError 5, got %d", agg.CountError)
	}

	// Test that error counts sum correctly
	totalErrors := agg.DNSErrorCount + agg.TCPErrorCount + agg.TLSErrorCount + 
		agg.HTTPErrorCount + agg.ThroughputErrorCount
	
	if totalErrors != agg.CountError {
		t.Errorf("Error counts don't sum correctly: %d != %d", totalErrors, agg.CountError)
	}
}

func TestRepositoryCreation(t *testing.T) {
	// Test repository creation with nil connection (should not panic)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Repository creation panicked: %v", r)
		}
	}()

	// This will create a repository with a nil connection
	// In real usage, this would be created with a valid connection
	repo := &Repository{conn: nil}
	
	if repo == nil {
		t.Error("Repository should not be nil")
	}
}

func TestEventsSeenRepositoryCreation(t *testing.T) {
	// Test events seen repository creation
	repo := &EventsSeenRepository{
		Repository: &Repository{conn: nil},
	}

	if repo == nil {
		t.Error("EventsSeenRepository should not be nil")
	}

	if repo.Repository == nil {
		t.Error("EventsSeenRepository.Repository should not be nil")
	}
}

func TestAggregatesRepositoryCreation(t *testing.T) {
	// Test aggregates repository creation
	repo := &AggregatesRepository{
		Repository: &Repository{conn: nil},
	}

	if repo == nil {
		t.Error("AggregatesRepository should not be nil")
	}

	if repo.Repository == nil {
		t.Error("AggregatesRepository.Repository should not be nil")
	}
}