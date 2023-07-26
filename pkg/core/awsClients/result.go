package awsClients

type Result[T any] struct {
	Data T
	Err  error
}
