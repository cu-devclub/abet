package domain

import "errors"

var (
	// ErrNotFound is returned when a requested entity cannot be found.
	ErrNotFound = errors.New("entity not found")

	// ErrInvalidInput is returned when validation fails.
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized is returned when the caller is not authenticated.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden is returned when the caller is authenticated but not allowed to perform the action.
	ErrForbidden = errors.New("forbidden")

	// ErrConflict is returned when an entity already exists or is still referenced.
	ErrConflict = errors.New("conflict")

	// ErrTooLarge is returned when an upload exceeds the configured size limit.
	ErrTooLarge = errors.New("payload too large")

	// ErrUnprocessable is returned for semantically invalid content (e.g. a bad CSV).
	ErrUnprocessable = errors.New("unprocessable")

	// ErrServiceUnavailable is returned when a downstream dependency fails.
	ErrServiceUnavailable = errors.New("service unavailable")
)
