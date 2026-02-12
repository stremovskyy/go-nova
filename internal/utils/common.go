package utils

func Ref[T any](value T) *T {
	return &value
}

func Ptr[T any](value *T) T {
	if value == nil {
		return *new(T)
	}
	return *value
}
