-- name: SetCurriculumCover :execrows
UPDATE curricula
SET cover_filename = $2, cover_size_bytes = $3, cover_updated_at = $4, updated_at = $4
WHERE id = $1;

-- name: ClearCurriculumCover :execrows
UPDATE curricula
SET cover_filename = NULL, cover_size_bytes = NULL, cover_updated_at = NULL, updated_at = $2
WHERE id = $1 AND cover_filename IS NOT NULL;
