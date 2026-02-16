package stats_bot

import (
	"sort"
)

// CalculateMedian returns the median of the given values.
func CalculateMedian(values []int) int {
	if len(values) == 0 {
		return 0
	}

	sorted := make([]int, len(values))
	copy(sorted, values)
	sort.Ints(sorted)

	mid := len(sorted) / 2
	if len(sorted)%2 == 0 && mid > 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}
