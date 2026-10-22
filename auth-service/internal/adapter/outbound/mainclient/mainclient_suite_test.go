package mainclient_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMainClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Main Service Client Suite")
}
