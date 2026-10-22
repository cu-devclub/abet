-- name: CreateCurriculum :exec
INSERT INTO curricula (id, code, name, department_id, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetCurriculum :one
SELECT c.id, c.code, c.name, c.department_id, c.cover_filename, c.cover_size_bytes, c.cover_updated_at,
       c.created_at, c.updated_at, d.code AS department_code, d.name AS department_name
FROM curricula c
JOIN departments d ON d.id = c.department_id
WHERE c.id = $1;

-- name: ListCurricula :many
SELECT c.id, c.code, c.name, c.department_id, c.cover_filename, c.cover_size_bytes, c.cover_updated_at,
       c.created_at, c.updated_at, d.code AS department_code, d.name AS department_name
FROM curricula c
JOIN departments d ON d.id = c.department_id
WHERE (sqlc.narg('department_id')::uuid IS NULL OR c.department_id = sqlc.narg('department_id')::uuid)
ORDER BY c.code;

-- name: UpdateCurriculum :execrows
UPDATE curricula SET code = $2, name = $3, department_id = $4, updated_at = $5 WHERE id = $1;

-- name: DeleteCurriculum :execrows
DELETE FROM curricula WHERE id = $1;
