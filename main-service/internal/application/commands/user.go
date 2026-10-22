package commands

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

// orgResolver validates department/curriculum/scope ids against the organisation repositories.
type orgResolver struct {
	depts port.DepartmentRepository
	currs port.CurriculumRepository
}

// resolvePlacement checks that the ids exist and are consistent. A curriculum implies its department
// (auto-filled when the department is nil, must match otherwise).
func (o orgResolver) resolvePlacement(ctx context.Context, deptID, currID *uuid.UUID) (*uuid.UUID, *uuid.UUID, error) {
	if currID != nil {
		c, err := o.currs.GetByID(ctx, *currID)
		if err != nil {
			return nil, nil, invalidIfNotFound(err, "curriculum not found")
		}
		if deptID == nil {
			id := c.DepartmentID
			deptID = &id
		} else if *deptID != c.DepartmentID {
			return nil, nil, fmt.Errorf("%w: curriculum does not belong to the department", domain.ErrInvalidInput)
		}
	}
	if deptID != nil {
		if _, err := o.depts.GetByID(ctx, *deptID); err != nil {
			return nil, nil, invalidIfNotFound(err, "department not found")
		}
	}
	return deptID, currID, nil
}

// resolveScope turns scope ids into refs (deduplicated); unknown ids are a 400.
func (o orgResolver) resolveScope(ctx context.Context, in *port.AdminScopeInput) (*domain.AdminScope, error) {
	scope := &domain.AdminScope{Departments: []domain.Ref{}, Curricula: []domain.Ref{}}
	if in == nil {
		return scope, nil
	}
	seen := map[uuid.UUID]bool{}
	for _, id := range in.DepartmentIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		d, err := o.depts.GetByID(ctx, id)
		if err != nil {
			return nil, invalidIfNotFound(err, "admin_scope department not found")
		}
		scope.Departments = append(scope.Departments, domain.Ref{ID: d.ID, Code: d.Code, Name: d.Name})
	}
	seen = map[uuid.UUID]bool{}
	for _, id := range in.CurriculumIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		c, err := o.currs.GetByID(ctx, id)
		if err != nil {
			return nil, invalidIfNotFound(err, "admin_scope curriculum not found")
		}
		scope.Curricula = append(scope.Curricula, domain.Ref{ID: c.ID, Code: c.Code, Name: c.Name})
	}
	return scope, nil
}

// CreateUserHandler implements port.CreateUserUseCase.
type CreateUserHandler struct {
	users          port.UserRepository
	org            orgResolver
	allowedDomains []string
}

var _ port.CreateUserUseCase = (*CreateUserHandler)(nil)

func NewCreateUserHandler(users port.UserRepository, depts port.DepartmentRepository, currs port.CurriculumRepository, allowedDomains []string) *CreateUserHandler {
	return &CreateUserHandler{users: users, org: orgResolver{depts: depts, currs: currs}, allowedDomains: allowedDomains}
}

func (h *CreateUserHandler) Execute(ctx context.Context, actor *domain.Actor, in port.CreateUserInput) (*domain.User, error) {
	if !actor.CanManageUsers() {
		return nil, domain.ErrForbidden
	}
	email, err := domain.ValidateEmail(in.Email, h.allowedDomains)
	if err != nil {
		return nil, err
	}
	if !in.Role.Valid() {
		return nil, fmt.Errorf("%w: role must be one of super_admin, admin, instructor, staff", domain.ErrInvalidInput)
	}
	if !actor.CanAssignRole(in.Role) {
		return nil, domain.ErrForbidden
	}
	if in.AdminScope != nil {
		if in.Role != domain.RoleAdmin {
			return nil, fmt.Errorf("%w: admin_scope is only valid for role admin", domain.ErrInvalidInput)
		}
		if !actor.CanSetAdminScope() {
			return nil, domain.ErrForbidden
		}
	}
	deptID, currID, err := h.org.resolvePlacement(ctx, in.DepartmentID, in.CurriculumID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	u := &domain.User{
		ID: uuid.New(), Email: email, Role: in.Role, DepartmentID: deptID, CurriculumID: currID,
		CreatedAt: now, UpdatedAt: now,
	}
	if in.Role == domain.RoleAdmin {
		if u.AdminScope, err = h.org.resolveScope(ctx, in.AdminScope); err != nil {
			return nil, err
		}
	}
	if !actor.CanManageUser(u.ScopeInfo()) {
		return nil, domain.ErrForbidden
	}
	if err := h.users.Create(ctx, u); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return nil, fmt.Errorf("%w: a user with this email already exists", domain.ErrConflict)
		}
		return nil, err
	}
	return h.users.GetByID(ctx, u.ID)
}

// UpdateUserHandler implements port.UpdateUserUseCase.
type UpdateUserHandler struct {
	users port.UserRepository
	org   orgResolver
}

var _ port.UpdateUserUseCase = (*UpdateUserHandler)(nil)

func NewUpdateUserHandler(users port.UserRepository, depts port.DepartmentRepository, currs port.CurriculumRepository) *UpdateUserHandler {
	return &UpdateUserHandler{users: users, org: orgResolver{depts: depts, currs: currs}}
}

func (h *UpdateUserHandler) Execute(ctx context.Context, actor *domain.Actor, in port.UpdateUserInput) (*domain.User, error) {
	if !actor.CanManageUsers() {
		return nil, domain.ErrForbidden
	}
	target, err := h.users.GetByID(ctx, in.ID)
	if err != nil {
		return nil, notFoundAs(err, "user not found")
	}
	// The caller must be allowed to manage the user as it is now ...
	if !actor.CanManageUser(target.ScopeInfo()) {
		return nil, domain.ErrForbidden
	}

	newRole := target.Role
	if in.Role != nil {
		if !in.Role.Valid() {
			return nil, fmt.Errorf("%w: role must be one of super_admin, admin, instructor, staff", domain.ErrInvalidInput)
		}
		newRole = *in.Role
	}
	if newRole != target.Role {
		if target.ID == actor.UserID {
			return nil, domain.ErrForbidden
		}
		if !actor.CanAssignRole(newRole) {
			return nil, domain.ErrForbidden
		}
	}
	if in.AdminScope != nil {
		if !actor.CanSetAdminScope() {
			return nil, domain.ErrForbidden
		}
		if newRole != domain.RoleAdmin {
			return nil, fmt.Errorf("%w: admin_scope is only valid for role admin", domain.ErrInvalidInput)
		}
	}

	deptID, currID := target.DepartmentID, target.CurriculumID
	if in.DepartmentID.Set {
		deptID = in.DepartmentID.Value
		// Moving to another department without naming a curriculum drops a curriculum of the old department.
		if !in.CurriculumID.Set && currID != nil && (deptID == nil || target.DepartmentID == nil || *deptID != *target.DepartmentID) {
			currID = nil
		}
	}
	if in.CurriculumID.Set {
		currID = in.CurriculumID.Value
		if currID != nil && !in.DepartmentID.Set {
			deptID = nil // derived from the curriculum below
		}
	}
	deptID, currID, err = h.org.resolvePlacement(ctx, deptID, currID)
	if err != nil {
		return nil, err
	}

	updated := &domain.User{
		ID: target.ID, Email: target.Email, Name: target.Name, PictureURL: target.PictureURL, Role: newRole,
		DepartmentID: deptID, CurriculumID: currID, UpdatedAt: time.Now().UTC(),
	}
	if newRole == domain.RoleAdmin {
		switch {
		case in.AdminScope != nil:
			updated.AdminScope, err = h.org.resolveScope(ctx, in.AdminScope)
			if err != nil {
				return nil, err
			}
		case target.AdminScope != nil:
			updated.AdminScope = target.AdminScope
		default:
			updated.AdminScope = &domain.AdminScope{Departments: []domain.Ref{}, Curricula: []domain.Ref{}}
		}
	}
	// ... and as it would be afterwards (an admin may not move a user out of scope or promote it).
	if !actor.CanManageUser(updated.ScopeInfo()) {
		return nil, domain.ErrForbidden
	}
	if err := h.users.Update(ctx, updated); err != nil {
		return nil, notFoundAs(err, "user not found")
	}
	return h.users.GetByID(ctx, updated.ID)
}

// DeleteUserHandler implements port.DeleteUserUseCase.
type DeleteUserHandler struct {
	users port.UserRepository
}

var _ port.DeleteUserUseCase = (*DeleteUserHandler)(nil)

func NewDeleteUserHandler(users port.UserRepository) *DeleteUserHandler {
	return &DeleteUserHandler{users: users}
}

func (h *DeleteUserHandler) Execute(ctx context.Context, actor *domain.Actor, id uuid.UUID) error {
	if !actor.CanManageUsers() {
		return domain.ErrForbidden
	}
	if id == actor.UserID {
		return fmt.Errorf("%w: you cannot delete your own account", domain.ErrConflict)
	}
	target, err := h.users.GetByID(ctx, id)
	if err != nil {
		return notFoundAs(err, "user not found")
	}
	if !actor.CanManageUser(target.ScopeInfo()) {
		return domain.ErrForbidden
	}
	return notFoundAs(h.users.Delete(ctx, id), "user not found")
}
