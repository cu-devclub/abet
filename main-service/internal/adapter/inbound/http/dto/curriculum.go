package dto

import (
	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

type CreateCurriculumRequest struct {
	Code         string    `json:"code" validate:"required,max=50"`
	Name         string    `json:"name" validate:"required,max=200"`
	DepartmentID uuid.UUID `json:"department_id" validate:"required"`
}

func (r CreateCurriculumRequest) Validate() error { return Validate(r) }

func (r CreateCurriculumRequest) ToInput() port.CreateCurriculumInput {
	return port.CreateCurriculumInput{Code: r.Code, Name: r.Name, DepartmentID: r.DepartmentID}
}

type UpdateCurriculumRequest struct {
	Code         *string    `json:"code" validate:"omitempty,max=50"`
	Name         *string    `json:"name" validate:"omitempty,max=200"`
	DepartmentID *uuid.UUID `json:"department_id"`
}

func (r UpdateCurriculumRequest) Validate() error { return Validate(r) }

func (r UpdateCurriculumRequest) ToInput(id uuid.UUID) port.UpdateCurriculumInput {
	return port.UpdateCurriculumInput{ID: id, Code: r.Code, Name: r.Name, DepartmentID: r.DepartmentID}
}

type CoverResponse struct {
	Filename  string `json:"filename"`
	SizeBytes int64  `json:"size_bytes"`
	UpdatedAt string `json:"updated_at"`
}

type CurriculumResponse struct {
	ID         uuid.UUID      `json:"id"`
	Code       string         `json:"code"`
	Name       string         `json:"name"`
	Department RefResponse    `json:"department"`
	Cover      *CoverResponse `json:"cover"`
	CreatedAt  string         `json:"created_at"`
	UpdatedAt  string         `json:"updated_at"`
}

func CurriculumFromDomain(c *domain.Curriculum) CurriculumResponse {
	out := CurriculumResponse{
		ID: c.ID, Code: c.Code, Name: c.Name, Department: RefFromDomain(c.Department),
		CreatedAt: FormatTime(c.CreatedAt), UpdatedAt: FormatTime(c.UpdatedAt),
	}
	if c.Cover != nil {
		out.Cover = &CoverResponse{Filename: c.Cover.Filename, SizeBytes: c.Cover.SizeBytes, UpdatedAt: FormatTime(c.Cover.UpdatedAt)}
	}
	return out
}

func CurriculaFromDomain(cs []domain.Curriculum) []CurriculumResponse {
	out := make([]CurriculumResponse, 0, len(cs))
	for i := range cs {
		out = append(out, CurriculumFromDomain(&cs[i]))
	}
	return out
}
