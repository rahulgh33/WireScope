package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TelemetryEvent represents a structured network performance measurement event
// with schema versioning for evolution support.
//
// Requirements: 2.1, 2.2, 2.3, 2.4, 2.5
type TelemetryEvent struct {
	// EventID uniquely identifies this event (UUID format)
	EventID string `json:"event_id"`

	// ClientID is a stable identifier for the probe agent
	ClientID string `json:"client_id"`

	// TimestampMs is the event timestamp in milliseconds since epoch
	TimestampMs int64 `json:"ts_ms"`

	// RecvTimestampMs is set by the ingest service for clock skew debugging
	RecvTimestampMs *int64 `json:"recv_ts_ms,omitempty"`

	// SchemaVersion indicates the event structure version for backward compatibility
	SchemaVersion string `json:"schema_version"`

	// Target is the endpoint being measured (e.g., "https://example.com")
	Target string `json:"target"`

	// NetworkContext provides additional context about the network environment
	NetworkContext NetworkContext `json:"network_context"`

	// Timings contains the detailed network timing measurements
	Timings TimingMeasurements `json:"timings"`

	// ThroughputKbps is the measured throughput in kilobits per second
	ThroughputKbps float64 `json:"throughput_kbps"`

	// ErrorStage indicates which stage failed (if any): DNS, TCP, TLS, HTTP, or throughput
	ErrorStage *string `json:"error_stage,omitempty"`
}

// NetworkContext provides additional context about the network environment
// where the measurement was taken.
//
// Requirements: 2.5
type NetworkContext struct {
	// InterfaceType describes the network interface (e.g., "wifi", "ethernet", "cellular")
	InterfaceType string `json:"interface_type"`

	// VPNEnabled indicates whether a VPN connection was active
	VPNEnabled bool `json:"vpn_enabled"`

	// UserLabel is an optional custom label for user-defined categorization
	UserLabel *string `json:"user_label,omitempty"`
}

// TimingMeasurements contains detailed network timing measurements in milliseconds.
//
// Requirements: 1.1, 1.2, 1.3, 1.4
type TimingMeasurements struct {
	// DNSMs is DNS resolution time in milliseconds
	DNSMs float64 `json:"dns_ms"`

	// TCPMs is TCP connection establishment time in milliseconds
	TCPMs float64 `json:"tcp_ms"`

	// TLSMs is TLS handshake time in milliseconds
	TLSMs float64 `json:"tls_ms"`

	// HTTPTTFBMs is HTTP time-to-first-byte in milliseconds
	HTTPTTFBMs float64 `json:"http_ttfb_ms"`
}

// Validate checks if the TelemetryEvent has valid data.
//
// Returns an error if any required field is missing or invalid.
func (e *TelemetryEvent) Validate() error {
	// Validate EventID is a valid UUID
	if _, err := uuid.Parse(e.EventID); err != nil {
		return fmt.Errorf("invalid event_id: must be a valid UUID: %w", err)
	}

	// Validate ClientID is not empty
	if e.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}

	// Validate TimestampMs is reasonable (not zero and not in the far future)
	if e.TimestampMs <= 0 {
		return fmt.Errorf("ts_ms must be positive")
	}
	now := time.Now().Unix() * 1000
	if e.TimestampMs > now+3600000 { // Allow 1 hour in the future for clock skew
		return fmt.Errorf("ts_ms is too far in the future")
	}

	// Validate SchemaVersion is not empty
	if e.SchemaVersion == "" {
		return fmt.Errorf("schema_version is required")
	}

	// Validate Target is not empty
	if e.Target == "" {
		return fmt.Errorf("target is required")
	}

	// Validate NetworkContext
	if err := e.NetworkContext.Validate(); err != nil {
		return fmt.Errorf("invalid network_context: %w", err)
	}

	// Validate Timings (only if no error occurred)
	if e.ErrorStage == nil {
		if err := e.Timings.Validate(); err != nil {
			return fmt.Errorf("invalid timings: %w", err)
		}

		// Validate ThroughputKbps is non-negative
		if e.ThroughputKbps < 0 {
			return fmt.Errorf("throughput_kbps must be non-negative")
		}
	}

	return nil
}

// Validate checks if the NetworkContext has valid data.
func (nc *NetworkContext) Validate() error {
	if nc.InterfaceType == "" {
		return fmt.Errorf("interface_type is required")
	}
	return nil
}

// Validate checks if the TimingMeasurements have valid data.
func (tm *TimingMeasurements) Validate() error {
	if tm.DNSMs < 0 {
		return fmt.Errorf("dns_ms must be non-negative")
	}
	if tm.TCPMs < 0 {
		return fmt.Errorf("tcp_ms must be non-negative")
	}
	if tm.TLSMs < 0 {
		return fmt.Errorf("tls_ms must be non-negative")
	}
	if tm.HTTPTTFBMs < 0 {
		return fmt.Errorf("http_ttfb_ms must be non-negative")
	}
	return nil
}

// GetWindowStartMs returns the window start timestamp for this event.
// Windows are 1-minute (60000ms) aligned to the epoch.
//
// Requirement: 4.1
func (e *TelemetryEvent) GetWindowStartMs() int64 {
	return (e.TimestampMs / 60000) * 60000
}

// GetWindowStart returns the window start time for this event.
func (e *TelemetryEvent) GetWindowStart() time.Time {
	return time.UnixMilli(e.GetWindowStartMs())
}

// IsLate checks if the event is too late to be processed based on a lateness tolerance.
// Events older than (processingTime - latenessTolerance) are considered late.
// Events exactly at the cutoff boundary are NOT considered late.
//
// Requirement: 3.4
func (e *TelemetryEvent) IsLate(processingTime time.Time, latenessTolerance time.Duration) bool {
	eventTime := time.UnixMilli(e.TimestampMs)
	cutoff := processingTime.Add(-latenessTolerance)
	// Use Before (not BeforeOrEqual) so events at the boundary are not late
	return eventTime.Before(cutoff)
}
