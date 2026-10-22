package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/propagation"
)

// HeaderCarrier lets the OpenTelemetry propagator read and write W3C trace context (traceparent, tracestate)
// in the headers of an AMQP message.
type HeaderCarrier amqp.Table

var _ propagation.TextMapCarrier = HeaderCarrier(nil)

func (c HeaderCarrier) Get(key string) string {
	switch v := c[key].(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

func (c HeaderCarrier) Set(key, value string) { c[key] = value }

func (c HeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}
