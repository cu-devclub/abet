package importplan_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"abet/main-service/internal/application/importplan"
	"abet/main-service/internal/core/domain"
)

var allowed = []string{"chula.ac.th"}

const header = "course id,course name,semester,user email,user role,upload permission,download permission\n"

func errorsOf(res importplan.ParseResult) []string {
	var out []string
	for _, e := range res.Errors {
		out = append(out, e.Message)
	}
	return out
}

var _ = Describe("ParseCSV", func() {
	It("parses a valid file with all columns", func() {
		res := importplan.ParseCSV([]byte(header+
			"2301173,Calculus,1,Alice@Chula.ac.th,Instructor,Yes,No\n"+
			"2301173,Calculus,1,bob@chula.ac.th,staff,n,y\n"), allowed)
		Expect(res.Errors).To(BeEmpty())
		Expect(res.RowCount).To(Equal(2))
		Expect(res.Rows).To(Equal([]domain.ImportRow{
			{Row: 2, CourseCode: "2301173", CourseName: "Calculus", Semester: 1, Email: "alice@chula.ac.th", Role: domain.RoleInstructor, CanUpload: true},
			{Row: 3, CourseCode: "2301173", CourseName: "Calculus", Semester: 1, Email: "bob@chula.ac.th", Role: domain.RoleStaff, CanDownload: true},
		}))
	})

	It("strips a UTF-8 BOM", func() {
		res := importplan.ParseCSV([]byte("\xef\xbb\xbf"+header+"1,A,1,,,,\n"), allowed)
		Expect(res.Errors).To(BeEmpty())
		Expect(res.Rows).To(HaveLen(1))
	})

	It("accepts header aliases (case, spaces, underscores, hyphens)", func() {
		res := importplan.ParseCSV([]byte("Course_ID,COURSE-NAME,Semester,User Email,user_role,Upload-Permission,DOWNLOAD PERMISSION\n"+
			"1,A,2,a@chula.ac.th,Staff,1,0\n"), allowed)
		Expect(res.Errors).To(BeEmpty())
		Expect(res.Rows).To(HaveLen(1))
		Expect(res.Rows[0].CanUpload).To(BeTrue())
		Expect(res.Rows[0].CanDownload).To(BeFalse())
	})

	It("allows a file without user columns (course-only)", func() {
		res := importplan.ParseCSV([]byte("course id,course name,semester\n1,A,3\n"), allowed)
		Expect(res.Errors).To(BeEmpty())
		Expect(res.Rows).To(HaveLen(1))
		Expect(res.Rows[0].Email).To(BeEmpty())
	})

	It("treats a row with an empty email and empty user columns as course-only", func() {
		res := importplan.ParseCSV([]byte(header+"1,A,1,,,,\n"), allowed)
		Expect(res.Errors).To(BeEmpty())
		Expect(res.Rows[0].Email).To(BeEmpty())
	})

	It("skips blank lines and blank rows but keeps real line numbers", func() {
		res := importplan.ParseCSV([]byte(header+"\n,,,,,,\n1,A,x,,,,\n"), allowed)
		Expect(res.RowCount).To(Equal(1))
		Expect(res.Errors).To(Equal([]domain.ImportError{{Row: 4, Message: "semester must be 1, 2 or 3"}}))
	})

	DescribeTable("row errors carry the right row number (header = row 1)",
		func(line string, row int, msg string) {
			res := importplan.ParseCSV([]byte(header+"9,Ok,1,ok@chula.ac.th,Staff,Yes,No\n"+line+"\n"), allowed)
			Expect(res.Errors).To(Equal([]domain.ImportError{{Row: row, Message: msg}}))
			Expect(res.Rows).To(HaveLen(1), "the valid row is still returned")
		},
		Entry("bad semester", "1,A,4,,,,", 3, "semester must be 1, 2 or 3"),
		Entry("empty semester", "1,A,,,,,", 3, "semester must be 1, 2 or 3"),
		Entry("missing course id", ",A,1,,,,", 3, "course id is required"),
		Entry("missing course name", "1,,1,,,,", 3, "course name is required"),
		Entry("bad role", "1,A,1,x@chula.ac.th,Admin,Yes,No", 3, "user role must be Instructor or Staff"),
		Entry("missing role", "1,A,1,x@chula.ac.th,,Yes,No", 3, "user role must be Instructor or Staff"),
		Entry("bad upload bool", "1,A,1,x@chula.ac.th,Staff,maybe,No", 3, "upload permission must be Yes or No"),
		Entry("bad download bool", "1,A,1,x@chula.ac.th,Staff,Yes,2", 3, "download permission must be Yes or No"),
		Entry("missing permission", "1,A,1,x@chula.ac.th,Staff,Yes,", 3, "download permission must be Yes or No"),
		Entry("email domain", "1,A,1,x@gmail.com,Staff,Yes,No", 3, "email domain is not allowed"),
		Entry("email syntax", "1,A,1,not-an-email,Staff,Yes,No", 3, "invalid email address"),
		Entry("role without email", "1,A,1,,Staff,,", 3, "user email is required when user role or permissions are given"),
	)

	It("reports every problem of a row", func() {
		res := importplan.ParseCSV([]byte(header+"1,A,9,x@gmail.com,Boss,Yes,No\n"), allowed)
		Expect(errorsOf(res)).To(ConsistOf(
			"semester must be 1, 2 or 3", "email domain is not allowed", "user role must be Instructor or Staff"))
		for _, e := range res.Errors {
			Expect(e.Row).To(Equal(2))
		}
	})

	It("reports missing required columns on row 1", func() {
		res := importplan.ParseCSV([]byte("course id,semester\n1,1\n"), allowed)
		Expect(res.Errors).To(Equal([]domain.ImportError{{Row: 1, Message: "missing column: course name"}}))
	})

	It("rejects an empty file and a header-only file", func() {
		Expect(importplan.ParseCSV(nil, allowed).Errors).To(HaveLen(1))
		res := importplan.ParseCSV([]byte(header), allowed)
		Expect(errorsOf(res)).To(Equal([]string{"the file has no data rows"}))
	})

	It("handles quoted fields with commas", func() {
		res := importplan.ParseCSV([]byte(header+`1,"Calculus, Part I",1,,,,`+"\n"), allowed)
		Expect(res.Errors).To(BeEmpty())
		Expect(res.Rows[0].CourseName).To(Equal("Calculus, Part I"))
	})

	It("handles CRLF line endings", func() {
		res := importplan.ParseCSV([]byte("course id,course name,semester\r\n1,A,1\r\n2,B,2\r\n"), allowed)
		Expect(res.Errors).To(BeEmpty())
		Expect(res.Rows).To(HaveLen(2))
	})
})
