-- name: ListOfferingAccess :many
SELECT a.user_id, u.name, u.email, u.role, a.can_upload, a.can_download, a.source
FROM course_access a
JOIN users u ON u.id = a.user_id
WHERE a.offering_id = $1
ORDER BY u.role DESC, u.email;

-- name: ListDefaultStaffForOffering :many
SELECT u.id AS user_id, u.name, u.email
FROM users u
WHERE u.role = 'staff'
  AND u.department_id = sqlc.arg('department_id')::uuid
  AND NOT EXISTS (SELECT 1 FROM course_access a WHERE a.offering_id = sqlc.arg('offering_id')::uuid AND a.user_id = u.id)
ORDER BY u.email;

-- name: UpsertAccess :exec
INSERT INTO course_access (id, offering_id, user_id, can_upload, can_download, source, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('offering_id'), sqlc.arg('user_id'), sqlc.arg('can_upload'), sqlc.arg('can_download'),
        sqlc.arg('source'), sqlc.arg('at')::timestamptz, sqlc.arg('at')::timestamptz)
ON CONFLICT (offering_id, user_id) DO UPDATE
SET can_upload = EXCLUDED.can_upload, can_download = EXCLUDED.can_download,
    source = EXCLUDED.source, updated_at = EXCLUDED.updated_at;

-- name: DeleteAccess :execrows
DELETE FROM course_access WHERE offering_id = $1 AND user_id = $2;
