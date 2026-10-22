// Package dto holds the JSON request/response shapes of the HTTP API (snake_case, RFC3339 UTC timestamps).
package dto

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
)

var (
	validate     *validator.Validate
	validateOnce sync.Once
)

// Validate runs struct-tag validation and wraps failures in domain.ErrInvalidInput.
func Validate(v any) error {
	validateOnce.Do(func() { validate = validator.New() })
	if err := validate.Struct(v); err != nil {
		return fmt.Errorf("%w: validation failed: %s", domain.ErrInvalidInput, err.Error())
	}
	return nil
}

// ErrorResponse is the body of every non-2xx answer.
type ErrorResponse struct {
	Error string `json:"error"`
}

// ItemsResponse wraps list answers: {"items": [...]}.
type ItemsResponse[T any] struct {
	Items []T `json:"items"`
}

// Items builds an ItemsResponse that serialises an empty slice as [] (never null).
func Items[T any](items []T) ItemsResponse[T] {
	if items == nil {
		items = []T{}
	}
	return ItemsResponse[T]{Items: items}
}

// RefResponse is a compact {id, code, name} reference.
type RefResponse struct {
	ID   uuid.UUID `json:"id"`
	Code string    `json:"code"`
	Name string    `json:"name"`
}

// RefFromDomain maps a domain.Ref.
func RefFromDomain(r domain.Ref) RefResponse {
	return RefResponse{ID: r.ID, Code: r.Code, Name: r.Name}
}

func refPtr(r *domain.Ref) *RefResponse {
	if r == nil {
		return nil
	}
	v := RefFromDomain(*r)
	return &v
}

// FormatTime renders a timestamp as RFC3339 UTC.
func FormatTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := FormatTime(*t)
	return &s
}
