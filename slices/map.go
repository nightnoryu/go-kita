package slices

// Map iterates through slice and maps values
func Map[T, TResult any](s []T, f func(T) TResult) []TResult {
	result := make([]TResult, 0, len(s))
	for _, t := range s {
		result = append(result, f(t))
	}
	return result
}

// MapErr iterates through slice, maps values and stops on error
func MapErr[T, TResult any](s []T, f func(T) (TResult, error)) ([]TResult, error) {
	result := make([]TResult, 0, len(s))
	for _, t := range s {
		e, err := f(t)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}
