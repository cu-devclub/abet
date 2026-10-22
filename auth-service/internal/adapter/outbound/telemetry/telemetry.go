// Package telemetry wires OpenTelemetry tracing (OTLP/HTTP to APM Server) and trace-aware slog records.
// When OTEL_ENABLED is not true nothing is created or dialled: the global no-op tracer stays in place.
package telemetry

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Config is read from the environment (see LoadConfig).
type Config struct {
	Enabled            bool
	Endpoint           string // OTLP/HTTP base URL, e.g. http://apm-server:8200
	SecretToken        string // sent as "Authorization: Bearer <token>"
	ServiceName        string
	ResourceAttributes string // k=v,k=v
	SampleRatio        float64
}

// LoadConfig reads OTEL_ENABLED, OTEL_EXPORTER_OTLP_ENDPOINT, APM_SECRET_TOKEN, OTEL_SERVICE_NAME,
// OTEL_RESOURCE_ATTRIBUTES and OTEL_SAMPLE_RATIO.
func LoadConfig(defaultService string) Config {
	ratio := 1.0
	if v, err := strconv.ParseFloat(strings.TrimSpace(os.Getenv("OTEL_SAMPLE_RATIO")), 64); err == nil && v >= 0 && v <= 1 {
		ratio = v
	}
	name := os.Getenv("OTEL_SERVICE_NAME")
	if name == "" {
		name = defaultService
	}
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://apm-server:8200"
	}
	return Config{
		Enabled:            strings.EqualFold(strings.TrimSpace(os.Getenv("OTEL_ENABLED")), "true"),
		Endpoint:           endpoint,
		SecretToken:        os.Getenv("APM_SECRET_TOKEN"),
		ServiceName:        name,
		ResourceAttributes: os.Getenv("OTEL_RESOURCE_ATTRIBUTES"),
		SampleRatio:        ratio,
	}
}

// Setup installs the W3C propagators and, when enabled, the tracer provider with the OTLP/HTTP exporter.
// The returned shutdown function flushes pending spans; it is never nil.
func Setup(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	noop := func(context.Context) error { return nil }
	if !cfg.Enabled {
		return noop, nil
	}

	opts := []otlptracehttp.Option{otlptracehttp.WithEndpointURL(strings.TrimRight(cfg.Endpoint, "/") + "/v1/traces")}
	if cfg.SecretToken != "" {
		opts = append(opts, otlptracehttp.WithHeaders(map[string]string{"Authorization": "Bearer " + cfg.SecretToken}))
	}
	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return noop, fmt.Errorf("create trace exporter: %w", err)
	}

	attrs := []attribute.KeyValue{attribute.String("service.name", cfg.ServiceName)}
	for _, kv := range strings.Split(cfg.ResourceAttributes, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(kv), "=")
		if ok && k != "" {
			attrs = append(attrs, attribute.String(k, v))
		}
	}
	res, err := resource.New(ctx, resource.WithAttributes(attrs...))
	if err != nil {
		return noop, fmt.Errorf("create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRatio))),
	)
	otel.SetTracerProvider(tp)
	return func(c context.Context) error {
		c, cancel := context.WithTimeout(c, 5*time.Second)
		defer cancel()
		return tp.Shutdown(c)
	}, nil
}
