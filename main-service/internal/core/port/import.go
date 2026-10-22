package port

import (
	"context"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
)

// ---- outbound ----

// ApplyImportParams describes one import transaction.
type ApplyImportParams struct {
	Curriculum   *domain.Curriculum // scope; its department is used for auto-created users (D10)
	AcademicYear int
	Emails       []string // e-mails mentioned in the CSV
	// Build is called inside the transaction with the freshly loaded (and locked) state; it must be pure.
	// When the returned plan has errors nothing is written and ErrUnprocessable is returned together with the plan.
	Build func(state *domain.ImportState) domain.ImportPlan
}

// ImportRepository loads the import scope and applies a plan atomically.
type ImportRepository interface {
	// LoadState reads (without locking) the offerings, their access rows and the users with the given e-mails.
	LoadState(ctx context.Context, curriculumID uuid.UUID, academicYear int, emails []string) (*domain.ImportState, error)
	// Apply locks the offerings of the scope (SELECT ... FOR UPDATE), loads the state, builds the plan and executes
	// it, all in one transaction. The returned plan has Applied=true on success.
	Apply(ctx context.Context, p ApplyImportParams) (*domain.ImportPlan, error)
}

// ---- inbound ----

// DefaultMaxImportBytes is the default upload limit of the mapping CSV (MAX_IMPORT_MB=5); the live limit comes from config.
const DefaultMaxImportBytes = 5 << 20

type ImportInput struct {
	CurriculumID uuid.UUID
	AcademicYear int
	Data         []byte // the CSV file content; never stored
}

// PreviewImportUseCase computes the plan without writing anything.
type PreviewImportUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, in ImportInput) (*domain.ImportPlan, error)
}

// ApplyImportUseCase executes the plan. When the CSV has row errors it returns the plan together with
// domain.ErrUnprocessable (nothing applied).
type ApplyImportUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor, in ImportInput) (*domain.ImportPlan, error)
}
