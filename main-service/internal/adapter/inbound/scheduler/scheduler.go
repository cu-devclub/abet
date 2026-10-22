// Package scheduler runs the periodic background jobs of main-service: the build-timeout reconciler, the build queue
// health check and the storage janitor. It is an inbound adapter driven by time instead of HTTP.
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"abet/main-service/internal/application/commands"
	"abet/main-service/internal/application/queuehealth"
	"abet/main-service/internal/core/port"
)

var (
	timedOutTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "abet_submissions_timed_out_total",
		Help: "Submissions failed by the reconciler because the portfolio build did not finish in time.",
	})
	queueConsumers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "abet_build_queue_consumers",
		Help: "Portfolio workers currently consuming the build queue.",
	})
	queueMessages = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "abet_build_queue_messages",
		Help: "Build jobs waiting in the queue.",
	})
)

// JanitorEvery is how often the storage janitor runs.
const JanitorEvery = time.Hour

// Reconciler is what the scheduler needs from the timeout use case.
type Reconciler interface {
	Execute(ctx context.Context) (int, error)
}

// Janitor is what the scheduler needs from the cleanup use case.
type Janitor interface {
	Execute(ctx context.Context) (commands.JanitorResult, error)
}

// Scheduler ties the jobs together. Tick does one pass; Run repeats it.
type Scheduler struct {
	Interval  time.Duration
	Reconcile Reconciler
	Janitor   Janitor
	Inspector port.QueueInspector
	Monitor   *queuehealth.Monitor
	Now       func() time.Time

	lastJanitor time.Time
}

// New builds a scheduler that ticks every interval.
func New(interval time.Duration, r Reconciler, j Janitor, qi port.QueueInspector, m *queuehealth.Monitor) *Scheduler {
	return &Scheduler{Interval: interval, Reconcile: r, Janitor: j, Inspector: qi, Monitor: m, Now: time.Now}
}

// Run ticks once at startup and then every Interval until ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context) {
	s.Tick(ctx)
	t := time.NewTicker(s.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.Tick(ctx)
		}
	}
}

// Tick runs one pass. Every step is independent: a failure is logged and never stops the others.
func (s *Scheduler) Tick(ctx context.Context) {
	if n, err := s.Reconcile.Execute(ctx); err != nil {
		slog.ErrorContext(ctx, "reconcile builds failed", "error", err)
	} else if n > 0 {
		timedOutTotal.Add(float64(n))
		slog.WarnContext(ctx, "timed out stuck portfolio builds", "count", n)
	}

	s.checkQueue(ctx)

	now := s.Now()
	if s.lastJanitor.IsZero() || now.Sub(s.lastJanitor) >= JanitorEvery {
		s.lastJanitor = now
		if res, err := s.Janitor.Execute(ctx); err != nil {
			slog.ErrorContext(ctx, "storage janitor failed", "error", err)
		} else {
			slog.InfoContext(ctx, "storage janitor finished", "uploads", res.Uploads, "output_dirs", res.Outputs, "portfolios", res.Portfolios)
		}
	}
}

func (s *Scheduler) checkQueue(ctx context.Context) {
	if s.Inspector == nil {
		return
	}
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	stats, err := s.Inspector.Inspect(cctx)
	if err != nil {
		slog.WarnContext(ctx, "build queue check failed", "error", err)
		return
	}
	queueConsumers.Set(float64(stats.Consumers))
	queueMessages.Set(float64(stats.Messages))
	if s.Monitor.Observe(s.Now(), stats) {
		slog.ErrorContext(ctx, "no portfolio worker has been consuming the build queue for more than 5 minutes; uploads will wait until a worker starts",
			"messages_waiting", stats.Messages)
	}
}
