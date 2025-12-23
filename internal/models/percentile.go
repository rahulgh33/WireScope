package models

import (
	"sort"
)

// calculatePercentile computes the exact percentile from a slice of float64 values.
// The percentile parameter should be between 0 and 100.
//
// This implementation uses the nearest-rank method:
// - Sorts the input data
// - Finds the index at the percentile position
// - Returns the value at that index
//
// Requirement: 4.2 - MVP exact percentile calculation from full sample set
func calculatePercentile(data []float64, percentile float64) float64 {
	if len(data) == 0 {
		return 0
	}

	// Make a copy to avoid modifying the original slice
	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)

	// Calculate the index using nearest-rank method
	// For P50 with 10 samples: floor((50/100) * (10-1)) = floor(4.5) = 4 (0-based index for 5th element)
	// For P95 with 10 samples: floor((95/100) * (10-1)) = floor(8.55) = 8 (0-based index for 9th element)
	// Using floor of (percentile/100 * (len-1)) for proper distribution
	index := int(float64(len(sorted)-1) * percentile / 100.0)

	// Handle edge cases
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}

// downsampleUniform performs uniform downsampling when sample size exceeds limit.
// This maintains statistical properties while reducing memory usage.
//
// Requirement: 4.2 - Uniform downsampling for windows exceeding sample limits
func downsampleUniform(data []float64, targetSize int) []float64 {
	if len(data) <= targetSize {
		return data
	}

	// Calculate step size for uniform sampling
	step := float64(len(data)) / float64(targetSize)
	result := make([]float64, targetSize)

	for i := 0; i < targetSize; i++ {
		index := int(float64(i) * step)
		result[i] = data[index]
	}

	return result
}
