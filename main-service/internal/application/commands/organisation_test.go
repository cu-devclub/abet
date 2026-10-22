package commands_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"abet/main-service/internal/application/commands"
	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
	"abet/main-service/internal/core/port/mocks"
)

var _ = Describe("department commands", func() {
	var (
		ctrl *gomock.Controller
		repo *mocks.MockDepartmentRepository
		ctx  = context.Background()
	)
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		repo = mocks.NewMockDepartmentRepository(ctrl)
	})
	AfterEach(func() { ctrl.Finish() })

	Describe("create", func() {
		It("creates a department with an upper-cased code", func() {
			repo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, d *domain.Department) error {
				Expect(d.Code).To(Equal("MATH-CS"))
				return nil
			})
			d, err := commands.NewCreateDepartmentHandler(repo).Execute(ctx, superActor, port.CreateDepartmentInput{Code: " math-cs ", Name: "Math"})
			Expect(err).NotTo(HaveOccurred())
			Expect(d.Code).To(Equal("MATH-CS"))
		})

		It("is forbidden for non super admins and never touches the repo", func() {
			_, err := commands.NewCreateDepartmentHandler(repo).Execute(ctx, staffActor, port.CreateDepartmentInput{Code: "X", Name: "X"})
			Expect(err).To(MatchError(domain.ErrForbidden))
		})

		It("rejects an empty name with ErrInvalidInput", func() {
			_, err := commands.NewCreateDepartmentHandler(repo).Execute(ctx, superActor, port.CreateDepartmentInput{Code: "X", Name: " "})
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})

		It("maps a duplicate code to a conflict", func() {
			repo.EXPECT().Create(ctx, gomock.Any()).Return(fmt.Errorf("create: %w", domain.ErrConflict))
			_, err := commands.NewCreateDepartmentHandler(repo).Execute(ctx, superActor, port.CreateDepartmentInput{Code: "X", Name: "X"})
			Expect(err).To(MatchError(domain.ErrConflict))
			Expect(err.Error()).To(ContainSubstring("department code already exists"))
		})

		It("passes unexpected repository errors through", func() {
			boom := errors.New("db down")
			repo.EXPECT().Create(ctx, gomock.Any()).Return(boom)
			_, err := commands.NewCreateDepartmentHandler(repo).Execute(ctx, superActor, port.CreateDepartmentInput{Code: "X", Name: "X"})
			Expect(err).To(MatchError(boom))
		})
	})

	Describe("update", func() {
		var id = uuid.New()

		It("applies a partial update", func() {
			existing := &domain.Department{ID: id, Code: "OLD", Name: "Old"}
			repo.EXPECT().GetByID(ctx, id).Return(existing, nil)
			repo.EXPECT().Update(ctx, existing).Return(nil)
			d, err := commands.NewUpdateDepartmentHandler(repo).Execute(ctx, superActor, port.UpdateDepartmentInput{ID: id, Name: ptr("New")})
			Expect(err).NotTo(HaveOccurred())
			Expect(d.Code).To(Equal("OLD"))
			Expect(d.Name).To(Equal("New"))
		})

		It("is forbidden for admins", func() {
			admin := &domain.Actor{UserID: uuid.New(), Role: domain.RoleAdmin}
			_, err := commands.NewUpdateDepartmentHandler(repo).Execute(ctx, admin, port.UpdateDepartmentInput{ID: id})
			Expect(err).To(MatchError(domain.ErrForbidden))
		})

		It("reports a missing department as not found", func() {
			repo.EXPECT().GetByID(ctx, id).Return(nil, domain.ErrNotFound)
			_, err := commands.NewUpdateDepartmentHandler(repo).Execute(ctx, superActor, port.UpdateDepartmentInput{ID: id})
			Expect(err).To(MatchError(domain.ErrNotFound))
		})

		It("maps a colliding code to a conflict", func() {
			repo.EXPECT().GetByID(ctx, id).Return(&domain.Department{ID: id, Code: "A", Name: "A"}, nil)
			repo.EXPECT().Update(ctx, gomock.Any()).Return(domain.ErrConflict)
			_, err := commands.NewUpdateDepartmentHandler(repo).Execute(ctx, superActor, port.UpdateDepartmentInput{ID: id, Code: ptr("B")})
			Expect(err).To(MatchError(domain.ErrConflict))
		})

		It("rejects blank values", func() {
			repo.EXPECT().GetByID(ctx, id).Return(&domain.Department{ID: id, Code: "A", Name: "A"}, nil)
			_, err := commands.NewUpdateDepartmentHandler(repo).Execute(ctx, superActor, port.UpdateDepartmentInput{ID: id, Code: ptr("  ")})
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})
	})

	Describe("delete", func() {
		var id = uuid.New()

		It("deletes", func() {
			repo.EXPECT().Delete(ctx, id).Return(nil)
			Expect(commands.NewDeleteDepartmentHandler(repo).Execute(ctx, superActor, id)).To(Succeed())
		})

		It("is forbidden for non super admins", func() {
			Expect(commands.NewDeleteDepartmentHandler(repo).Execute(ctx, staffActor, id)).To(MatchError(domain.ErrForbidden))
		})

		It("answers 409 text when the department is referenced", func() {
			repo.EXPECT().Delete(ctx, id).Return(domain.ErrConflict)
			err := commands.NewDeleteDepartmentHandler(repo).Execute(ctx, superActor, id)
			Expect(err).To(MatchError(domain.ErrConflict))
			Expect(err.Error()).To(ContainSubstring("department is referenced by curricula or users"))
		})

		It("reports a missing department as not found", func() {
			repo.EXPECT().Delete(ctx, id).Return(domain.ErrNotFound)
			Expect(commands.NewDeleteDepartmentHandler(repo).Execute(ctx, superActor, id)).To(MatchError(domain.ErrNotFound))
		})
	})
})

var _ = Describe("curriculum commands", func() {
	var (
		ctrl    *gomock.Controller
		repo    *mocks.MockCurriculumRepository
		depts   *mocks.MockDepartmentRepository
		storage *mocks.MockStorage
		ctx     = context.Background()
		deptID  = uuid.New()
	)
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		repo = mocks.NewMockCurriculumRepository(ctrl)
		depts = mocks.NewMockDepartmentRepository(ctrl)
		storage = mocks.NewMockStorage(ctrl)
	})
	AfterEach(func() { ctrl.Finish() })

	Describe("create", func() {
		It("creates and returns the stored curriculum", func() {
			depts.EXPECT().GetByID(ctx, deptID).Return(&domain.Department{ID: deptID}, nil)
			var created uuid.UUID
			repo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, c *domain.Curriculum) error {
				created = c.ID
				Expect(c.Code).To(Equal("BSC"))
				return nil
			})
			repo.EXPECT().GetByID(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, id uuid.UUID) (*domain.Curriculum, error) {
				Expect(id).To(Equal(created))
				return &domain.Curriculum{ID: id, Code: "BSC"}, nil
			})
			c, err := commands.NewCreateCurriculumHandler(repo, depts).Execute(ctx, superActor, port.CreateCurriculumInput{Code: "bsc", Name: "B", DepartmentID: deptID})
			Expect(err).NotTo(HaveOccurred())
			Expect(c.Code).To(Equal("BSC"))
		})

		It("is forbidden for admins", func() {
			admin := &domain.Actor{UserID: uuid.New(), Role: domain.RoleAdmin, ScopeDepartmentIDs: []uuid.UUID{deptID}}
			_, err := commands.NewCreateCurriculumHandler(repo, depts).Execute(ctx, admin, port.CreateCurriculumInput{Code: "x", Name: "x", DepartmentID: deptID})
			Expect(err).To(MatchError(domain.ErrForbidden))
		})

		It("rejects an unknown department with ErrInvalidInput", func() {
			depts.EXPECT().GetByID(ctx, deptID).Return(nil, domain.ErrNotFound)
			_, err := commands.NewCreateCurriculumHandler(repo, depts).Execute(ctx, superActor, port.CreateCurriculumInput{Code: "x", Name: "x", DepartmentID: deptID})
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})

		It("rejects blank code", func() {
			_, err := commands.NewCreateCurriculumHandler(repo, depts).Execute(ctx, superActor, port.CreateCurriculumInput{Code: "", Name: "x", DepartmentID: deptID})
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})

		It("maps duplicate code to conflict", func() {
			depts.EXPECT().GetByID(ctx, deptID).Return(&domain.Department{ID: deptID}, nil)
			repo.EXPECT().Create(ctx, gomock.Any()).Return(domain.ErrConflict)
			_, err := commands.NewCreateCurriculumHandler(repo, depts).Execute(ctx, superActor, port.CreateCurriculumInput{Code: "x", Name: "x", DepartmentID: deptID})
			Expect(err).To(MatchError(domain.ErrConflict))
		})
	})

	Describe("update", func() {
		id := uuid.New()
		newDept := uuid.New()

		It("moves a curriculum to another existing department", func() {
			existing := &domain.Curriculum{ID: id, Code: "A", Name: "A", DepartmentID: deptID}
			repo.EXPECT().GetByID(ctx, id).Return(existing, nil).Times(2)
			depts.EXPECT().GetByID(ctx, newDept).Return(&domain.Department{ID: newDept}, nil)
			repo.EXPECT().Update(ctx, existing).Return(nil)
			c, err := commands.NewUpdateCurriculumHandler(repo, depts).Execute(ctx, superActor, port.UpdateCurriculumInput{ID: id, DepartmentID: &newDept})
			Expect(err).NotTo(HaveOccurred())
			Expect(c.DepartmentID).To(Equal(newDept))
		})

		It("rejects a move to an unknown department", func() {
			repo.EXPECT().GetByID(ctx, id).Return(&domain.Curriculum{ID: id, Code: "A", Name: "A", DepartmentID: deptID}, nil)
			depts.EXPECT().GetByID(ctx, newDept).Return(nil, domain.ErrNotFound)
			_, err := commands.NewUpdateCurriculumHandler(repo, depts).Execute(ctx, superActor, port.UpdateCurriculumInput{ID: id, DepartmentID: &newDept})
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})

		It("is forbidden for non super admins", func() {
			_, err := commands.NewUpdateCurriculumHandler(repo, depts).Execute(ctx, staffActor, port.UpdateCurriculumInput{ID: id})
			Expect(err).To(MatchError(domain.ErrForbidden))
		})

		It("reports missing curriculum as not found", func() {
			repo.EXPECT().GetByID(ctx, id).Return(nil, domain.ErrNotFound)
			_, err := commands.NewUpdateCurriculumHandler(repo, depts).Execute(ctx, superActor, port.UpdateCurriculumInput{ID: id})
			Expect(err).To(MatchError(domain.ErrNotFound))
		})

		It("maps a colliding code to conflict", func() {
			repo.EXPECT().GetByID(ctx, id).Return(&domain.Curriculum{ID: id, Code: "A", Name: "A", DepartmentID: deptID}, nil)
			repo.EXPECT().Update(ctx, gomock.Any()).Return(domain.ErrConflict)
			_, err := commands.NewUpdateCurriculumHandler(repo, depts).Execute(ctx, superActor, port.UpdateCurriculumInput{ID: id, Code: ptr("B")})
			Expect(err).To(MatchError(domain.ErrConflict))
		})
	})

	Describe("delete", func() {
		id := uuid.New()

		It("deletes the row and the cover file", func() {
			repo.EXPECT().Delete(ctx, id).Return(nil)
			storage.EXPECT().Delete(ctx, "covers/"+id.String()+".pdf").Return(nil)
			Expect(commands.NewDeleteCurriculumHandler(repo, storage).Execute(ctx, superActor, id)).To(Succeed())
		})

		It("does not fail when the cover file cannot be removed", func() {
			repo.EXPECT().Delete(ctx, id).Return(nil)
			storage.EXPECT().Delete(ctx, gomock.Any()).Return(errors.New("disk"))
			Expect(commands.NewDeleteCurriculumHandler(repo, storage).Execute(ctx, superActor, id)).To(Succeed())
		})

		It("is forbidden for non super admins", func() {
			Expect(commands.NewDeleteCurriculumHandler(repo, storage).Execute(ctx, staffActor, id)).To(MatchError(domain.ErrForbidden))
		})

		It("answers conflict when referenced by users or courses and keeps the cover", func() {
			repo.EXPECT().Delete(ctx, id).Return(domain.ErrConflict)
			err := commands.NewDeleteCurriculumHandler(repo, storage).Execute(ctx, superActor, id)
			Expect(err).To(MatchError(domain.ErrConflict))
			Expect(err.Error()).To(ContainSubstring("referenced by users or courses"))
		})

		It("reports a missing curriculum as not found", func() {
			repo.EXPECT().Delete(ctx, id).Return(domain.ErrNotFound)
			Expect(commands.NewDeleteCurriculumHandler(repo, storage).Execute(ctx, superActor, id)).To(MatchError(domain.ErrNotFound))
		})
	})
})
