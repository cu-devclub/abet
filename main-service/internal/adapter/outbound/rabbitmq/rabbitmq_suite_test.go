package rabbitmq_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRabbitMQ(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RabbitMQ Adapter Suite")
}
