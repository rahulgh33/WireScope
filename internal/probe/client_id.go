package probe

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

const (
	clientIDFile = ".telemetry_client_id"
)

// GetOrCreateClientID returns a stable client ID, creating one if it doesn't exist
//
// Requirement: 2.2 - Stable client_id stored locally by probe
func GetOrCreateClientID() (string, error) {
	// Try to get from environment first (for containerized deployments)
	if clientID := os.Getenv("TELEMETRY_CLIENT_ID"); clientID != "" {
		return clientID, nil
	}

	// Determine storage location
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	clientIDPath := filepath.Join(homeDir, clientIDFile)

	// Try to read existing client ID
	if data, err := os.ReadFile(clientIDPath); err == nil {
		clientID := string(data)
		if len(clientID) > 0 {
			return clientID, nil
		}
	}

	// Generate new client ID
	clientID, err := generateClientID()
	if err != nil {
		return "", fmt.Errorf("failed to generate client ID: %w", err)
	}

	// Save to file
	err = os.WriteFile(clientIDPath, []byte(clientID), 0600)
	if err != nil {
		return "", fmt.Errorf("failed to save client ID: %w", err)
	}

	return clientID, nil
}

// generateClientID creates a new random client ID
func generateClientID() (string, error) {
	// Generate 16 random bytes
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Convert to hex string
	return "probe-" + hex.EncodeToString(bytes), nil
}
