package port

import (
	"context"
	"time"
)

// Pinger checks that a dependency (the database) is reachable.
type Pinger interface {
	Ping(ctx context.Context) error
}

// QueueStats is the state of the build queue as the broker reports it.
type QueueStats struct {
	// Consumers is the number of workers currently consuming the build queue.
	Consumers int
	// Messages is the number of jobs waiting in the queue (ready, not yet delivered).
	Messages int
}

// QueueInspector reads the build queue state (a passive declare: it never creates or changes the queue).
type QueueInspector interface {
	Inspect(ctx context.Context) (QueueStats, error)
}

// QueueSnapshot is the last known queue state for /readyz.
type QueueSnapshot struct {
	Known     bool
	Consumers int
	Messages  int
	CheckedAt time.Time
}

// QueueReporter exposes the last known build queue state.
type QueueReporter interface {
	Snapshot() QueueSnapshot
}
