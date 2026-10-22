package queries

import (
	"context"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

// ListDepartmentsHandler implements port.ListDepartmentsUseCase (any authenticated caller).
// super_admin and admin see every department; instructor and staff only their own.
type ListDepartmentsHandler struct {
	repo port.DepartmentRepository
}

var _ port.ListDepartmentsUseCase = (*ListDepartmentsHandler)(nil)

func NewListDepartmentsHandler(repo port.DepartmentRepository) *ListDepartmentsHandler {
	return &ListDepartmentsHandler{repo: repo}
}

func (h *ListDepartmentsHandler) Execute(ctx context.Context, actor *domain.Actor) ([]domain.Department, error) {
	if seesAllOrganisation(actor) {
		return h.repo.List(ctx)
	}
	if actor.DepartmentID == nil {
		return []domain.Department{}, nil
	}
	all, err := h.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := []domain.Department{}
	for _, d := range all {
		if d.ID == *actor.DepartmentID {
			out = append(out, d)
		}
	}
	return out, nil
}

// seesAllOrganisation is true for the roles that may list every department and curriculum.
func seesAllOrganisation(actor *domain.Actor) bool {
	return actor.Role == domain.RoleSuperAdmin || actor.Role == domain.RoleAdmin
}
