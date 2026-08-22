package slices

// Map iterates through slice and maps values
func Map[T, TResult any](s []T, f func(T) TResult) []TResult {
	result := make([]TResult, 0, len(s))
	for _, t := range s {
		result = append(result, f(t))
	}
	return result
}
