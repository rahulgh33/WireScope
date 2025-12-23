package models

import (
	"sort"
)

// CalculatePercentiles computes both P50 and P95 percentiles from a slice of float64 values.
// Returns (p50, p95) values.
func CalculatePercentiles(data []float64) (float64, float64) {
	if len(data) == 0 {
		return 0, 0
	}
	return calculatePercentile(data, 50), calculatePercentile(data, 95)
}

// calculatePercentile computes the exact percentile from a slice of float64 values.
// The percentile parameter should be between 0 and 100.
//
// This implementation uses linear interpolation method:
// - Sorts the input data
// - Calculates the exact position using (percentile/100) * (N-1)
// - Interpolates between adjacent values if position is not an integer
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

	// Calculate the exact position using linear interpolation method
	// For P50 with 2 samples: 0.5 * (2-1) = 0.5 → interpolate between index 0 and 1
	// For P95 with 2 samples: 0.95 * (2-1) = 0.95 → interpolate between index 0 and 1
	position := (percentile / 100.0) * float64(len(sorted)-1)
	
	// Get the lower and upper indices
	lowerIndex := int(position)
	upperIndex := lowerIndex + 1
	
	// Handle edge cases
	if lowerIndex < 0 {
		lowerIndex = 0
	}
	if upperIndex >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	
	// Linear interpolation
	fraction := position - float64(lowerIndex)
	return sorted[lowerIndex] + fraction*(sorted[upperIndex]-sorted[lowerIndex])
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
