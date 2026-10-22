package domain_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"abet/main-service/internal/core/domain"
)

var _ = Describe("department and curriculum entities", func() {
	It("normalises department code to upper case and trims", func() {
		d, err := domain.NewDepartment("  math-cs ", " Math & CS ")
		Expect(err).NotTo(HaveOccurred())
		Expect(d.Code).To(Equal("MATH-CS"))
		Expect(d.Name).To(Equal("Math & CS"))
	})

	It("rejects empty code or name", func() {
		_, err := domain.NewDepartment("  ", "x")
		Expect(err).To(MatchError(domain.ErrInvalidInput))
		_, err = domain.NewDepartment("x", "")
		Expect(err).To(MatchError(domain.ErrInvalidInput))
	})

	It("applies partial updates", func() {
		d, _ := domain.NewDepartment("a", "Alpha")
		newName := "Beta"
		Expect(d.Apply(nil, &newName)).To(Succeed())
		Expect(d.Code).To(Equal("A"))
		Expect(d.Name).To(Equal("Beta"))
		empty := " "
		Expect(d.Apply(&empty, nil)).To(MatchError(domain.ErrInvalidInput))
	})

	It("builds and updates curricula", func() {
		dep := uuid.New()
		c, err := domain.NewCurriculum("bsc-cs", "BSc CS", dep)
		Expect(err).NotTo(HaveOccurred())
		Expect(c.Code).To(Equal("BSC-CS"))
		other := uuid.New()
		Expect(c.Apply(nil, nil, &other)).To(Succeed())
		Expect(c.DepartmentID).To(Equal(other))
	})

	It("ranks roles", func() {
		Expect(domain.RoleSuperAdmin.Rank()).To(BeNumerically(">", domain.RoleAdmin.Rank()))
		Expect(domain.RoleAdmin.Rank()).To(BeNumerically(">", domain.RoleInstructor.Rank()))
		Expect(domain.RoleInstructor.Rank()).To(BeNumerically(">", domain.RoleStaff.Rank()))
		Expect(domain.Role("x").Rank()).To(Equal(0))
		Expect(domain.Role("x").Valid()).To(BeFalse())
	})
})
