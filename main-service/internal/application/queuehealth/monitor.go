// Package queuehealth tracks whether the portfolio workers are consuming the build queue.
package queuehealth

import (
	"sync"
	"time"

	"abet/main-service/internal/core/port"
)

// Monitor remembers the last queue state and detects "no consumers for too long". It is safe for concurrent use.
type Monitor struct {
	mu         sync.Mutex
	threshold  time.Duration
	last       port.QueueSnapshot
	zeroSince  time.Time // zero value = consumers were seen on the last observation
	incidentOn bool
}

var _ port.QueueReporter = (*Monitor)(nil)

// New returns a Monitor that raises an incident after consumers stayed at 0 for longer than threshold.
func New(threshold time.Duration) *Monitor { return &Monitor{threshold: threshold} }

// Observe records a successful queue check. It returns true exactly once per incident: when consumers have been 0 for
// longer than the threshold. The incident ends (and can be raised again) as soon as a consumer is seen.
func (m *Monitor) Observe(now time.Time, s port.QueueStats) (raise bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.last = port.QueueSnapshot{Known: true, Consumers: s.Consumers, Messages: s.Messages, CheckedAt: now}
	if s.Consumers > 0 {
		m.zeroSince = time.Time{}
		m.incidentOn = false
		return false
	}
	if m.zeroSince.IsZero() {
		m.zeroSince = now
	}
	if !m.incidentOn && now.Sub(m.zeroSince) > m.threshold {
		m.incidentOn = true
		return true
	}
	return false
}

// Snapshot returns the last observation (Known=false until the first successful check).
func (m *Monitor) Snapshot() port.QueueSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.last
}
