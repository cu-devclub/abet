package templates_test

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"abet/main-service/internal/adapter/outbound/templates"
)

var _ = Describe("templates service", func() {
	It("zips the package below course-id-course-name/ with the same files and contents", func() {
		dir := GinkgoT().TempDir()
		Expect(os.MkdirAll(filepath.Join(dir, "00-meta"), 0o755)).To(Succeed())
		Expect(os.MkdirAll(filepath.Join(dir, "10-lectures"), 0o755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "README_FIRST.txt"), []byte("read me"), 0o644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "00-meta", "a.csv"), []byte("x,y\n1,2\n"), 0o644)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(dir, "10-lectures", "week01-part01.pdf"), []byte("%PDF-1.4"), 0o644)).To(Succeed())

		var buf bytes.Buffer
		Expect(templates.New(dir).WriteInstructorPackage(&buf)).To(Succeed())
		zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		Expect(err).NotTo(HaveOccurred())
		got := map[string]string{}
		for _, f := range zr.File {
			rc, _ := f.Open()
			b, _ := io.ReadAll(rc)
			rc.Close()
			got[f.Name] = string(b)
		}
		Expect(got).To(HaveKeyWithValue("course-id-course-name/README_FIRST.txt", "read me"))
		Expect(got).To(HaveKeyWithValue("course-id-course-name/00-meta/a.csv", "x,y\n1,2\n"))
		Expect(got).To(HaveKeyWithValue("course-id-course-name/10-lectures/week01-part01.pdf", "%PDF-1.4"))
		for name := range got {
			Expect(name).To(HavePrefix("course-id-course-name/"))
		}
	})

	It("zips the real instructor package of portfolio-service completely", func() {
		real := filepath.Join("..", "..", "..", "..", "..", "portfolio-service", "templates", "instructor-package")
		if _, err := os.Stat(real); err != nil {
			Skip("portfolio-service template not present")
		}
		var buf bytes.Buffer
		Expect(templates.New(real).WriteInstructorPackage(&buf)).To(Succeed())
		zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		Expect(err).NotTo(HaveOccurred())
		files := 0
		for _, f := range zr.File {
			if !strings.HasSuffix(f.Name, "/") {
				files++
			}
		}
		want := 0
		_ = filepath.WalkDir(real, func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				want++
			}
			return nil
		})
		Expect(files).To(Equal(want))
		Expect(files).To(BeNumerically(">", 50))
	})

	It("fails clearly when the directory is missing", func() {
		var buf bytes.Buffer
		Expect(templates.New(filepath.Join(GinkgoT().TempDir(), "nope")).WriteInstructorPackage(&buf)).NotTo(Succeed())
	})

	It("serves a mapping CSV with the documented header and 2 example rows", func() {
		recs, err := csv.NewReader(bytes.NewReader(templates.New("").CourseMappingCSV())).ReadAll()
		Expect(err).NotTo(HaveOccurred())
		Expect(recs).To(HaveLen(3))
		Expect(recs[0]).To(Equal([]string{"course id", "course name", "semester", "user email", "user role", "upload permission", "download permission"}))
		for _, r := range recs[1:] {
			Expect(r).To(HaveLen(7))
		}
	})
})
