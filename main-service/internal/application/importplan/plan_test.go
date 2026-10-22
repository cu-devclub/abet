package importplan_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"abet/main-service/internal/application/importplan"
	"abet/main-service/internal/core/domain"
)

func row(n int, code, name string, sem int, email string, role domain.Role, up, down bool) domain.ImportRow {
	return domain.ImportRow{Row: n, CourseCode: code, CourseName: name, Semester: sem, Email: email, Role: role, CanUpload: up, CanDownload: down}
}

func off(code, name string, sem int, uploads bool, access ...domain.ImportAccessState) domain.ImportOfferingState {
	return domain.ImportOfferingState{
		OfferingID: uuid.New(), CourseID: uuid.New(), CourseCode: code, CourseName: name, Semester: sem,
		HasSubmissions: uploads, Access: access,
	}
}

func acc(email string, role domain.Role, up, down bool) domain.ImportAccessState {
	return domain.ImportAccessState{UserID: uuid.New(), Email: email, Role: role, CanUpload: up, CanDownload: down, Source: "csv"}
}

func users(us ...domain.ImportUserState) map[string]domain.ImportUserState {
	m := map[string]domain.ImportUserState{}
	for _, u := range us {
		m[u.Email] = u
	}
	return m
}

func user(email string, role domain.Role) domain.ImportUserState {
	return domain.ImportUserState{UserID: uuid.New(), Email: email, Role: role}
}

func courseActions(p domain.ImportPlan) []string {
	var out []string
	for _, c := range p.Courses {
		out = append(out, c.Action+":"+c.CourseCode)
	}
	return out
}

func accessActions(p domain.ImportPlan) []string {
	var out []string
	for _, a := range p.Access {
		out = append(out, a.Action+":"+a.CourseCode+":"+a.Email)
	}
	return out
}

var _ = Describe("BuildImportPlan", func() {
	const (
		alice = "alice@chula.ac.th"
		bob   = "bob@chula.ac.th"
		carol = "carol@chula.ac.th"
	)

	It("returns an empty, non-nil plan for no rows and no state", func() {
		p := importplan.BuildImportPlan(nil, domain.ImportState{})
		Expect(p.Courses).NotTo(BeNil())
		Expect(p.Access).NotTo(BeNil())
		Expect(p.Users).NotTo(BeNil())
		Expect(p.Errors).NotTo(BeNil())
		Expect(p.Summary).To(Equal(domain.ImportSummary{}))
		Expect(p.Applied).To(BeFalse())
	})

	Describe("courses", func() {
		It("creates courses missing in the database", func() {
			p := importplan.BuildImportPlan([]domain.ImportRow{row(2, "100", "A", 1, "", "", false, false), row(3, "100", "A", 2, "", "", false, false)}, domain.ImportState{})
			Expect(courseActions(p)).To(Equal([]string{"create:100", "create:100"}))
			Expect(p.Summary.CoursesCreate).To(Equal(2))
			Expect(p.Summary.Rows).To(Equal(2))
			Expect(p.Access).To(BeEmpty())
		})

		It("keeps an existing course and updates it when the name differs", func() {
			st := domain.ImportState{Offerings: []domain.ImportOfferingState{off("100", "A", 1, false), off("200", "Old", 1, false)}}
			p := importplan.BuildImportPlan([]domain.ImportRow{row(2, "100", "A", 1, "", "", false, false), row(3, "200", "New", 1, "", "", false, false)}, st)
			Expect(courseActions(p)).To(Equal([]string{"keep:100", "update:200"}))
			Expect(p.Courses[1].CourseName).To(Equal("New"))
			Expect(p.Summary.CoursesKeep).To(Equal(1))
			Expect(p.Summary.CoursesUpdate).To(Equal(1))
		})

		It("treats the same code in another semester as a different offering", func() {
			st := domain.ImportState{Offerings: []domain.ImportOfferingState{off("100", "A", 1, false)}}
			p := importplan.BuildImportPlan([]domain.ImportRow{row(2, "100", "A", 2, "", "", false, false)}, st)
			Expect(courseActions(p)).To(Equal([]string{"create:100", "delete:100"}))
		})

		It("deletes offerings missing from the file when they never had uploads (D4)", func() {
			st := domain.ImportState{Offerings: []domain.ImportOfferingState{off("100", "A", 1, false, acc(alice, domain.RoleInstructor, true, true))}}
			p := importplan.BuildImportPlan([]domain.ImportRow{row(2, "200", "B", 1, "", "", false, false)}, st)
			Expect(courseActions(p)).To(Equal([]string{"create:200", "delete:100"}))
			Expect(p.Summary.CoursesDelete).To(Equal(1))
			Expect(p.Access).To(BeEmpty(), "access of a deleted offering vanishes with it and is not listed")
		})

		It("keeps offerings with uploads that are not in the file and clears their access (D8, D9)", func() {
			st := domain.ImportState{
				Offerings: []domain.ImportOfferingState{off("100", "A", 1, true, acc(bob, domain.RoleStaff, false, true), acc(alice, domain.RoleInstructor, true, true))},
			}
			p := importplan.BuildImportPlan([]domain.ImportRow{row(2, "200", "B", 1, "", "", false, false)}, st)
			Expect(courseActions(p)).To(Equal([]string{"create:200", "keep_has_uploads:100"}))
			Expect(p.Summary.CoursesKeptWithUploads).To(Equal(1))
			Expect(accessActions(p)).To(Equal([]string{"remove:100:" + alice, "remove:100:" + bob}))
			Expect(p.Access[0].Role).To(Equal(domain.RoleInstructor))
			Expect(p.Access[0].CanUpload).To(BeTrue())
			Expect(p.Summary.AccessRemove).To(Equal(2))
		})

		It("lists out-of-file offerings in a stable (code, semester) order", func() {
			st := domain.ImportState{Offerings: []domain.ImportOfferingState{off("300", "C", 2, false), off("100", "A", 3, true), off("100", "A", 1, false)}}
			p := importplan.BuildImportPlan([]domain.ImportRow{row(2, "999", "Z", 1, "", "", false, false)}, st)
			Expect(courseActions(p)).To(Equal([]string{"create:999", "delete:100", "keep_has_uploads:100", "delete:300"}))
			Expect(p.Courses[1].Semester).To(Equal(1))
			Expect(p.Courses[2].Semester).To(Equal(3))
		})

		It("handles duplicate course rows: one entry, last name wins", func() {
			p := importplan.BuildImportPlan([]domain.ImportRow{row(2, "100", "First", 1, "", "", false, false), row(3, "100", "Second", 1, "", "", false, false)}, domain.ImportState{})
			Expect(p.Courses).To(HaveLen(1))
			Expect(p.Courses[0].CourseName).To(Equal("Second"))
			Expect(p.Summary.Rows).To(Equal(2))
		})
	})

	Describe("access mirror (D4)", func() {
		It("adds access rows for a new offering", func() {
			rows := []domain.ImportRow{row(2, "100", "A", 1, alice, domain.RoleInstructor, true, true), row(3, "100", "A", 1, bob, domain.RoleStaff, false, true)}
			st := domain.ImportState{Users: users(user(alice, domain.RoleInstructor), user(bob, domain.RoleStaff))}
			p := importplan.BuildImportPlan(rows, st)
			Expect(accessActions(p)).To(Equal([]string{"add:100:" + alice, "add:100:" + bob}))
			Expect(p.Access[1].Role).To(Equal(domain.RoleStaff))
			Expect(p.Access[1].CanDownload).To(BeTrue())
			Expect(p.Access[1].CanUpload).To(BeFalse())
			Expect(p.Summary.AccessAdd).To(Equal(2))
			Expect(p.Users).To(BeEmpty())
		})

		It("adds, updates, removes and leaves unchanged rows alone", func() {
			st := domain.ImportState{
				Offerings: []domain.ImportOfferingState{off("100", "A", 1, false,
					acc(alice, domain.RoleInstructor, true, true), // unchanged
					acc(bob, domain.RoleStaff, false, true),       // changed to upload+download
					acc(carol, domain.RoleStaff, true, false),     // not in CSV any more
				)},
				Users: users(user(alice, domain.RoleInstructor), user(bob, domain.RoleStaff), user(carol, domain.RoleStaff), user("dan@chula.ac.th", domain.RoleStaff)),
			}
			rows := []domain.ImportRow{
				row(2, "100", "A", 1, alice, domain.RoleInstructor, true, true),
				row(3, "100", "A", 1, bob, domain.RoleStaff, true, true),
				row(4, "100", "A", 1, "dan@chula.ac.th", domain.RoleStaff, false, true),
			}
			p := importplan.BuildImportPlan(rows, st)
			Expect(accessActions(p)).To(Equal([]string{"update:100:" + bob, "add:100:dan@chula.ac.th", "remove:100:" + carol}))
			Expect(p.Summary.AccessAdd).To(Equal(1))
			Expect(p.Summary.AccessUpdate).To(Equal(1))
			Expect(p.Summary.AccessRemove).To(Equal(1))
		})

		It("ignores both-No rows and removes an existing grant for them", func() {
			st := domain.ImportState{
				Offerings: []domain.ImportOfferingState{off("100", "A", 1, false, acc(alice, domain.RoleInstructor, true, false))},
				Users:     users(user(alice, domain.RoleInstructor), user(bob, domain.RoleStaff)),
			}
			rows := []domain.ImportRow{row(2, "100", "A", 1, alice, domain.RoleInstructor, false, false), row(3, "100", "A", 1, bob, domain.RoleStaff, false, false)}
			p := importplan.BuildImportPlan(rows, st)
			Expect(courseActions(p)).To(Equal([]string{"keep:100"}), "the course stays listed")
			Expect(accessActions(p)).To(Equal([]string{"remove:100:" + alice}))
		})

		It("mirrors only the offerings in the file; other offerings keep their access", func() {
			st := domain.ImportState{Offerings: []domain.ImportOfferingState{
				off("100", "A", 1, false), off("200", "B", 1, false, acc(alice, domain.RoleInstructor, true, true))}}
			rows := []domain.ImportRow{row(2, "100", "A", 1, bob, domain.RoleStaff, false, true), row(3, "200", "B", 1, "", "", false, false)}
			st.Users = users(user(bob, domain.RoleStaff))
			p := importplan.BuildImportPlan(rows, st)
			Expect(accessActions(p)).To(Equal([]string{"add:100:" + bob, "remove:200:" + alice}),
				"a course-only row means no assignments: the mirror empties it")
		})

		It("last duplicate row wins per (course, semester, email) without errors", func() {
			rows := []domain.ImportRow{
				row(2, "100", "A", 1, alice, domain.RoleInstructor, true, true),
				row(3, "100", "A", 1, alice, domain.RoleInstructor, false, true),
			}
			st := domain.ImportState{Users: users(user(alice, domain.RoleInstructor))}
			p := importplan.BuildImportPlan(rows, st)
			Expect(p.Errors).To(BeEmpty())
			Expect(p.Access).To(HaveLen(1))
			Expect(p.Access[0].CanUpload).To(BeFalse())
			Expect(p.Access[0].CanDownload).To(BeTrue())
		})

		It("lets a later both-No duplicate cancel an earlier grant", func() {
			rows := []domain.ImportRow{
				row(2, "100", "A", 1, alice, domain.RoleInstructor, true, true),
				row(3, "100", "A", 1, alice, domain.RoleInstructor, false, false),
			}
			p := importplan.BuildImportPlan(rows, domain.ImportState{})
			Expect(p.Access).To(BeEmpty())
			Expect(p.Users).To(BeEmpty())
		})

		It("keeps the same user on different semesters independent", func() {
			rows := []domain.ImportRow{
				row(2, "100", "A", 1, alice, domain.RoleInstructor, true, true),
				row(3, "100", "A", 2, alice, domain.RoleInstructor, false, true),
			}
			st := domain.ImportState{Users: users(user(alice, domain.RoleInstructor))}
			p := importplan.BuildImportPlan(rows, st)
			Expect(accessActions(p)).To(Equal([]string{"add:100:" + alice, "add:100:" + alice}))
		})
	})

	Describe("users (D10) and role raising", func() {
		It("creates unknown users with the CSV role", func() {
			rows := []domain.ImportRow{row(2, "100", "A", 1, alice, domain.RoleInstructor, true, true), row(3, "100", "A", 1, bob, domain.RoleStaff, false, true)}
			p := importplan.BuildImportPlan(rows, domain.ImportState{})
			Expect(p.Users).To(Equal([]domain.ImportUserPlan{
				{Action: "create", Email: alice, Role: domain.RoleInstructor},
				{Action: "create", Email: bob, Role: domain.RoleStaff},
			}))
			Expect(p.Summary.UsersCreate).To(Equal(2))
			Expect(p.Access[0].Role).To(Equal(domain.RoleInstructor))
		})

		It("creates a user once even when listed on many rows, using the highest role", func() {
			rows := []domain.ImportRow{
				row(2, "100", "A", 1, alice, domain.RoleStaff, false, true),
				row(3, "200", "B", 1, alice, domain.RoleInstructor, true, true),
			}
			p := importplan.BuildImportPlan(rows, domain.ImportState{})
			Expect(p.Users).To(Equal([]domain.ImportUserPlan{{Action: "create", Email: alice, Role: domain.RoleInstructor}}))
			for _, a := range p.Access {
				Expect(a.Role).To(Equal(domain.RoleInstructor))
			}
		})

		It("does not create users that only appear on both-No rows", func() {
			p := importplan.BuildImportPlan([]domain.ImportRow{row(2, "100", "A", 1, alice, domain.RoleStaff, false, false)}, domain.ImportState{})
			Expect(p.Users).To(BeEmpty())
			Expect(p.Summary.UsersCreate).To(BeZero())
		})

		It("raises staff to instructor when the CSV says Instructor", func() {
			st := domain.ImportState{Users: users(user(alice, domain.RoleStaff))}
			p := importplan.BuildImportPlan([]domain.ImportRow{row(2, "100", "A", 1, alice, domain.RoleInstructor, true, true)}, st)
			Expect(p.Users).To(Equal([]domain.ImportUserPlan{{Action: "raise_role", Email: alice, Role: domain.RoleInstructor}}))
			Expect(p.Summary.UsersRoleRaised).To(Equal(1))
			Expect(p.Access[0].Role).To(Equal(domain.RoleInstructor))
		})

		It("never lowers an instructor to staff", func() {
			st := domain.ImportState{Users: users(user(alice, domain.RoleInstructor))}
			p := importplan.BuildImportPlan([]domain.ImportRow{row(2, "100", "A", 1, alice, domain.RoleStaff, false, true)}, st)
			Expect(p.Users).To(BeEmpty())
			Expect(p.Access[0].Role).To(Equal(domain.RoleInstructor))
		})

		It("never touches admins and super admins but still writes their access", func() {
			st := domain.ImportState{Users: users(user(alice, domain.RoleAdmin), user(bob, domain.RoleSuperAdmin))}
			rows := []domain.ImportRow{row(2, "100", "A", 1, alice, domain.RoleInstructor, true, true), row(3, "100", "A", 1, bob, domain.RoleStaff, false, true)}
			p := importplan.BuildImportPlan(rows, st)
			Expect(p.Users).To(BeEmpty())
			Expect(accessActions(p)).To(Equal([]string{"add:100:" + alice, "add:100:" + bob}))
			Expect(p.Access[0].Role).To(Equal(domain.RoleAdmin))
			Expect(p.Access[1].Role).To(Equal(domain.RoleSuperAdmin))
		})

		It("keeps an existing staff as staff when the CSV says Staff", func() {
			st := domain.ImportState{Users: users(user(alice, domain.RoleStaff))}
			p := importplan.BuildImportPlan([]domain.ImportRow{row(2, "100", "A", 1, alice, domain.RoleStaff, false, true)}, st)
			Expect(p.Users).To(BeEmpty())
		})
	})

	It("is deterministic: the same input yields the same plan", func() {
		st := domain.ImportState{Offerings: []domain.ImportOfferingState{off("100", "A", 1, false, acc(alice, domain.RoleInstructor, true, true), acc(bob, domain.RoleStaff, false, true))}}
		rows := []domain.ImportRow{row(2, "200", "B", 1, carol, domain.RoleInstructor, true, true)}
		Expect(importplan.BuildImportPlan(rows, st)).To(Equal(importplan.BuildImportPlan(rows, st)))
	})

	It("does not mutate its inputs", func() {
		st := domain.ImportState{Offerings: []domain.ImportOfferingState{off("100", "A", 1, true, acc(bob, domain.RoleStaff, false, true), acc(alice, domain.RoleInstructor, true, true))}}
		before := st.Offerings[0].Access[0].Email
		importplan.BuildImportPlan(nil, st)
		Expect(st.Offerings[0].Access[0].Email).To(Equal(before))
	})
})
