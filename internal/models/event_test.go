package models

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTelemetryEventValidation(t *testing.T) {
	tests := []struct {
		name    string
		event   *TelemetryEvent
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid event",
			event: &TelemetryEvent{
				EventID:       uuid.New().String(),
				ClientID:      "test-client-123",
				TimestampMs:   time.Now().UnixMilli(),
				SchemaVersion: "1.0",
				Target:        "https://example.com",
				NetworkContext: NetworkContext{
					InterfaceType: "wifi",
					VPNEnabled:    false,
				},
				Timings: TimingMeasurements{
					DNSMs:      10.5,
					TCPMs:      20.3,
					TLSMs:      30.1,
					HTTPTTFBMs: 40.2,
				},
				ThroughputKbps: 5000.0,
			},
			wantErr: false,
		},
		{
			name: "invalid event_id",
			event: &TelemetryEvent{
				EventID:       "not-a-uuid",
				ClientID:      "test-client-123",
				TimestampMs:   time.Now().UnixMilli(),
				SchemaVersion: "1.0",
				Target:        "https://example.com",
				NetworkContext: NetworkContext{
					InterfaceType: "wifi",
				},
			},
			wantErr: true,
			errMsg:  "invalid event_id",
		},
		{
			name: "missing client_id",
			event: &TelemetryEvent{
				EventID:       uuid.New().String(),
				ClientID:      "",
				TimestampMs:   time.Now().UnixMilli(),
				SchemaVersion: "1.0",
				Target:        "https://example.com",
				NetworkContext: NetworkContext{
					InterfaceType: "wifi",
				},
			},
			wantErr: true,
			errMsg:  "client_id is required",
		},
		{
			name: "zero timestamp",
			event: &TelemetryEvent{
				EventID:       uuid.New().String(),
				ClientID:      "test-client-123",
				TimestampMs:   0,
				SchemaVersion: "1.0",
				Target:        "https://example.com",
				NetworkContext: NetworkContext{
					InterfaceType: "wifi",
				},
			},
			wantErr: true,
			errMsg:  "ts_ms must be positive",
		},
		{
			name: "missing schema_version",
			event: &TelemetryEvent{
				EventID:       uuid.New().String(),
				ClientID:      "test-client-123",
				TimestampMs:   time.Now().UnixMilli(),
				SchemaVersion: "",
				Target:        "https://example.com",
				NetworkContext: NetworkContext{
					InterfaceType: "wifi",
				},
			},
			wantErr: true,
			errMsg:  "schema_version is required",
		},
		{
			name: "missing target",
			event: &TelemetryEvent{
				EventID:       uuid.New().String(),
				ClientID:      "test-client-123",
				TimestampMs:   time.Now().UnixMilli(),
				SchemaVersion: "1.0",
				Target:        "",
				NetworkContext: NetworkContext{
					InterfaceType: "wifi",
				},
			},
			wantErr: true,
			errMsg:  "target is required",
		},
		{
			name: "negative timing",
			event: &TelemetryEvent{
				EventID:       uuid.New().String(),
				ClientID:      "test-client-123",
				TimestampMs:   time.Now().UnixMilli(),
				SchemaVersion: "1.0",
				Target:        "https://example.com",
				NetworkContext: NetworkContext{
					InterfaceType: "wifi",
				},
				Timings: TimingMeasurements{
					DNSMs:      -5.0,
					TCPMs:      20.3,
					TLSMs:      30.1,
					HTTPTTFBMs: 40.2,
				},
				ThroughputKbps: 5000.0,
			},
			wantErr: true,
			errMsg:  "dns_ms must be non-negative",
		},
		{
			name: "event with error stage - no validation on timings",
			event: &TelemetryEvent{
				EventID:       uuid.New().String(),
				ClientID:      "test-client-123",
				TimestampMs:   time.Now().UnixMilli(),
				SchemaVersion: "1.0",
				Target:        "https://example.com",
				NetworkContext: NetworkContext{
					InterfaceType: "wifi",
				},
				ErrorStage: stringPtr("DNS"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("TelemetryEvent.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("TelemetryEvent.Validate() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestGetWindowStartMs(t *testing.T) {
	tests := []struct {
		name        string
		timestampMs int64
		want        int64
	}{
		{
			name:        "exact window boundary",
			timestampMs: 60000,
			want:        60000,
		},
		{
			name:        "middle of window",
			timestampMs: 90000,
			want:        60000,
		},
		{
			name:        "near end of window",
			timestampMs: 119999,
			want:        60000,
		},
		{
			name:        "start of next window",
			timestampMs: 120000,
			want:        120000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &TelemetryEvent{
				TimestampMs: tt.timestampMs,
			}
			if got := event.GetWindowStartMs(); got != tt.want {
				t.Errorf("TelemetryEvent.GetWindowStartMs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsLate(t *testing.T) {
	now := time.Now()
	tolerance := 2 * time.Minute

	tests := []struct {
		name       string
		eventTime  time.Time
		procTime   time.Time
		tolerance  time.Duration
		wantIsLate bool
	}{
		{
			name:       "recent event",
			eventTime:  now.Add(-30 * time.Second),
			procTime:   now,
			tolerance:  tolerance,
			wantIsLate: false,
		},
		{
			name:       "event within tolerance",
			eventTime:  now.Add(-90 * time.Second),
			procTime:   now,
			tolerance:  tolerance,
			wantIsLate: false,
		},
		{
			name:       "event at tolerance boundary",
			eventTime:  now.Add(-2 * time.Minute),
			procTime:   now,
			tolerance:  tolerance,
			wantIsLate: true, // At exact boundary, time delta == tolerance, so it IS late
		},
		{
			name:       "late event",
			eventTime:  now.Add(-3 * time.Minute),
			procTime:   now,
			tolerance:  tolerance,
			wantIsLate: true,
		},
		{
			name:       "very late event",
			eventTime:  now.Add(-10 * time.Minute),
			procTime:   now,
			tolerance:  tolerance,
			wantIsLate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &TelemetryEvent{
				TimestampMs: tt.eventTime.UnixMilli(),
			}
			if got := event.IsLate(tt.procTime, tt.tolerance); got != tt.wantIsLate {
				t.Errorf("TelemetryEvent.IsLate() = %v, want %v", got, tt.wantIsLate)
			}
		})
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}
