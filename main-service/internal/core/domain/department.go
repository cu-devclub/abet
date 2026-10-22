package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Ref is a compact id/code/name reference to a department or curriculum.
type Ref struct {
	ID   uuid.UUID
	Code string
	Name string
}

// Department groups curricula.
type Department struct {
	ID              uuid.UUID
	Code            string
	Name            string
	CurriculumCount int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// NewDepartment validates and builds a department (code trimmed + upper-cased).
func NewDepartment(code, name string) (*Department, error) {
	code, name, err := normalizeCodeName(code, name)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	return &Department{ID: uuid.New(), Code: code, Name: name, CreatedAt: now, UpdatedAt: now}, nil
}

// Apply changes code/name (nil = keep) with the same validation as NewDepartment.
func (d *Department) Apply(code, name *string) error {
	c, n := d.Code, d.Name
	if code != nil {
		c = *code
	}
	if name != nil {
		n = *name
	}
	c, n, err := normalizeCodeName(c, n)
	if err != nil {
		return err
	}
	d.Code, d.Name, d.UpdatedAt = c, n, time.Now().UTC()
	return nil
}

// normalizeCodeName trims both values, upper-cases the code and rejects empties.
func normalizeCodeName(code, name string) (string, string, error) {
	code, name = strings.ToUpper(strings.TrimSpace(code)), strings.TrimSpace(name)
	if code == "" {
		return "", "", fmt.Errorf("%w: code cannot be empty", ErrInvalidInput)
	}
	if name == "" {
		return "", "", fmt.Errorf("%w: name cannot be empty", ErrInvalidInput)
	}
	return code, name, nil
}
