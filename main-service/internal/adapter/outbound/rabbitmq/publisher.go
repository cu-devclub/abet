package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
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
	publishTimeout = 10 * time.Second
	tracerName     = "abet/main-service/rabbitmq"
)

// Publisher sends build jobs. It connects lazily and reconnects on the next call after a failure,
// so the API keeps starting (and answering with 201 + QUEUE_UNAVAILABLE issues) while the broker is down.
type Publisher struct {
	cfg Config

	mu   sync.Mutex
	conn *amqp.Connection
	ch   *amqp.Channel
}

var _ port.JobPublisher = (*Publisher)(nil)

func NewPublisher(cfg Config) *Publisher { return &Publisher{cfg: cfg} }

// jobMessage is the JSON body the worker reads (worker.py process_job). No callback_url: results arrive on the status queue.
type jobMessage struct {
	JobID                 string `json:"job_id"`
	ProgramName           string `json:"program_name"`
	Storage               string `json:"storage"`
	ZipKey                string `json:"zip_key"`
	CoverKey              string `json:"cover_key,omitempty"`
	OutputPrefix          string `json:"output_prefix"`
	DeleteZipOnCompletion bool   `json:"delete_zip_on_completion"`
}

// EncodeJob renders a job as the worker's JSON message.
func EncodeJob(job domain.BuildJob) ([]byte, error) {
	return json.Marshal(jobMessage{
		JobID: job.JobID.String(), ProgramName: job.ProgramName, Storage: "s3", ZipKey: job.ZipKey, CoverKey: job.CoverKey,
		OutputPrefix: job.OutputPrefix, DeleteZipOnCompletion: job.DeleteZipOnCompletion,
	})
}

// Publish sends the job as a persistent application/json message and waits for the broker's publisher confirm.
func (p *Publisher) Publish(ctx context.Context, job domain.BuildJob) error {
	body, err := EncodeJob(job)
	if err != nil {
		return fmt.Errorf("encode job: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()
	ctx, span := otel.Tracer(tracerName).Start(ctx, p.cfg.JobQueue+" publish", trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(attribute.String("messaging.system", "rabbitmq"), attribute.String("messaging.destination.name", p.cfg.JobQueue),
			attribute.String("messaging.operation", "publish"), attribute.String("abet.job_id", job.JobID.String())))
	defer span.End()

	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.publishLocked(ctx, body); err != nil {
		p.closeLocked()
		span.RecordError(err)
		span.SetStatus(codes.Error, "publish failed")
		slog.ErrorContext(ctx, "publish build job failed", "job_id", job.JobID, "error", err)
		return fmt.Errorf("%w: publish job: %v", domain.ErrServiceUnavailable, err)
	}
	return nil
}

func (p *Publisher) publishLocked(ctx context.Context, body []byte) error {
	if err := p.ensureLocked(); err != nil {
		return err
	}
	headers := amqp.Table{}
	otel.GetTextMapPropagator().Inject(ctx, HeaderCarrier(headers)) // W3C traceparent/tracestate for the worker
	confirm, err := p.ch.PublishWithDeferredConfirmWithContext(ctx, "", p.cfg.JobQueue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now(),
		Headers:      headers,
		Body:         body,
	})
	if err != nil {
		return err
	}
	acked, err := confirm.WaitContext(ctx)
	if err != nil {
		return err
	}
	if !acked {
		return fmt.Errorf("broker nacked the message")
	}
	return nil
}

func (p *Publisher) ensureLocked() error {
	if p.conn != nil && !p.conn.IsClosed() && p.ch != nil && !p.ch.IsClosed() {
		return nil
	}
	p.closeLocked()
	conn, err := amqp.DialConfig(p.cfg.URL(), amqp.Config{Dial: amqp.DefaultDial(5 * time.Second), Heartbeat: 30 * time.Second})
	if err != nil {
		return fmt.Errorf("dial rabbitmq: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("open channel: %w", err)
	}
	if err := declareQueue(ch, p.cfg.JobQueue); err != nil {
		_ = conn.Close()
		return fmt.Errorf("declare queue %q: %w", p.cfg.JobQueue, err)
	}
	if err := ch.Confirm(false); err != nil {
		_ = conn.Close()
		return fmt.Errorf("enable publisher confirms: %w", err)
	}
	p.conn, p.ch = conn, ch
	return nil
}

func (p *Publisher) closeLocked() {
	if p.conn != nil && !p.conn.IsClosed() {
		_ = p.conn.Close()
	}
	p.conn, p.ch = nil, nil
}

// Close releases the connection.
func (p *Publisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closeLocked()
}
