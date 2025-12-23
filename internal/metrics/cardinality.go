package metrics

import (
	"crypto/sha256"
	"fmt"
)

// HashLabel creates a short hash of a label value to reduce cardinality
// while maintaining uniqueness for monitoring purposes.
//
// Returns first 8 characters of SHA256 hash.
//
// Requirement: 6.1 - Cardinality management for high-cardinality labels
func HashLabel(value string) string {
	if value == "" {
		return "unknown"
	}

	hash := sha256.Sum256([]byte(value))
	// Use first 8 hex characters (4 bytes) for manageable cardinality
	return fmt.Sprintf("%x", hash[:4])
}

// HashClientID creates a hashed version of client_id for metrics
func HashClientID(clientID string) string {
	return HashLabel(clientID)
}

// HashTarget creates a hashed version of target URL for metrics
func HashTarget(target string) string {
	return HashLabel(target)
}
