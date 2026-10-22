package queries

import (
	"context"
	"fmt"
	"sort"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

const maxActAsUsers = 500

// roleOrder is the display order of the act-as list.
var roleOrder = map[domain.Role]int{
	domain.RoleSuperAdmin: 0, domain.RoleAdmin: 1, domain.RoleInstructor: 2, domain.RoleStaff: 3,
}

// ListActAsUsersHandler implements port.ListActAsUsersUseCase.
type ListActAsUsersHandler struct {
	repo port.UserRepository
}

var _ port.ListActAsUsersUseCase = (*ListActAsUsersHandler)(nil)

func NewListActAsUsersHandler(repo port.UserRepository) *ListActAsUsersHandler {
	return &ListActAsUsersHandler{repo: repo}
}

func (h *ListActAsUsersHandler) Execute(ctx context.Context, real *domain.Actor, in port.ListActAsUsersInput) ([]domain.User, error) {
	if real == nil || real.Role != domain.RoleSuperAdmin {
		return nil, domain.ErrForbidden
	}
	if in.Role != nil && !in.Role.Valid() {
		return nil, fmt.Errorf("%w: unknown role filter", domain.ErrInvalidInput)
	}
	all, err := h.repo.List(ctx, port.UserFilter{Search: in.Search, Role: in.Role})
	if err != nil {
		return nil, err
	}
	out := make([]domain.User, 0, len(all))
	for _, u := range all {
		if u.ID != real.UserID {
			out = append(out, u)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if ri, rj := roleOrder[out[i].Role], roleOrder[out[j].Role]; ri != rj {
			return ri < rj
		}
		return out[i].Email < out[j].Email
	})
	if len(out) > maxActAsUsers {
		out = out[:maxActAsUsers]
	}
	return out, nil
}

// GetActAsStateHandler implements port.GetActAsStateUseCase.
type GetActAsStateHandler struct {
	repo    port.UserRepository
	enabled bool
}

var _ port.GetActAsStateUseCase = (*GetActAsStateHandler)(nil)

// NewGetActAsStateHandler: enabled is the resolved feature flag (DEV_ACT_AS_ENABLED and APP_ENV=development).
func NewGetActAsStateHandler(repo port.UserRepository, enabled bool) *GetActAsStateHandler {
	return &GetActAsStateHandler{repo: repo, enabled: enabled}
}

func (h *GetActAsStateHandler) Execute(ctx context.Context, real, effective *domain.Actor) (*port.ActAsState, error) {
	state := &port.ActAsState{Available: h.enabled && real != nil && real.Role == domain.RoleSuperAdmin}
	if !state.Available || effective == nil || effective.UserID == real.UserID {
		return state, nil
	}
	u, err := h.repo.GetByID(ctx, real.UserID)
	if err != nil {
		return nil, err
	}
	state.RealUser = u
	return state, nil
}
