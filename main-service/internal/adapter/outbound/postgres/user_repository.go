package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"abet/main-service/internal/adapter/outbound/postgres/sqlc"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

type UserRepository struct {
	store *Store
}

var (
	_ port.UserRepository = (*UserRepository)(nil)
	_ port.ActorLoader    = (*UserRepository)(nil)
)

func NewUserRepository(store *Store) *UserRepository {
	return &UserRepository{store: store}
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	err := r.store.WithTx(ctx, func(q *sqlc.Queries) error {
		if err := q.CreateUser(ctx, sqlc.CreateUserParams{
			ID: u.ID, Email: u.Email, Name: u.Name, PictureUrl: u.PictureURL, Role: string(u.Role),
			DepartmentID: nullUUID(u.DepartmentID), CurriculumID: nullUUID(u.CurriculumID),
			CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
		}); err != nil {
			return err
		}
		return replaceScopes(ctx, q, u)
	})
	return mapErr("create user", err)
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	row, err := r.store.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, mapErr("get user", err)
	}
	return r.withScope(ctx, userFromRow(sqlc.ListUsersRow(row)))
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := r.store.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, mapErr("get user by email", err)
	}
	return r.withScope(ctx, userFromRow(sqlc.ListUsersRow(row)))
}

func (r *UserRepository) List(ctx context.Context, f port.UserFilter) ([]domain.User, error) {
	p := sqlc.ListUsersParams{DepartmentID: nullUUID(f.DepartmentID)}
	if f.Search != "" {
		p.Search = &f.Search
	}
	if f.Role != nil {
		role := string(*f.Role)
		p.Role = &role
	}
	rows, err := r.store.q.ListUsers(ctx, p)
	if err != nil {
		return nil, mapErr("list users", err)
	}
	users := make([]domain.User, 0, len(rows))
	var adminIDs []uuid.UUID
	for _, row := range rows {
		u := userFromRow(row)
		if u.Role == domain.RoleAdmin {
			adminIDs = append(adminIDs, u.ID)
		}
		users = append(users, *u)
	}
	if len(adminIDs) > 0 {
		scopes, err := r.loadScopes(ctx, adminIDs)
		if err != nil {
			return nil, err
		}
		for i := range users {
			if users[i].Role == domain.RoleAdmin {
				users[i].AdminScope = scopes[users[i].ID]
			}
		}
	}
	return users, nil
}

func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	err := r.store.WithTx(ctx, func(q *sqlc.Queries) error {
		n, err := q.UpdateUser(ctx, sqlc.UpdateUserParams{
			ID: u.ID, Role: string(u.Role), DepartmentID: nullUUID(u.DepartmentID),
			CurriculumID: nullUUID(u.CurriculumID), UpdatedAt: u.UpdatedAt,
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return domain.ErrNotFound
		}
		return replaceScopes(ctx, q, u)
	})
	return mapErr("update user", err)
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	n, err := r.store.q.DeleteUser(ctx, id)
	if err != nil {
		return mapErr("delete user", err)
	}
	if n == 0 {
		return fmt.Errorf("delete user: %w", domain.ErrNotFound)
	}
	return nil
}

func (r *UserRepository) UpsertSuperAdmin(ctx context.Context, email string) (bool, error) {
	created, err := r.store.q.UpsertSuperAdmin(ctx, email)
	return created, mapErr("upsert super admin", err)
}

func (r *UserRepository) TouchLogin(ctx context.Context, email, name, pictureURL string) (*domain.User, error) {
	now := time.Now().UTC()
	n, err := r.store.q.TouchUserLogin(ctx, sqlc.TouchUserLoginParams{
		Email: email, Name: name, PictureUrl: pictureURL, LastLoginAt: &now,
	})
	if err != nil {
		return nil, mapErr("touch login", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("touch login: %w", domain.ErrNotFound)
	}
	return r.GetByEmail(ctx, email)
}

// LoadActor builds the authorization actor: the user plus (for admins) the expanded admin scope.
func (r *UserRepository) LoadActor(ctx context.Context, id uuid.UUID) (*domain.Actor, error) {
	row, err := r.store.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, mapErr("load actor", err)
	}
	a := &domain.Actor{
		UserID: row.ID, Email: row.Email, Role: domain.Role(row.Role),
		DepartmentID: ptrUUID(row.DepartmentID), CurriculumID: ptrUUID(row.CurriculumID),
	}
	if a.Role != domain.RoleAdmin {
		return a, nil
	}
	depts, err := r.store.q.ListAdminScopeDepartments(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, mapErr("load actor scope", err)
	}
	for _, d := range depts {
		a.ScopeDepartmentIDs = append(a.ScopeDepartmentIDs, d.ID)
	}
	curricula, err := r.store.q.ListActorScopeCurricula(ctx, id)
	if err != nil {
		return nil, mapErr("load actor scope", err)
	}
	seenDept := map[uuid.UUID]bool{}
	for _, c := range curricula {
		a.ScopeCurriculumIDs = append(a.ScopeCurriculumIDs, c.ID)
		if !seenDept[c.DepartmentID] {
			seenDept[c.DepartmentID] = true
			a.ScopeCurriculumDepartmentIDs = append(a.ScopeCurriculumDepartmentIDs, c.DepartmentID)
		}
	}
	return a, nil
}

// replaceScopes rewrites the admin_scopes rows of u (none unless role=admin and AdminScope != nil).
func replaceScopes(ctx context.Context, q *sqlc.Queries, u *domain.User) error {
	if err := q.DeleteAdminScopes(ctx, u.ID); err != nil {
		return err
	}
	if u.Role != domain.RoleAdmin || u.AdminScope == nil {
		return nil
	}
	for _, d := range u.AdminScope.Departments {
		if err := q.InsertAdminScopeDepartment(ctx, sqlc.InsertAdminScopeDepartmentParams{
			UserID: u.ID, DepartmentID: uuid.NullUUID{UUID: d.ID, Valid: true},
		}); err != nil {
			return err
		}
	}
	for _, c := range u.AdminScope.Curricula {
		if err := q.InsertAdminScopeCurriculum(ctx, sqlc.InsertAdminScopeCurriculumParams{
			UserID: u.ID, CurriculumID: uuid.NullUUID{UUID: c.ID, Valid: true},
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *UserRepository) withScope(ctx context.Context, u *domain.User) (*domain.User, error) {
	if u.Role != domain.RoleAdmin {
		return u, nil
	}
	scopes, err := r.loadScopes(ctx, []uuid.UUID{u.ID})
	if err != nil {
		return nil, err
	}
	u.AdminScope = scopes[u.ID]
	return u, nil
}

// loadScopes returns the admin scope of every given user id (always non-nil entries, possibly empty).
func (r *UserRepository) loadScopes(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.AdminScope, error) {
	out := make(map[uuid.UUID]*domain.AdminScope, len(ids))
	for _, id := range ids {
		out[id] = &domain.AdminScope{Departments: []domain.Ref{}, Curricula: []domain.Ref{}}
	}
	depts, err := r.store.q.ListAdminScopeDepartments(ctx, ids)
	if err != nil {
		return nil, mapErr("load admin scope", err)
	}
	for _, d := range depts {
		s := out[d.UserID]
		s.Departments = append(s.Departments, domain.Ref{ID: d.ID, Code: d.Code, Name: d.Name})
	}
	curricula, err := r.store.q.ListAdminScopeCurricula(ctx, ids)
	if err != nil {
		return nil, mapErr("load admin scope", err)
	}
	for _, c := range curricula {
		s := out[c.UserID]
		s.Curricula = append(s.Curricula, domain.Ref{ID: c.ID, Code: c.Code, Name: c.Name})
	}
	return out, nil
}

func userFromRow(row sqlc.ListUsersRow) *domain.User {
	u := &domain.User{
		ID: row.ID, Email: row.Email, Name: row.Name, PictureURL: row.PictureUrl, Role: domain.Role(row.Role),
		DepartmentID: ptrUUID(row.DepartmentID), CurriculumID: ptrUUID(row.CurriculumID),
		LastLoginAt: row.LastLoginAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if u.DepartmentID != nil {
		u.Department = &domain.Ref{ID: *u.DepartmentID, Code: row.DepartmentCode, Name: row.DepartmentName}
	}
	if u.CurriculumID != nil {
		u.Curriculum = &domain.Ref{ID: *u.CurriculumID, Code: row.CurriculumCode, Name: row.CurriculumName}
	}
	return u
}

func nullUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}

func ptrUUID(n uuid.NullUUID) *uuid.UUID {
	if !n.Valid {
		return nil
	}
	id := n.UUID
	return &id
}
