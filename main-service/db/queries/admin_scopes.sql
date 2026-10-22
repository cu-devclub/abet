-- name: InsertAdminScopeDepartment :exec
INSERT INTO admin_scopes (user_id, department_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: InsertAdminScopeCurriculum :exec
INSERT INTO admin_scopes (user_id, curriculum_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: DeleteAdminScopes :exec
DELETE FROM admin_scopes WHERE user_id = $1;

-- name: ListAdminScopeDepartments :many
SELECT s.user_id, d.id, d.code, d.name
FROM admin_scopes s
JOIN departments d ON d.id = s.department_id
WHERE s.user_id = ANY($1::uuid[])
ORDER BY d.code;

-- name: ListAdminScopeCurricula :many
SELECT s.user_id, c.id, c.code, c.name
FROM admin_scopes s
JOIN curricula c ON c.id = s.curriculum_id
WHERE s.user_id = ANY($1::uuid[])
ORDER BY c.code;

-- name: ListActorScopeCurricula :many
SELECT c.id, c.department_id
FROM curricula c
WHERE c.id IN (SELECT s.curriculum_id FROM admin_scopes s WHERE s.user_id = $1 AND s.curriculum_id IS NOT NULL)
   OR c.department_id IN (SELECT s.department_id FROM admin_scopes s WHERE s.user_id = $1 AND s.department_id IS NOT NULL);
