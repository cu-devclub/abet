package scheduler_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"abet/main-service/internal/adapter/inbound/scheduler"
	"abet/main-service/internal/application/commands"
	"abet/main-service/internal/application/queuehealth"
	"abet/main-service/internal/core/port"
)

func TestScheduler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scheduler Suite")
}

type fakeReconciler struct {
	calls atomic.Int32
	err   error
}

func (f *fakeReconciler) Execute(context.Context) (int, error) { f.calls.Add(1); return 1, f.err }

type fakeJanitor struct{ calls atomic.Int32 }

func (f *fakeJanitor) Execute(context.Context) (commands.JanitorResult, error) {
	f.calls.Add(1)
	return commands.JanitorResult{}, nil
}

type fakeInspector struct {
	stats port.QueueStats
	err   error
}

func (f *fakeInspector) Inspect(context.Context) (port.QueueStats, error) { return f.stats, f.err }

var _ = Describe("Scheduler", func() {
	var (
		now  time.Time
		rec  *fakeReconciler
		jan  *fakeJanitor
		insp *fakeInspector
		mon  *queuehealth.Monitor
		s    *scheduler.Scheduler
	)
	BeforeEach(func() {
		now = time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
		rec, jan, insp = &fakeReconciler{}, &fakeJanitor{}, &fakeInspector{stats: port.QueueStats{Consumers: 2, Messages: 4}}
		mon = queuehealth.New(5 * time.Minute)
		s = scheduler.New(time.Minute, rec, jan, insp, mon)
		s.Now = func() time.Time { return now }
	})

	It("reconciles and checks the queue on every tick but runs the janitor hourly", func() {
		s.Tick(context.Background())
		Expect(rec.calls.Load()).To(BeEquivalentTo(1))
		Expect(jan.calls.Load()).To(BeEquivalentTo(1), "the first tick runs the janitor")
		Expect(mon.Snapshot()).To(Equal(port.QueueSnapshot{Known: true, Consumers: 2, Messages: 4, CheckedAt: now}))

		now = now.Add(30 * time.Minute)
		s.Tick(context.Background())
		Expect(rec.calls.Load()).To(BeEquivalentTo(2))
		Expect(jan.calls.Load()).To(BeEquivalentTo(1))

		now = now.Add(31 * time.Minute)
		s.Tick(context.Background())
		Expect(jan.calls.Load()).To(BeEquivalentTo(2))
	})

	It("keeps going when a step fails", func() {
		rec.err = errors.New("db")
		insp.err = errors.New("broker")
		s.Tick(context.Background())
		Expect(jan.calls.Load()).To(BeEquivalentTo(1))
		Expect(mon.Snapshot().Known).To(BeFalse(), "a failed check does not overwrite the last known state")
	})

	It("Run ticks at once and stops with the context", func() {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() { s.Run(ctx); close(done) }()
		Eventually(func() int32 { return rec.calls.Load() }).Should(BeNumerically(">=", 1))
		cancel()
		Eventually(done).Should(BeClosed())
	})
})
