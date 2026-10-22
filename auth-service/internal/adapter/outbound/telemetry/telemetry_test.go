package telemetry_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"abet/auth-service/internal/adapter/outbound/telemetry"
)

func TestSetupDisabledInstallsNoExporter(t *testing.T) {
	shutdown, err := telemetry.Setup(context.Background(), telemetry.Config{Enabled: false, Endpoint: "http://127.0.0.1:1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, isSDK := otel.GetTracerProvider().(*sdktrace.TracerProvider); isSDK {
		t.Fatal("SDK tracer provider installed although OTEL_ENABLED=false")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestTraceHandlerAddsIDs(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	ctx, span := tp.Tracer("t").Start(context.Background(), "op")
	defer span.End()

	var buf bytes.Buffer
	l := slog.New(telemetry.NewTraceHandler(slog.NewJSONHandler(&buf, nil)))
	l.InfoContext(ctx, "with span")
	l.Info("without span")

	var lines []map[string]any
	for _, b := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, m)
	}
	if lines[0]["trace_id"] != span.SpanContext().TraceID().String() || lines[0]["span_id"] != span.SpanContext().SpanID().String() {
		t.Fatalf("ids missing/wrong: %v", lines[0])
	}
	if _, ok := lines[1]["trace_id"]; ok {
		t.Fatalf("trace_id must be absent without a span: %v", lines[1])
	}
}
