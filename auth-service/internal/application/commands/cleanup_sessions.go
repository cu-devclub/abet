package commands

import (
	"context"
	"time"

	"abet/auth-service/internal/core/port"
)

// CleanupRetention is how long an expired session row is kept before deletion.
const CleanupRetention = 7 * 24 * time.Hour

// CleanupSessionsHandler deletes sessions that expired more than CleanupRetention ago.
type CleanupSessionsHandler struct {
	sessions port.SessionRepository
	now      func() time.Time
}

// NewCleanupSessionsHandler creates the handler; now may be nil (defaults to time.Now).
func NewCleanupSessionsHandler(sessions port.SessionRepository, now func() time.Time) *CleanupSessionsHandler {
	if now == nil {
		now = time.Now
	}
	return &CleanupSessionsHandler{sessions: sessions, now: now}
}

func (h *CleanupSessionsHandler) Execute(ctx context.Context) (int64, error) {
	return h.sessions.DeleteExpiredBefore(ctx, h.now().UTC().Add(-CleanupRetention))
}

var _ port.CleanupSessionsUseCase = (*CleanupSessionsHandler)(nil)
