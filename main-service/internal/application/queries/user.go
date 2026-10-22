package queries

import (
	"context"
	"fmt"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

// ListUsersHandler implements port.ListUsersUseCase: super_admin sees all, admin only users in scope.
type ListUsersHandler struct {
	repo port.UserRepository
}

var _ port.ListUsersUseCase = (*ListUsersHandler)(nil)

func NewListUsersHandler(repo port.UserRepository) *ListUsersHandler {
	return &ListUsersHandler{repo: repo}
}

func (h *ListUsersHandler) Execute(ctx context.Context, actor *domain.Actor, in port.ListUsersInput) ([]domain.User, error) {
	if !actor.CanManageUsers() {
		return nil, domain.ErrForbidden
	}
	if in.Role != nil && !in.Role.Valid() {
		return nil, fmt.Errorf("%w: unknown role filter", domain.ErrInvalidInput)
	}
	all, err := h.repo.List(ctx, port.UserFilter{Search: in.Search, Role: in.Role, DepartmentID: in.DepartmentID})
	if err != nil {
		return nil, err
	}
	visible := make([]domain.User, 0, len(all))
	for _, u := range all {
		if actor.CanSeeUser(u.ScopeInfo()) {
			visible = append(visible, u)
		}
	}
	return visible, nil
}

// GetMeHandler implements port.GetMeUseCase.
type GetMeHandler struct {
	repo port.UserRepository
}

var _ port.GetMeUseCase = (*GetMeHandler)(nil)

func NewGetMeHandler(repo port.UserRepository) *GetMeHandler {
	return &GetMeHandler{repo: repo}
}

func (h *GetMeHandler) Execute(ctx context.Context, actor *domain.Actor) (*port.MeResult, error) {
	u, err := h.repo.GetByID(ctx, actor.UserID)
	if err != nil {
		return nil, err
	}
	privileged := actor.CanManageUsers()
	return &port.MeResult{
		User:          u,
		ManageOrg:     actor.CanManageOrg(),
		ManageUsers:   privileged,
		ManageCourses: actor.CanManageCoursesAny(),
		ManageCovers:  actor.CanManageCoursesAny(),
	}, nil
}
