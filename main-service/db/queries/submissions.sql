-- name: GetOfferingForSubmission :one
SELECT o.id, c.code AS course_code, c.name AS course_name, o.academic_year, o.semester,
       c.curriculum_id, cu.code AS curriculum_code, cu.name AS curriculum_name,
       cu.department_id, d.code AS department_code, d.name AS department_name
FROM course_offerings o
JOIN courses c ON c.id = o.course_id
JOIN curricula cu ON cu.id = c.curriculum_id
JOIN departments d ON d.id = cu.department_id
WHERE o.id = $1;

-- name: ListOfferingsForSubmissions :many
SELECT o.id, c.code AS course_code, c.name AS course_name, o.academic_year, o.semester,
       c.curriculum_id, cu.code AS curriculum_code, cu.name AS curriculum_name,
       cu.department_id, d.code AS department_code, d.name AS department_name
FROM course_offerings o
JOIN courses c ON c.id = o.course_id
JOIN curricula cu ON cu.id = c.curriculum_id
JOIN departments d ON d.id = cu.department_id
WHERE o.id = ANY(sqlc.arg('ids')::uuid[]);

-- name: ListUserGrants :many
SELECT offering_id, can_upload, can_download
FROM course_access
WHERE user_id = sqlc.arg('user_id')::uuid
  AND (sqlc.arg('all')::bool OR offering_id = ANY(sqlc.arg('offering_ids')::uuid[]));

-- name: InsertSubmission :exec
INSERT INTO submissions (id, offering_id, uploaded_by, original_filename, size_bytes, status, job_id,
                         error_count, warning_count, created_at, updated_at, completed_at)
VALUES (sqlc.arg('id'), sqlc.arg('offering_id'), sqlc.arg('uploaded_by'), sqlc.arg('original_filename'),
        sqlc.arg('size_bytes'), sqlc.arg('status'), sqlc.arg('job_id'), sqlc.arg('error_count'),
        sqlc.arg('warning_count'), sqlc.arg('at')::timestamptz, sqlc.arg('at')::timestamptz,
        sqlc.narg('completed_at')::timestamptz);

-- name: InsertSubmissionIssues :exec
INSERT INTO submission_issues (submission_id, severity, stage, code, title, description, location, suggested_fix, sort_order)
SELECT sqlc.arg('submission_id')::uuid,
       unnest(sqlc.arg('severities')::text[]), unnest(sqlc.arg('stages')::text[]), unnest(sqlc.arg('codes')::text[]),
       unnest(sqlc.arg('titles')::text[]), unnest(sqlc.arg('descriptions')::text[]), unnest(sqlc.arg('locations')::text[]),
       unnest(sqlc.arg('suggested_fixes')::text[]), unnest(sqlc.arg('sort_orders')::int[]);

-- name: GetSubmission :one
SELECT s.id, s.offering_id, s.uploaded_by, u.name AS uploader_name, u.email AS uploader_email,
       s.original_filename, s.size_bytes, s.status, s.job_id, s.pdf_size_bytes,
       s.error_count, s.warning_count, s.created_at, s.updated_at, s.completed_at
FROM submissions s
LEFT JOIN users u ON u.id = s.uploaded_by
WHERE s.id = $1;

-- name: GetSubmissionByJobID :one
SELECT s.id, s.offering_id, s.uploaded_by, u.name AS uploader_name, u.email AS uploader_email,
       s.original_filename, s.size_bytes, s.status, s.job_id, s.pdf_size_bytes,
       s.error_count, s.warning_count, s.created_at, s.updated_at, s.completed_at
FROM submissions s
LEFT JOIN users u ON u.id = s.uploaded_by
WHERE s.job_id = $1;

-- name: ListSubmissionIssues :many
SELECT severity, stage, code, title, description, location, suggested_fix
FROM submission_issues
WHERE submission_id = $1
ORDER BY sort_order;

-- name: ListSubmissionsByOffering :many
SELECT s.id, s.offering_id, s.uploaded_by, u.name AS uploader_name, u.email AS uploader_email,
       s.original_filename, s.size_bytes, s.status, s.job_id, s.pdf_size_bytes,
       s.error_count, s.warning_count, s.created_at, s.updated_at, s.completed_at
FROM submissions s
LEFT JOIN users u ON u.id = s.uploaded_by
WHERE s.offering_id = $1
ORDER BY s.created_at DESC;

-- name: GetLatestGeneratedSubmission :one
SELECT id, pdf_size_bytes
FROM submissions
WHERE offering_id = $1 AND status = 'generated' AND pdf_size_bytes IS NOT NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: ListLatestGeneratedSubmissions :many
SELECT DISTINCT ON (offering_id) offering_id, id
FROM submissions
WHERE offering_id = ANY(sqlc.arg('offering_ids')::uuid[]) AND status = 'generated' AND pdf_size_bytes IS NOT NULL
ORDER BY offering_id, created_at DESC;

-- name: MarkSubmissionGenerated :execrows
UPDATE submissions
SET status = 'generated', pdf_size_bytes = sqlc.narg('pdf_size_bytes')::bigint, error_count = 0,
    completed_at = sqlc.arg('at')::timestamptz, updated_at = sqlc.arg('at')::timestamptz
WHERE id = sqlc.arg('id') AND status IN ('queued', 'processing');

-- name: MarkSubmissionFailed :execrows
UPDATE submissions
SET status = 'failed', error_count = sqlc.arg('error_count')::int,
    completed_at = sqlc.arg('at')::timestamptz, updated_at = sqlc.arg('at')::timestamptz
WHERE id = sqlc.arg('id') AND status IN ('queued', 'processing');

-- name: HasNewerGeneratedSubmission :one
SELECT EXISTS (
  SELECT 1 FROM submissions n
  JOIN submissions s ON s.id = sqlc.arg('id')::uuid
  WHERE n.offering_id = s.offering_id AND n.id <> s.id AND n.status = 'generated'
    AND n.pdf_size_bytes IS NOT NULL AND n.created_at > s.created_at
);

-- name: SupersedeOlderGenerated :many
UPDATE submissions o
SET pdf_size_bytes = NULL, updated_at = sqlc.arg('at')::timestamptz
FROM submissions s
WHERE s.id = sqlc.arg('id')::uuid
  AND o.offering_id = s.offering_id AND o.id <> s.id AND o.status = 'generated'
  AND o.pdf_size_bytes IS NOT NULL AND o.created_at < s.created_at
RETURNING o.id;

-- name: ClearSubmissionPDF :exec
UPDATE submissions SET pdf_size_bytes = NULL, updated_at = sqlc.arg('at')::timestamptz WHERE id = sqlc.arg('id');

-- name: ListPortfolioRows :many
-- The latest submission is joined as a plain table (nullable columns for sqlc) through a correlated "newest id" lookup.
SELECT o.id AS offering_id, c.code AS course_code, c.name AS course_name, o.academic_year, o.semester,
       c.curriculum_id, cu.code AS curriculum_code, cu.name AS curriculum_name,
       cu.department_id, d.code AS department_code, d.name AS department_name,
       s.id AS sub_id, s.uploaded_by AS sub_uploaded_by, u.name AS sub_uploader_name,
       u.email AS sub_uploader_email, s.original_filename AS sub_original_filename,
       s.size_bytes AS sub_size_bytes, s.status AS sub_status, s.error_count AS sub_error_count,
       s.warning_count AS sub_warning_count, s.created_at AS sub_created_at, s.completed_at AS sub_completed_at,
       EXISTS (SELECT 1 FROM submissions g WHERE g.offering_id = o.id AND g.status = 'generated' AND g.pdf_size_bytes IS NOT NULL) AS pdf_available
FROM course_offerings o
JOIN courses c ON c.id = o.course_id
JOIN curricula cu ON cu.id = c.curriculum_id
JOIN departments d ON d.id = cu.department_id
LEFT JOIN submissions s ON s.id = (
  SELECT l.id FROM submissions l WHERE l.offering_id = o.id ORDER BY l.created_at DESC LIMIT 1
)
LEFT JOIN users u ON u.id = s.uploaded_by
WHERE (sqlc.narg('academic_year')::int IS NULL OR o.academic_year = sqlc.narg('academic_year')::int)
  AND (sqlc.narg('department_id')::uuid IS NULL OR cu.department_id = sqlc.narg('department_id')::uuid)
  AND (sqlc.narg('curriculum_id')::uuid IS NULL OR c.curriculum_id = sqlc.narg('curriculum_id')::uuid)
  AND (sqlc.narg('semester')::smallint IS NULL OR o.semester = sqlc.narg('semester')::smallint)
  AND (sqlc.narg('search')::text IS NULL
        OR c.code ILIKE '%' || sqlc.narg('search')::text || '%'
        OR c.name ILIKE '%' || sqlc.narg('search')::text || '%')
  AND (sqlc.arg('all')::bool
        OR c.curriculum_id = ANY(sqlc.arg('curriculum_ids')::uuid[])
        OR (sqlc.narg('staff_department_id')::uuid IS NOT NULL AND cu.department_id = sqlc.narg('staff_department_id')::uuid)
        OR EXISTS (SELECT 1 FROM course_access a WHERE a.offering_id = o.id AND a.user_id = sqlc.arg('user_id')::uuid))
ORDER BY c.code, o.semester;

-- name: NextSubmissionIssueOrder :one
SELECT COALESCE(MAX(sort_order) + 1, 0)::int AS next_order FROM submission_issues WHERE submission_id = $1;

-- name: TimeOutStaleSubmissions :many
-- Atomic: only rows still waiting for the worker are touched, so a result that arrives meanwhile always wins or loses cleanly.
UPDATE submissions
SET status = 'failed', error_count = 1, completed_at = sqlc.arg('at')::timestamptz, updated_at = sqlc.arg('at')::timestamptz
WHERE status = 'queued' AND created_at < sqlc.arg('cutoff')::timestamptz
RETURNING id;

-- name: MarkTimedOutSubmissionGenerated :execrows
-- Late success: a submission that was failed by the reconciler (BUILD_TIMEOUT) becomes generated.
-- error_count is recalculated from the issues that remain once the timeout issue is gone.
UPDATE submissions
SET status = 'generated', pdf_size_bytes = sqlc.narg('pdf_size_bytes')::bigint,
    error_count = (SELECT COUNT(*) FROM submission_issues i
                   WHERE i.submission_id = submissions.id AND i.severity = 'error' AND i.code <> 'BUILD_TIMEOUT')::int,
    completed_at = sqlc.arg('at')::timestamptz, updated_at = sqlc.arg('at')::timestamptz
WHERE submissions.id = sqlc.arg('id') AND submissions.status = 'failed'
  AND EXISTS (SELECT 1 FROM submission_issues i WHERE i.submission_id = submissions.id AND i.code = 'BUILD_TIMEOUT');

-- name: DeleteSubmissionIssuesByCode :exec
DELETE FROM submission_issues WHERE submission_id = sqlc.arg('submission_id') AND code = sqlc.arg('code')::text;

-- name: ListSubmissionStatuses :many
SELECT id, status FROM submissions WHERE id = ANY(sqlc.arg('ids')::uuid[]);
