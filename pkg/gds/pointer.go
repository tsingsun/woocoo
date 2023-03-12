package gds

// Ptr returns a pointer to the given value.
func Ptr[T any](i T) *T {
	return &i
}
