package utils

// Result struct can be used for passing data on channels
type Result[T any] struct {
	Data T
	Err  error
}
