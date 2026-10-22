package commands_test

import (
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"abet/main-service/internal/core/domain"
)

func TestCommands(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Commands Suite")
}

// Shared fixtures for the command specs.
var (
	superActor = &domain.Actor{UserID: uuid.New(), Email: "root@chula.ac.th", Role: domain.RoleSuperAdmin}
	staffActor = &domain.Actor{UserID: uuid.New(), Email: "staff@chula.ac.th", Role: domain.RoleStaff}
)

func ptr[T any](v T) *T { return &v }
