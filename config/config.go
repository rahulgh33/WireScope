package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the application configuration
type Config struct {
	Database DatabaseConfig
	NATS     NATSConfig
	HTTP     HTTPConfig
	Metrics  MetricsConfig
	Tracing  TracingConfig
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// NATSConfig holds NATS connection settings
type NATSConfig struct {
	URL     string
	Stream  string
	Subject string
}

// HTTPConfig holds HTTP server settings
type HTTPConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// MetricsConfig holds metrics settings
type MetricsConfig struct {
	Port int
	Path string
}

// TracingConfig holds tracing settings
type TracingConfig struct {
	Enabled     bool
	ServiceName string
	Endpoint    string
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "telemetry"),
			Password: getEnv("DB_PASSWORD", "telemetry"),
			Database: getEnv("DB_NAME", "telemetry"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		NATS: NATSConfig{
			URL:     getEnv("NATS_URL", "nats://localhost:4222"),
			Stream:  getEnv("NATS_STREAM", "telemetry-events"),
			Subject: getEnv("NATS_SUBJECT", "telemetry.events.*"),
		},
		HTTP: HTTPConfig{
			Port:         getEnvInt("HTTP_PORT", 8080),
			ReadTimeout:  getEnvDuration("HTTP_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getEnvDuration("HTTP_WRITE_TIMEOUT", 30*time.Second),
		},
		Metrics: MetricsConfig{
			Port: getEnvInt("METRICS_PORT", 9090),
			Path: getEnv("METRICS_PATH", "/metrics"),
		},
		Tracing: TracingConfig{
			Enabled:     getEnvBool("TRACING_ENABLED", true),
			ServiceName: getEnv("SERVICE_NAME", "wirescope"),
			Endpoint:    getEnv("JAEGER_ENDPOINT", "http://localhost:4318/v1/traces"),
		},
	}
}

// Helper functions for environment variable parsing
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
