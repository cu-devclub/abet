package commands_test

import (
	"context"
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

var _ = Describe("access commands", func() {
	var (
		ctrl      *gomock.Controller
		offerings *mocks.MockOfferingRepository
		access    *mocks.MockAccessRepository
		users     *mocks.MockUserRepository
		ctx       = context.Background()
		curID     = uuid.New()
		off       *domain.Offering
	)
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		offerings = mocks.NewMockOfferingRepository(ctrl)
		access = mocks.NewMockAccessRepository(ctrl)
		users = mocks.NewMockUserRepository(ctrl)
		off = offeringIn(curID)
	})
	AfterEach(func() { ctrl.Finish() })

	Describe("set", func() {
		in := func() port.SetAccessInput { return port.SetAccessInput{CanUpload: true, CanDownload: true} }
		var target *domain.User
		BeforeEach(func() {
			target = &domain.User{ID: uuid.New(), Email: "i@chula.ac.th", Name: "I", Role: domain.RoleInstructor, CurriculumID: &curID}
		})
		run := func(actor *domain.Actor, i port.SetAccessInput) (*domain.AccessEntry, error) {
			i.OfferingID, i.UserID = off.ID, target.ID
			return commands.NewSetAccessHandler(offerings, access, users).Execute(ctx, actor, i)
		}

		It("upserts a manual row for an instructor the admin can see", func() {
			offerings.EXPECT().GetByID(ctx, off.ID).Return(off, nil)
			users.EXPECT().GetByID(ctx, target.ID).Return(target, nil)
			access.EXPECT().Upsert(ctx, off.ID, target.ID, true, true).Return(nil)
			e, err := run(adminOf(curID), in())
			Expect(err).NotTo(HaveOccurred())
			Expect(e.Source).To(Equal("manual"))
			Expect(e.Email).To(Equal("i@chula.ac.th"))
			Expect(e.Role).To(Equal(domain.RoleInstructor))
		})

		It("is forbidden for an admin without the curriculum and for non admins", func() {
			offerings.EXPECT().GetByID(ctx, off.ID).Return(off, nil).Times(2)
			_, err := run(adminOf(uuid.New()), in())
			Expect(err).To(MatchError(domain.ErrForbidden))
			_, err = run(staffActor, in())
			Expect(err).To(MatchError(domain.ErrForbidden))
		})

		It("answers 404 for an unknown offering or user", func() {
			offerings.EXPECT().GetByID(ctx, off.ID).Return(nil, domain.ErrNotFound)
			_, err := run(superActor, in())
			Expect(err).To(MatchError(domain.ErrNotFound))

			offerings.EXPECT().GetByID(ctx, off.ID).Return(off, nil)
			users.EXPECT().GetByID(ctx, target.ID).Return(nil, domain.ErrNotFound)
			_, err = run(superActor, in())
			Expect(err).To(MatchError(domain.ErrNotFound))
		})

		It("requires at least one permission", func() {
			offerings.EXPECT().GetByID(ctx, off.ID).Return(off, nil)
			_, err := run(superActor, port.SetAccessInput{})
			Expect(err).To(MatchError(domain.ErrInvalidInput))
		})

		It("only accepts instructors and staff as targets", func() {
			for _, r := range []domain.Role{domain.RoleAdmin, domain.RoleSuperAdmin} {
				t := *target
				t.Role = r
				offerings.EXPECT().GetByID(ctx, off.ID).Return(off, nil)
				users.EXPECT().GetByID(ctx, target.ID).Return(&t, nil)
				_, err := run(superActor, in())
				Expect(err).To(MatchError(domain.ErrInvalidInput))
			}
		})

		It("forbids an admin from assigning a user outside their scope", func() {
			target.CurriculumID = ptr(uuid.New())
			offerings.EXPECT().GetByID(ctx, off.ID).Return(off, nil)
			users.EXPECT().GetByID(ctx, target.ID).Return(target, nil)
			_, err := run(adminOf(curID), in())
			Expect(err).To(MatchError(domain.ErrForbidden))
		})

		It("lets a super admin assign anybody", func() {
			target.CurriculumID = nil
			offerings.EXPECT().GetByID(ctx, off.ID).Return(off, nil)
			users.EXPECT().GetByID(ctx, target.ID).Return(target, nil)
			access.EXPECT().Upsert(ctx, off.ID, target.ID, true, true).Return(nil)
			_, err := run(superActor, in())
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("remove", func() {
		uid := uuid.New()

		It("deletes the explicit row", func() {
			offerings.EXPECT().GetByID(ctx, off.ID).Return(off, nil)
			access.EXPECT().Delete(ctx, off.ID, uid).Return(nil)
			Expect(commands.NewRemoveAccessHandler(offerings, access).Execute(ctx, adminOf(curID), off.ID, uid)).To(Succeed())
		})

		It("answers 404 for an unknown offering or a missing row, 403 out of scope", func() {
			h := commands.NewRemoveAccessHandler(offerings, access)
			offerings.EXPECT().GetByID(ctx, off.ID).Return(nil, domain.ErrNotFound)
			Expect(h.Execute(ctx, superActor, off.ID, uid)).To(MatchError(domain.ErrNotFound))

			offerings.EXPECT().GetByID(ctx, off.ID).Return(off, nil)
			access.EXPECT().Delete(ctx, off.ID, uid).Return(fmt.Errorf("x: %w", domain.ErrNotFound))
			Expect(h.Execute(ctx, superActor, off.ID, uid)).To(MatchError(domain.ErrNotFound))

			offerings.EXPECT().GetByID(ctx, off.ID).Return(off, nil)
			Expect(h.Execute(ctx, adminOf(uuid.New()), off.ID, uid)).To(MatchError(domain.ErrForbidden))
		})
	})
})
