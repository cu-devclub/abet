package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

const (
	prefetch     = 10
	retryPause   = 2 * time.Second // pause before a failed delivery is requeued
	consumerName = "main-service-status"
)

// StatusConsumer reads the worker's status queue, hands each event to the use case and acks only after it succeeded.
type StatusConsumer struct {
	cfg     Config
	handler port.ProcessBuildStatusUseCase
}

func NewStatusConsumer(cfg Config, handler port.ProcessBuildStatusUseCase) *StatusConsumer {
	return &StatusConsumer{cfg: cfg, handler: handler}
}

// statusMessage is the JSON the worker publishes (worker.py publish_status_event).
type statusMessage struct {
	Status          string   `json:"status"`
	JobID           string   `json:"job_id"`
	Error           string   `json:"error"`
	ExportedPDFKeys []string `json:"exported_pdf_keys"`
}

// DecodeStatus parses a status message body.
func DecodeStatus(body []byte) (domain.BuildStatusEvent, error) {
	var m statusMessage
	if err := json.Unmarshal(body, &m); err != nil {
		return domain.BuildStatusEvent{}, err
	}
	return domain.BuildStatusEvent{Status: m.Status, JobID: m.JobID, Error: m.Error, ExportedPDFKeys: m.ExportedPDFKeys}, nil
}

// Run consumes until ctx is cancelled, reconnecting with exponential backoff (1s .. 30s). It returns only when ctx is done
// (after the in-flight delivery was finished and the connection closed).
func (c *StatusConsumer) Run(ctx context.Context) {
	attempt := 0
	for ctx.Err() == nil {
		started, err := c.session(ctx)
		if started {
			attempt = 0
		}
		if ctx.Err() != nil {
			return
		}
		attempt++
		wait := backoff(attempt)
		slog.Warn("status consumer disconnected, retrying", "error", err, "in", wait.String())
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

// session runs one connection until it breaks or ctx ends. started reports whether consuming began.
func (c *StatusConsumer) session(ctx context.Context) (started bool, err error) {
	conn, err := amqp.DialConfig(c.cfg.URL(), amqp.Config{Dial: amqp.DefaultDial(5 * time.Second), Heartbeat: 30 * time.Second})
	if err != nil {
		return false, err
	}
	defer func() { _ = conn.Close() }()
	ch, err := conn.Channel()
	if err != nil {
		return false, err
	}
	if err := declareQueue(ch, c.cfg.StatusQueue); err != nil {
		return false, err
	}
	if err := ch.Qos(prefetch, 0, false); err != nil {
		return false, err
	}
	deliveries, err := ch.ConsumeWithContext(ctx, c.cfg.StatusQueue, consumerName, false, false, false, false, nil)
	if err != nil {
		return false, err
	}
	slog.Info("status consumer started", "queue", c.cfg.StatusQueue)

	closed := conn.NotifyClose(make(chan *amqp.Error, 1))
	for {
		select {
		case <-ctx.Done():
			// Stop taking new deliveries; unacked ones are requeued by the broker when the connection closes.
			_ = ch.Cancel(consumerName, false)
			return true, nil
		case e := <-closed:
			if e == nil {
				return true, errors.New("connection closed")
			}
			return true, e
		case d, ok := <-deliveries:
			if !ok {
				return true, errors.New("delivery channel closed")
			}
			c.handle(ctx, d)
		}
	}
}

// handle processes one delivery: malformed bodies are dropped (ack), handler errors requeue after a short pause.
func (c *StatusConsumer) handle(ctx context.Context, d amqp.Delivery) {
	ev, err := DecodeStatus(d.Body)
	if err != nil {
		slog.Warn("dropping malformed status message", "error", err, "body", truncate(string(d.Body), 300))
		_ = d.Ack(false)
		return
	}
	// An event that was already taken is finished even while the server shuts down.
	pctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Minute)
	defer cancel()
	// Continue the trace the worker propagated in the message headers (no headers = a new root span).
	pctx = otel.GetTextMapPropagator().Extract(pctx, HeaderCarrier(d.Headers))
	pctx, span := otel.Tracer(tracerName).Start(pctx, c.cfg.StatusQueue+" process", trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(attribute.String("messaging.system", "rabbitmq"), attribute.String("messaging.destination.name", c.cfg.StatusQueue),
			attribute.String("messaging.operation", "process"), attribute.String("abet.job_id", ev.JobID), attribute.String("abet.status", ev.Status)))
	defer span.End()
	if err := c.handler.Execute(pctx, ev); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "status event failed")
		slog.ErrorContext(pctx, "status event failed, requeueing", "job_id", ev.JobID, "status", ev.Status, "error", err)
		select {
		case <-ctx.Done():
		case <-time.After(retryPause):
		}
		_ = d.Nack(false, true)
		return
	}
	_ = d.Ack(false)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
