package gds

// Ptr returns a pointer to the given value.
func Ptr[T any](i T) *T {
	return &i
}

// IIF returns the trueValue if condition is true, otherwise returns falseValue.
func IIF[T comparable](condition bool, trueValue, falseValue T) T {
	if condition {
		return trueValue
	}
	return falseValue
}
