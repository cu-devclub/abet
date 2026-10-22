package queuehealth_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"abet/main-service/internal/application/queuehealth"
	"abet/main-service/internal/core/port"
)

func TestQueueHealth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Queue health Suite")
}

var _ = Describe("Monitor", func() {
	t0 := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	none := port.QueueStats{Consumers: 0, Messages: 3}
	one := port.QueueStats{Consumers: 1}

	It("is unknown until the first observation, then reports the latest numbers", func() {
		m := queuehealth.New(5 * time.Minute)
		Expect(m.Snapshot().Known).To(BeFalse())
		m.Observe(t0, port.QueueStats{Consumers: 2, Messages: 7})
		Expect(m.Snapshot()).To(Equal(port.QueueSnapshot{Known: true, Consumers: 2, Messages: 7, CheckedAt: t0}))
	})

	It("raises once when consumers stay at 0 for more than the threshold", func() {
		m := queuehealth.New(5 * time.Minute)
		Expect(m.Observe(t0, none)).To(BeFalse())
		Expect(m.Observe(t0.Add(4*time.Minute), none)).To(BeFalse())
		Expect(m.Observe(t0.Add(5*time.Minute), none)).To(BeFalse(), "exactly the threshold is not more than the threshold")
		Expect(m.Observe(t0.Add(6*time.Minute), none)).To(BeTrue())
		Expect(m.Observe(t0.Add(7*time.Minute), none)).To(BeFalse(), "once per incident")
	})

	It("a seen consumer ends the incident and the clock restarts", func() {
		m := queuehealth.New(5 * time.Minute)
		m.Observe(t0, none)
		Expect(m.Observe(t0.Add(6*time.Minute), none)).To(BeTrue())
		Expect(m.Observe(t0.Add(7*time.Minute), one)).To(BeFalse())
		Expect(m.Observe(t0.Add(8*time.Minute), none)).To(BeFalse())
		Expect(m.Observe(t0.Add(12*time.Minute), none)).To(BeFalse())
		Expect(m.Observe(t0.Add(14*time.Minute), none)).To(BeTrue(), "a new incident")
	})
})
