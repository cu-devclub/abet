-- name: ListScopeOfferingsForUpdate :many
SELECT o.id, o.course_id, c.code AS course_code, c.name AS course_name, o.semester,
       EXISTS (SELECT 1 FROM submissions s WHERE s.offering_id = o.id) AS has_submissions
FROM course_offerings o
JOIN courses c ON c.id = o.course_id
WHERE c.curriculum_id = sqlc.arg('curriculum_id') AND o.academic_year = sqlc.arg('academic_year')
ORDER BY c.code, o.semester
FOR UPDATE OF o;

-- name: ListScopeAccess :many
SELECT a.offering_id, a.user_id, u.email, u.role, a.can_upload, a.can_download, a.source
FROM course_access a
JOIN users u ON u.id = a.user_id
JOIN course_offerings o ON o.id = a.offering_id
JOIN courses c ON c.id = o.course_id
WHERE c.curriculum_id = sqlc.arg('curriculum_id') AND o.academic_year = sqlc.arg('academic_year');

-- name: ListUsersByEmails :many
SELECT id, email, role FROM users WHERE email = ANY(sqlc.arg('emails')::text[]);

-- name: UpdateUserRole :execrows
UPDATE users SET role = $2, updated_at = $3 WHERE id = $1;

-- name: UpdateCourseName :exec
UPDATE courses SET name = $2, updated_at = $3 WHERE id = $1;
