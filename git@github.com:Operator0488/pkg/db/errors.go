package db

import "fmt"

// Custom errors
var (
	ErrConnectionFailed  = fmt.Errorf("postgres connection failed")
	ErrNotConnected      = fmt.Errorf("postgres not connected")
	ErrHealthCheckFailed = fmt.Errorf("postgres health check failed")
	ErrNestedTransaction = fmt.Errorf("nested transactions are not allowed")
)
