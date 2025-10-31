package wazevoapi

// ResetMap resets the map to an empty state, or creates a new map if it is nil.
// If the map is nil, it is created with the specified capacity hint.
func ResetMap[K comparable, V any](m map[K]V, capacityHint int) map[K]V {
	if m == nil {
		m = make(map[K]V, capacityHint)
	} else {
		clear(m)
	}
	return m
}

// ResetSlice resets the slice to empty, or creates a new slice with the specified capacity if it is nil.
func ResetSlice[T any](s []T, capacityHint int) []T {
	if s == nil {
		return make([]T, 0, capacityHint)
	}
	return s[:0]
}
