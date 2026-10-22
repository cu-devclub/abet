-- name: CreateDepartment :exec
INSERT INTO departments (id, code, name, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetDepartment :one
SELECT d.id, d.code, d.name, d.created_at, d.updated_at,
       (SELECT count(*) FROM curricula c WHERE c.department_id = d.id)::int AS curriculum_count
FROM departments d
WHERE d.id = $1;

-- name: ListDepartments :many
SELECT d.id, d.code, d.name, d.created_at, d.updated_at,
       (SELECT count(*) FROM curricula c WHERE c.department_id = d.id)::int AS curriculum_count
FROM departments d
ORDER BY d.code;

-- name: UpdateDepartment :execrows
UPDATE departments SET code = $2, name = $3, updated_at = $4 WHERE id = $1;

-- name: DeleteDepartment :execrows
DELETE FROM departments WHERE id = $1;
