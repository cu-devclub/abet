package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

// StartActAsHandler implements port.StartActAsUseCase.
type StartActAsHandler struct {
	repo port.UserRepository
}

var _ port.StartActAsUseCase = (*StartActAsHandler)(nil)

func NewStartActAsHandler(repo port.UserRepository) *StartActAsHandler {
	return &StartActAsHandler{repo: repo}
}

func (h *StartActAsHandler) Execute(ctx context.Context, real *domain.Actor, targetID uuid.UUID) error {
	if real == nil || real.Role != domain.RoleSuperAdmin {
		return domain.ErrForbidden
	}
	if targetID == real.UserID {
		return fmt.Errorf("%w: already yourself, use DELETE /dev/act-as to go back", domain.ErrInvalidInput)
	}
	if _, err := h.repo.GetByID(ctx, targetID); err != nil {
		return err // ErrNotFound for unknown ids
	}
	return nil
}
