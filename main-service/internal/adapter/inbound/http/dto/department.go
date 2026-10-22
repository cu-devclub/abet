package dto

import (
	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

type CreateDepartmentRequest struct {
	Code string `json:"code" validate:"required,max=50"`
	Name string `json:"name" validate:"required,max=200"`
}

func (r CreateDepartmentRequest) Validate() error { return Validate(r) }

func (r CreateDepartmentRequest) ToInput() port.CreateDepartmentInput {
	return port.CreateDepartmentInput{Code: r.Code, Name: r.Name}
}

type UpdateDepartmentRequest struct {
	Code *string `json:"code" validate:"omitempty,max=50"`
	Name *string `json:"name" validate:"omitempty,max=200"`
}

func (r UpdateDepartmentRequest) Validate() error { return Validate(r) }

func (r UpdateDepartmentRequest) ToInput(id uuid.UUID) port.UpdateDepartmentInput {
	return port.UpdateDepartmentInput{ID: id, Code: r.Code, Name: r.Name}
}

type DepartmentResponse struct {
	ID              uuid.UUID `json:"id"`
	Code            string    `json:"code"`
	Name            string    `json:"name"`
	CurriculumCount int       `json:"curriculum_count"`
	CreatedAt       string    `json:"created_at"`
	UpdatedAt       string    `json:"updated_at"`
}

func DepartmentFromDomain(d *domain.Department) DepartmentResponse {
	return DepartmentResponse{
		ID: d.ID, Code: d.Code, Name: d.Name, CurriculumCount: d.CurriculumCount,
		CreatedAt: FormatTime(d.CreatedAt), UpdatedAt: FormatTime(d.UpdatedAt),
	}
}

func DepartmentsFromDomain(ds []domain.Department) []DepartmentResponse {
	out := make([]DepartmentResponse, 0, len(ds))
	for i := range ds {
		out = append(out, DepartmentFromDomain(&ds[i]))
	}
	return out
}
