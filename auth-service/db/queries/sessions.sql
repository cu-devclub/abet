-- name: CreateSession :exec
INSERT INTO sessions (id, user_id, email, user_agent, ip, created_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetSession :one
SELECT id, user_id, email, user_agent, ip, created_at, expires_at, revoked_at
FROM sessions
WHERE id = $1;

-- name: RevokeSession :exec
UPDATE sessions
SET revoked_at = $2
WHERE id = $1 AND revoked_at IS NULL;

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions
WHERE expires_at < $1;
