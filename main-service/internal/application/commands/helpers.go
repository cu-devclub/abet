package commands

import (
	"errors"
	"fmt"

	"abet/main-service/internal/core/domain"
)

// notFoundAs rewrites a repository ErrNotFound into a not-found error with a user-facing message; other errors pass through.
func notFoundAs(err error, msg string) error {
	if errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("%w: %s", domain.ErrNotFound, msg)
	}
	return err
}

// invalidIfNotFound turns a missing referenced entity (e.g. an unknown department_id in a body) into a 400.
func invalidIfNotFound(err error, msg string) error {
	if errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("%w: %s", domain.ErrInvalidInput, msg)
	}
	return err
}
