package port

import (
	"context"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
)

// ---- outbound ----

// UserFilter narrows UserRepository.List; nil/empty fields are ignored.
type UserFilter struct {
	Search       string // case-insensitive substring of name or email
	Role         *domain.Role
	DepartmentID *uuid.UUID
}

// UserRepository persists users and their admin scopes.
// Returned users include Department/Curriculum refs and (for role=admin) AdminScope. Duplicate email -> ErrConflict.
type UserRepository interface {
	// Create inserts the user and, when u.AdminScope != nil, its admin_scopes rows (atomically).
	Create(ctx context.Context, u *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context, f UserFilter) ([]domain.User, error)
	// Update writes the mutable columns (role, department, curriculum) and replaces the admin_scopes set
	// with u.AdminScope (nil = no scopes). One transaction.
	Update(ctx context.Context, u *domain.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	// UpsertSuperAdmin creates the user as super_admin or, if it exists, promotes it. Returns true if created.
	UpsertSuperAdmin(ctx context.Context, email string) (created bool, err error)
	// TouchLogin updates name/picture/last_login_at of an existing user; ErrNotFound if the email is unknown.
	TouchLogin(ctx context.Context, email, name, pictureURL string) (*domain.User, error)
}

// ActorLoader builds the authorization Actor (user + expanded admin scope). ErrNotFound for unknown ids.
type ActorLoader interface {
	LoadActor(ctx context.Context, id uuid.UUID) (*domain.Actor, error)
}

// ---- inbound ----

// OptionalID distinguishes "field absent" from "field set to null" in PATCH bodies.
type OptionalID struct {
	Set   bool
	Value *uuid.UUID
}

// AdminScopeInput is the full replacement set of an admin's scope.
type AdminScopeInput struct {
	DepartmentIDs []uuid.UUID
	CurriculumIDs []uuid.UUID
}

type CreateUserInput struct {
	Email        string
	Role         domain.Role
	DepartmentID *uuid.UUID
	CurriculumID *uuid.UUID
	AdminScope   *AdminScopeInput
}

type UpdateUserInput struct {
	ID           uuid.UUID
	Role         *domain.Role
	DepartmentID OptionalID
	CurriculumID OptionalID
	AdminScope   *AdminScopeInput
}

type ListUsersInput struct {
	Search       string
	Role         *domain.Role
	DepartmentID *uuid.UUID
}

type CreateUserUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, in CreateUserInput) (*domain.User, error)
}

type UpdateUserUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, in UpdateUserInput) (*domain.User, error)
}

type DeleteUserUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, id uuid.UUID) error
}

type ListUsersUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, in ListUsersInput) ([]domain.User, error)
}

// MeResult is the payload of GET /me.
type MeResult struct {
	User          *domain.User
	ManageOrg     bool
	ManageUsers   bool
	ManageCourses bool
	ManageCovers  bool
}

type GetMeUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor) (*MeResult, error)
}
