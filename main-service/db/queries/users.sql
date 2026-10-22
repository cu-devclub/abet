-- name: CreateUser :exec
INSERT INTO users (id, email, name, picture_url, role, department_id, curriculum_id, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetUserByID :one
SELECT u.id, u.email, u.name, u.picture_url, u.role, u.department_id, u.curriculum_id, u.last_login_at,
       u.created_at, u.updated_at,
       COALESCE(d.code, '')::text AS department_code, COALESCE(d.name, '')::text AS department_name,
       COALESCE(c.code, '')::text AS curriculum_code, COALESCE(c.name, '')::text AS curriculum_name
FROM users u
LEFT JOIN departments d ON d.id = u.department_id
LEFT JOIN curricula c ON c.id = u.curriculum_id
WHERE u.id = $1;

-- name: GetUserByEmail :one
SELECT u.id, u.email, u.name, u.picture_url, u.role, u.department_id, u.curriculum_id, u.last_login_at,
       u.created_at, u.updated_at,
       COALESCE(d.code, '')::text AS department_code, COALESCE(d.name, '')::text AS department_name,
       COALESCE(c.code, '')::text AS curriculum_code, COALESCE(c.name, '')::text AS curriculum_name
FROM users u
LEFT JOIN departments d ON d.id = u.department_id
LEFT JOIN curricula c ON c.id = u.curriculum_id
WHERE u.email = $1;

-- name: ListUsers :many
SELECT u.id, u.email, u.name, u.picture_url, u.role, u.department_id, u.curriculum_id, u.last_login_at,
       u.created_at, u.updated_at,
       COALESCE(d.code, '')::text AS department_code, COALESCE(d.name, '')::text AS department_name,
       COALESCE(c.code, '')::text AS curriculum_code, COALESCE(c.name, '')::text AS curriculum_name
FROM users u
LEFT JOIN departments d ON d.id = u.department_id
LEFT JOIN curricula c ON c.id = u.curriculum_id
WHERE (sqlc.narg('search')::text IS NULL
        OR u.email ILIKE '%' || sqlc.narg('search')::text || '%'
        OR u.name ILIKE '%' || sqlc.narg('search')::text || '%')
  AND (sqlc.narg('role')::text IS NULL OR u.role = sqlc.narg('role')::text)
  AND (sqlc.narg('department_id')::uuid IS NULL OR u.department_id = sqlc.narg('department_id')::uuid)
ORDER BY u.email;

-- name: UpdateUser :execrows
UPDATE users SET role = $2, department_id = $3, curriculum_id = $4, updated_at = $5 WHERE id = $1;

-- name: DeleteUser :execrows
DELETE FROM users WHERE id = $1;

-- name: UpsertSuperAdmin :one
INSERT INTO users (email, role)
VALUES ($1, 'super_admin')
ON CONFLICT (email) DO UPDATE SET role = 'super_admin', updated_at = now()
RETURNING (xmax = 0)::boolean AS created;

-- name: TouchUserLogin :execrows
UPDATE users
SET name = $2, picture_url = $3, last_login_at = $4, updated_at = $4
WHERE email = $1;
