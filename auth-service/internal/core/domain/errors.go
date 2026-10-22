package domain

import "errors"

var (
	// ErrNotFound is returned when a requested domain entity cannot be found.
	ErrNotFound = errors.New("entity not found")

	// ErrInvalidInput is returned when domain validation fails.
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized is returned when the caller is not authenticated.
	ErrUnauthorized = errors.New("unauthenticated")

	// ErrForbidden is returned when the caller is authenticated but not allowed.
	ErrForbidden = errors.New("forbidden")

	// ErrConflict is returned when an entity already exists.
	ErrConflict = errors.New("conflict: entity already exists")

	// ErrServiceUnavailable is returned when a downstream external service or DB fails.
	ErrServiceUnavailable = errors.New("service unavailable")

	// ErrCircuitOpen is returned when the circuit breaker is open.
	ErrCircuitOpen = errors.New("circuit breaker is open: downstream service unavailable")

	// ErrOAuthNotConfigured is returned when Google OAuth credentials are not set.
	ErrOAuthNotConfigured = errors.New("google oauth not configured")
)
