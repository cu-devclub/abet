package domain

import (
	"time"

	"github.com/google/uuid"
)

// Cover describes the optional cover PDF of a curriculum (the file itself lives in storage).
type Cover struct {
	Filename  string
	SizeBytes int64
	UpdatedAt time.Time
}

// Curriculum belongs to exactly one department.
type Curriculum struct {
	ID           uuid.UUID
	Code         string
	Name         string
	DepartmentID uuid.UUID
	Department   Ref
	Cover        *Cover
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewCurriculum validates and builds a curriculum (code trimmed + upper-cased).
func NewCurriculum(code, name string, departmentID uuid.UUID) (*Curriculum, error) {
	code, name, err := normalizeCodeName(code, name)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	return &Curriculum{ID: uuid.New(), Code: code, Name: name, DepartmentID: departmentID, CreatedAt: now, UpdatedAt: now}, nil
}

// Apply changes code/name/department (nil = keep).
func (c *Curriculum) Apply(code, name *string, departmentID *uuid.UUID) error {
	cd, n := c.Code, c.Name
	if code != nil {
		cd = *code
	}
	if name != nil {
		n = *name
	}
	cd, n, err := normalizeCodeName(cd, n)
	if err != nil {
		return err
	}
	c.Code, c.Name, c.UpdatedAt = cd, n, time.Now().UTC()
	if departmentID != nil {
		c.DepartmentID = *departmentID
	}
	return nil
}
