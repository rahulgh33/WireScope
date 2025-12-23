package database

import (
	"context"
	"testing"
	"time"
)

func TestDefaultConnectionConfig(t *testing.T) {
	config := DefaultConnectionConfig()

	if config.Host != "localhost" {
		t.Errorf("Expected host 'localhost', got '%s'", config.Host)
	}

	if config.Port != 5432 {
		t.Errorf("Expected port 5432, got %d", config.Port)
	}

	if config.MaxOpenConns != 25 {
		t.Errorf("Expected MaxOpenConns 25, got %d", config.MaxOpenConns)
	}

	if config.MaxIdleConns != 5 {
		t.Errorf("Expected MaxIdleConns 5, got %d", config.MaxIdleConns)
	}

	if config.ConnMaxLifetime != time.Hour {
		t.Errorf("Expected ConnMaxLifetime 1h, got %v", config.ConnMaxLifetime)
	}
}

func TestConnectionConfigValidation(t *testing.T) {
	// Test with invalid config (should fail to connect)
	config := &ConnectionConfig{
		Host:            "nonexistent-host",
		Port:            9999,
		User:            "invalid",
		Password:        "invalid",
		Database:        "invalid",
		SSLMode:         "disable",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
		ConnMaxIdleTime: time.Second * 30,
	}

	conn, err := NewConnection(config)
	if err == nil {
		conn.Close()
		t.Error("Expected connection to fail with invalid config, but it succeeded")
	}
}

func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-connection error",
			err:      context.DeadlineExceeded,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsConnectionError(tt.err)
			if result != tt.expected {
				t.Errorf("IsConnectionError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-retryable error",
			err:      context.Canceled,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryableError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}