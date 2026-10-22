package rabbitmq

import (
	"context"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestHeaderCarrierRoundTrip(t *testing.T) {
	prop := propagation.TraceContext{}
	tp := sdktrace.NewTracerProvider()
	ctx, span := tp.Tracer("t").Start(context.Background(), "publish")
	defer span.End()

	headers := amqp.Table{}
	prop.Inject(ctx, HeaderCarrier(headers))
	if _, ok := headers["traceparent"].(string); !ok {
		t.Fatalf("traceparent not injected: %v", headers)
	}

	// brokers may hand headers back as []byte; the carrier must cope with both
	headers["traceparent"] = []byte(headers["traceparent"].(string))
	got := trace.SpanContextFromContext(prop.Extract(context.Background(), HeaderCarrier(headers)))
	if !got.IsValid() || got.TraceID() != span.SpanContext().TraceID() || got.SpanID() != span.SpanContext().SpanID() || !got.IsRemote() {
		t.Fatalf("extracted %v, want trace %v span %v", got, span.SpanContext().TraceID(), span.SpanContext().SpanID())
	}

	if (HeaderCarrier(nil)).Get("traceparent") != "" || trace.SpanContextFromContext(prop.Extract(context.Background(), HeaderCarrier(nil))).IsValid() {
		t.Fatal("missing headers must yield no span context")
	}
}
