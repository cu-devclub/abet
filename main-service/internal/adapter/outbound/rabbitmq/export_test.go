package rabbitmq

import "time"

// Backoff exposes the reconnect delay for tests.
func Backoff(attempt int) time.Duration { return backoff(attempt) }
