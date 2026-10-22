package importplan_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestImportPlan(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Import Plan Suite")
}
