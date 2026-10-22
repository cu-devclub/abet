package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"abet/main-service/internal/core/port"
)

// QueueInspector reads the job queue's consumer and message counts with a passive declare. It uses a short-lived
// connection per call (a passive declare of a missing queue closes its channel, so nothing is shared or cached).
type QueueInspector struct {
	cfg Config
}

var _ port.QueueInspector = (*QueueInspector)(nil)

func NewQueueInspector(cfg Config) *QueueInspector { return &QueueInspector{cfg: cfg} }

func (q *QueueInspector) Inspect(_ context.Context) (port.QueueStats, error) {
	conn, err := amqp.DialConfig(q.cfg.URL(), amqp.Config{Dial: amqp.DefaultDial(5 * time.Second), Heartbeat: 10 * time.Second})
	if err != nil {
		return port.QueueStats{}, fmt.Errorf("connect broker: %w", err)
	}
	defer func() { _ = conn.Close() }()
	ch, err := conn.Channel()
	if err != nil {
		return port.QueueStats{}, fmt.Errorf("open channel: %w", err)
	}
	info, err := ch.QueueDeclarePassive(q.cfg.JobQueue, true, false, false, false, nil)
	if err != nil {
		// The queue is created by whoever declares it first (a worker or the first publish). Missing = no worker has
		// ever started, i.e. zero consumers and nothing waiting.
		var amqpErr *amqp.Error
		if errors.As(err, &amqpErr) && amqpErr.Code == amqp.NotFound {
			return port.QueueStats{}, nil
		}
		return port.QueueStats{}, fmt.Errorf("inspect queue %s: %w", q.cfg.JobQueue, err)
	}
	return port.QueueStats{Consumers: info.Consumers, Messages: info.Messages}, nil
}
