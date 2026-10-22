-- name: ListOfferings :many
SELECT o.id, o.course_id, c.code AS course_code, c.name AS course_name, o.academic_year, o.semester,
       c.curriculum_id, cu.code AS curriculum_code, cu.name AS curriculum_name,
       cu.department_id, d.code AS department_code, d.name AS department_name,
       (SELECT count(*) FROM course_access a JOIN users u ON u.id = a.user_id
         WHERE a.offering_id = o.id AND u.role = 'instructor')::int AS instructor_count,
       (SELECT count(*) FROM course_access a JOIN users u ON u.id = a.user_id
         WHERE a.offering_id = o.id AND u.role = 'staff')::int AS staff_count,
       EXISTS (SELECT 1 FROM submissions s WHERE s.offering_id = o.id) AS has_submissions,
       o.created_at, o.updated_at
FROM course_offerings o
JOIN courses c ON c.id = o.course_id
JOIN curricula cu ON cu.id = c.curriculum_id
JOIN departments d ON d.id = cu.department_id
WHERE o.academic_year = sqlc.arg('academic_year')::int
  AND (sqlc.narg('department_id')::uuid IS NULL OR cu.department_id = sqlc.narg('department_id')::uuid)
  AND (sqlc.narg('curriculum_id')::uuid IS NULL OR c.curriculum_id = sqlc.narg('curriculum_id')::uuid)
  AND (sqlc.narg('semester')::smallint IS NULL OR o.semester = sqlc.narg('semester')::smallint)
  AND (sqlc.narg('search')::text IS NULL
        OR c.code ILIKE '%' || sqlc.narg('search')::text || '%'
        OR c.name ILIKE '%' || sqlc.narg('search')::text || '%')
  AND (NOT sqlc.arg('restrict')::bool OR c.curriculum_id = ANY(sqlc.arg('curriculum_ids')::uuid[]))
ORDER BY c.code, o.semester;

-- name: ListOfferingsByIDs :many
SELECT o.id, o.course_id, c.code AS course_code, c.name AS course_name, o.academic_year, o.semester,
       c.curriculum_id, cu.code AS curriculum_code, cu.name AS curriculum_name,
       cu.department_id, d.code AS department_code, d.name AS department_name,
       (SELECT count(*) FROM course_access a JOIN users u ON u.id = a.user_id
         WHERE a.offering_id = o.id AND u.role = 'instructor')::int AS instructor_count,
       (SELECT count(*) FROM course_access a JOIN users u ON u.id = a.user_id
         WHERE a.offering_id = o.id AND u.role = 'staff')::int AS staff_count,
       EXISTS (SELECT 1 FROM submissions s WHERE s.offering_id = o.id) AS has_submissions,
       o.created_at, o.updated_at
FROM course_offerings o
JOIN courses c ON c.id = o.course_id
JOIN curricula cu ON cu.id = c.curriculum_id
JOIN departments d ON d.id = cu.department_id
WHERE o.id = ANY(sqlc.arg('ids')::uuid[])
ORDER BY c.code, o.semester;

-- name: ListOfferingYears :many
SELECT DISTINCT academic_year FROM course_offerings ORDER BY academic_year DESC;

-- name: UpsertCourse :one
INSERT INTO courses (id, code, name, curriculum_id, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('code'), sqlc.arg('name'), sqlc.arg('curriculum_id'), sqlc.arg('at')::timestamptz, sqlc.arg('at')::timestamptz)
ON CONFLICT (curriculum_id, code) DO UPDATE SET
  name = CASE WHEN sqlc.arg('rename')::bool THEN EXCLUDED.name ELSE courses.name END,
  updated_at = CASE WHEN sqlc.arg('rename')::bool AND courses.name <> EXCLUDED.name THEN EXCLUDED.updated_at ELSE courses.updated_at END
RETURNING id;

-- name: CreateOffering :exec
INSERT INTO course_offerings (id, course_id, academic_year, semester, created_at, updated_at)
VALUES (sqlc.arg('id'), sqlc.arg('course_id'), sqlc.arg('academic_year'), sqlc.arg('semester'), sqlc.arg('at')::timestamptz, sqlc.arg('at')::timestamptz);

-- name: LockOffering :one
SELECT course_id FROM course_offerings WHERE id = $1 FOR UPDATE;

-- name: UpdateOffering :execrows
UPDATE course_offerings
SET course_id = sqlc.arg('course_id'), academic_year = sqlc.arg('academic_year'), semester = sqlc.arg('semester'),
    updated_at = sqlc.arg('at')::timestamptz
WHERE id = sqlc.arg('id');

-- name: ListSubmissionIDsForOfferings :many
SELECT id FROM submissions WHERE offering_id = ANY(sqlc.arg('offering_ids')::uuid[]);

-- name: DeleteOfferings :many
DELETE FROM course_offerings WHERE id = ANY(sqlc.arg('ids')::uuid[]) RETURNING course_id;

-- name: DeleteOrphanCourses :execrows
DELETE FROM courses c
WHERE c.id = ANY(sqlc.arg('ids')::uuid[])
  AND NOT EXISTS (SELECT 1 FROM course_offerings o WHERE o.course_id = c.id);
