// Package rabbitmq is the AMQP adapter towards the portfolio worker: a confirm-mode job publisher and a
// status-queue consumer. Queues are declared durable with NO extra arguments, exactly like worker.py does
// (a differing declaration would make the broker close the channel with PRECONDITION_FAILED).
package rabbitmq

import (
	"net"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Config holds the broker connection settings and queue names.
type Config struct {
	Host        string
	Port        string
	User        string
	Pass        string
	VHost       string
	JobQueue    string
	StatusQueue string
}

// URL builds the AMQP URI (credentials and vhost are escaped by amqp.URI).
func (c Config) URL() string {
	vhost := c.VHost
	if vhost == "" {
		vhost = "/"
	}
	return amqp.URI{
		Scheme:   "amqp",
		Host:     c.Host,
		Port:     portNumber(c.Port),
		Username: c.User,
		Password: c.Pass,
		Vhost:    vhost,
	}.String()
}

func portNumber(p string) int {
	n, err := net.LookupPort("tcp", p)
	if err != nil || n == 0 {
		return 5672
	}
	return n
}

// backoff returns the reconnect delay for the given attempt (1-based): 1s, 2s, 4s ... capped at 30s.
func backoff(attempt int) time.Duration {
	const max = 30 * time.Second
	if attempt < 1 {
		attempt = 1
	}
	d := time.Second
	for i := 1; i < attempt; i++ {
		d *= 2
		if d >= max {
			return max
		}
	}
	return d
}

// declareQueue declares a durable queue with no arguments (must match worker.py).
func declareQueue(ch *amqp.Channel, name string) error {
	_, err := ch.QueueDeclare(name, true, false, false, false, nil)
	return err
}
