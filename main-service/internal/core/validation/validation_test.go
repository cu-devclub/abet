package validation_test

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/validation"
)

const courseCode = "1234567"

// pkg is an in-memory zip tree: slash path -> content. Paths ending in "/" are explicit directory entries.
type pkg map[string][]byte

// loadFixture reads the valid instructor package (testdata/package, a filled copy of portfolio-service's template).
func loadFixture() pkg {
	out := pkg{}
	root := "testdata/package"
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		out[filepath.ToSlash(rel)] = b
		return nil
	})
	Expect(err).NotTo(HaveOccurred())
	return out
}

func (p pkg) clone() pkg {
	out := make(pkg, len(p))
	for k, v := range p {
		out[k] = v
	}
	return out
}

func (p pkg) with(path, content string) pkg { p[path] = []byte(content); return p }
func (p pkg) without(paths ...string) pkg {
	for _, x := range paths {
		delete(p, x)
	}
	return p
}

// prefixed moves every entry below prefix (e.g. "course/").
func (p pkg) prefixed(prefix string) pkg {
	out := pkg{}
	for k, v := range p {
		out[prefix+k] = v
	}
	return out
}

func (p pkg) zipBytes() []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, data := range p {
		w, err := zw.Create(name)
		Expect(err).NotTo(HaveOccurred())
		_, err = w.Write(data)
		Expect(err).NotTo(HaveOccurred())
	}
	Expect(zw.Close()).To(Succeed())
	return buf.Bytes()
}

func run(p pkg, code string) []domain.Issue {
	b := p.zipBytes()
	return validation.Validate(bytes.NewReader(b), int64(len(b)), code)
}

func errorsOf(issues []domain.Issue) []domain.Issue {
	var out []domain.Issue
	for _, i := range issues {
		if i.Severity == domain.SeverityError {
			out = append(out, i)
		}
	}
	return out
}

func codes(issues []domain.Issue) []string {
	var out []string
	for _, i := range issues {
		out = append(out, i.Code)
	}
	return out
}

func titles(issues []domain.Issue) []string {
	var out []string
	for _, i := range issues {
		out = append(out, i.Title)
	}
	return out
}

const validSyllabusHeader = "course_code,course_name,required_or_elective,catalog_description,prerequisites,categorization_of_credits,instructors,textbooks_and_other_required_material,course_learning_outcomes_expected_performance_criteria,topics_covered,class_laboratory_schedule,ice_outcome_1,ice_outcome_2,ice_outcome_3,ice_outcome_4,ice_outcome_5,ice_outcome_6,ice_outcome_7,prepared_by,prepared_date"

const (
	identity = "00-meta/instructor_course_identity_template.csv"
	syllabus = "01-syllabus/instructor_syllabus_abet_reference.csv"
	schedule = "03-schedule/instructor_course_schedule_template.csv"
	grading  = "90-grades/instructor_course_grading_template.csv"
)

var _ = Describe("Validate", func() {
	var fx pkg
	BeforeEach(func() { fx = loadFixture() })

	Describe("the valid fixture", func() {
		It("produces no issues at all", func() {
			Expect(run(fx, courseCode)).To(BeEmpty())
		})
		It("accepts it without an expected course code", func() {
			Expect(run(fx, "")).To(BeEmpty())
		})
		It("accepts the course code case-insensitively", func() {
			Expect(run(fx, " "+strings.ToLower(courseCode)+" ")).To(BeEmpty())
		})
	})

	Describe("wrapper folders and junk", func() {
		It("accepts one wrapper folder", func() {
			Expect(run(fx.prefixed("course-1234567/"), courseCode)).To(BeEmpty())
		})
		It("collapses up to 5 single-folder levels", func() {
			Expect(run(fx.prefixed("a/b/c/d/e/"), courseCode)).To(BeEmpty())
		})
		It("does not collapse deeper than that", func() {
			issues := run(fx.prefixed("a/b/c/d/e/f/"), courseCode)
			Expect(codes(issues)).To(ContainElement("MISSING_FOLDER"))
		})
		It("accepts the 9 folders directly at the zip root", func() {
			Expect(run(fx, courseCode)).To(BeEmpty()) // fx is already the root layout
		})
		It("ignores macOS junk everywhere", func() {
			p := fx.prefixed("course/")
			p["__MACOSX/course/._README_FIRST.txt"] = []byte("\x00\x05junk")
			p["__macosx/x"] = []byte{0}
			p["course/.DS_Store"] = []byte{0, 1}
			p["course/10-lectures/._week01-part01.pdf"] = []byte("not a pdf")
			p["course/10-lectures/.DS_Store"] = []byte{0}
			Expect(run(p, courseCode)).To(BeEmpty())
		})
		It("ignores root files next to a single course folder, like the worker", func() {
			p := fx.prefixed("course/")
			p["notes.docx"] = []byte("x")
			Expect(run(p, courseCode)).To(BeEmpty())
		})
		It("reports several unrelated top-level folders", func() {
			p := pkg{"a/x.txt": []byte("x"), "b/y.txt": []byte("y")}
			issues := run(p, courseCode)
			Expect(issues).To(HaveLen(1))
			Expect(issues[0].Code).To(Equal("ZIP_ROOT"))
			Expect(issues[0].Title).To(Equal("Submission zip must contain exactly one top-level folder. Found 2."))
			Expect(issues[0].Stage).To(Equal(domain.StageStructure))
		})
		It("reports an empty zip as Found 0", func() {
			issues := run(pkg{}, courseCode)
			Expect(titles(issues)).To(ConsistOf("Submission zip must contain exactly one top-level folder. Found 0."))
		})
		It("treats root files only as a (broken) course", func() {
			issues := run(pkg{"README_FIRST.txt": []byte("hi")}, courseCode)
			Expect(codes(issues)).To(ContainElement("MISSING_FOLDER"))
		})
		It("does not treat Thumbs.db as junk (the worker rejects it too)", func() {
			p := fx.clone()
			p["Thumbs.db"] = []byte{1}
			Expect(codes(run(p, courseCode))).To(ConsistOf("UNEXPECTED_TOP_LEVEL_FILE"))
		})
	})

	Describe("missing", func() {
		It("MISSING_FOLDER for each absent required folder", func() {
			p := fx.clone()
			for k := range p {
				if strings.HasPrefix(k, "02-textbook/") {
					delete(p, k)
				}
			}
			issues := run(p, courseCode)
			Expect(titles(errorsOf(issues))).To(ConsistOf("Missing required folder: 02-textbook"))
			Expect(issues[0].Stage).To(Equal(domain.StageMissing))
			Expect(issues[0].Description).To(ContainSubstring("9 required course folders"))
			Expect(issues[0].SuggestedFix).To(Equal("Ensure all 9 required subdirectories exist and that all 4 mandatory CSV templates are included."))
		})
		It("MISSING_FILE for each absent mandatory CSV", func() {
			for _, rel := range []string{identity, syllabus, schedule, grading} {
				issues := run(fx.clone().without(rel), courseCode)
				Expect(titles(errorsOf(issues))).To(ConsistOf("Missing required file: " + rel))
				Expect(issues[0].Code).To(Equal("MISSING_FILE"))
			}
		})
		It("reports both folder and file when a whole folder is gone", func() {
			p := fx.clone()
			for k := range p {
				if strings.HasPrefix(k, "90-grades/") {
					delete(p, k)
				}
			}
			Expect(codes(errorsOf(run(p, courseCode)))).To(ConsistOf("MISSING_FOLDER", "MISSING_FILE"))
		})
	})

	Describe("structure", func() {
		It("UNEXPECTED_TOP_LEVEL_FOLDER (and its files are outside the required folders)", func() {
			// next to the 9 folders at the zip root an unknown folder makes the zip ambiguous (worker: "Found N")
			Expect(codes(run(fx.clone().with("extra/readme.txt", "x"), courseCode))).To(ConsistOf("ZIP_ROOT"))
			p := fx.prefixed("course/").with("course/extra/readme.txt", "x")
			Expect(titles(run(p, courseCode))).To(ConsistOf( // the implied directory is reported like the worker's rglob does
				"Unexpected top-level folder: extra",
				"Unexpected file outside required folders: extra/readme.txt"))
			p["course/extra/"] = nil // an explicit directory entry changes nothing
			Expect(titles(run(p, courseCode))).To(ConsistOf(
				"Unexpected top-level folder: extra",
				"Unexpected file outside required folders: extra/readme.txt"))
		})
		It("UNEXPECTED_NESTED_FOLDER and UNEXPECTED_NESTED_FILE", func() {
			p := fx.clone().with("10-lectures/sub/week01-part09.pdf", "%PDF-1.4")
			Expect(titles(run(p, courseCode))).To(ConsistOf(
				"Unexpected nested file path: 10-lectures/sub/week01-part09.pdf",
				"Unexpected nested folder: 10-lectures/sub"))
		})
		It("UNEXPECTED_TOP_LEVEL_FILE", func() {
			issues := run(fx.clone().with("notes.txt", "x"), courseCode)
			Expect(titles(issues)).To(ConsistOf("Unexpected top-level file: notes.txt"))
			Expect(issues[0].Stage).To(Equal(domain.StageStructure))
			Expect(issues[0].SuggestedFix).To(ContainSubstring("maximum depth of two levels"))
		})
		It("allows the three documented root files", func() {
			Expect(fx).To(HaveKey("README_FIRST.txt"))
			Expect(fx).To(HaveKey("INSTRUCTOR_DETAILED_SUBMISSION_GUIDE.md"))
			Expect(fx).To(HaveKey("INSTRUCTOR_DETAILED_SUBMISSION_GUIDE.pdf"))
			Expect(run(fx.clone(), courseCode)).To(BeEmpty())
		})
		It("reports relative paths without the wrapper folder", func() {
			p := fx.clone().with("20-assignments/old/asg09-high.pdf", "%PDF-1.4")
			Expect(titles(run(p.prefixed("course/"), courseCode))).To(ContainElement(
				"Unexpected nested file path: 20-assignments/old/asg09-high.pdf"))
		})
	})

	Describe("naming and content", func() {
		It("UNEXPECTED_FILE_NAME", func() {
			issues := run(fx.clone().with("10-lectures/Week1.pdf", "%PDF-1.4"), courseCode)
			Expect(titles(issues)).To(ConsistOf("Unexpected file name in 10-lectures: 10-lectures/Week1.pdf"))
			Expect(issues[0].Stage).To(Equal(domain.StageNaming))
		})
		It("rejects names the strict patterns do not allow", func() {
			for _, rel := range []string{
				"10-lectures/week1-part01.pdf", "20-assignments/asg1-high.pdf", "30-quizzes/quiz01-median.pdf",
				"40-exams/exam-high.pdf", "02-textbook/textbook-part1.pdf", "00-meta/other.csv", "90-grades/grades.pdf",
			} {
				Expect(codes(run(fx.clone().with(rel, "%PDF-1.4"), courseCode))).To(ContainElement("UNEXPECTED_FILE_NAME"), rel)
			}
		})
		It("accepts the special names", func() {
			p := fx.clone().with("20-assignments/asg-all-scores.pdf", "%PDF-1.4").
				with("03-schedule/lecture_schedule_mapping.csv", "a,b\n1,2\n").
				with("40-exams/midterm-scores.pdf", "%PDF-1.4")
			Expect(run(p, courseCode)).To(BeEmpty())
		})
		It("INVALID_PDF when the header is missing", func() {
			issues := run(fx.clone().with("10-lectures/week01-part01.pdf", "hello"), courseCode)
			Expect(titles(issues)).To(ConsistOf("Invalid PDF content for file with .pdf extension: 10-lectures/week01-part01.pdf"))
			Expect(issues[0].Code).To(Equal("INVALID_PDF"))
		})
		It("INVALID_PDF for an empty pdf, case-insensitive extension", func() {
			Expect(codes(run(fx.clone().with("10-lectures/week01-part01.pdf", ""), courseCode))).To(ConsistOf("INVALID_PDF"))
			Expect(codes(run(fx.clone().with("10-lectures/week01-part03.PDF", "x"), courseCode))).To(
				ConsistOf("UNEXPECTED_FILE_NAME", "INVALID_PDF"))
		})
		It("accepts %PDF- anywhere in the first 1024 bytes like the worker", func() {
			body := strings.Repeat("x", 1000) + "%PDF-1.4"
			Expect(run(fx.clone().with("10-lectures/week01-part01.pdf", body), courseCode)).To(BeEmpty())
			body = strings.Repeat("x", 1020) + "%PDF-1.4"
			Expect(codes(run(fx.clone().with("10-lectures/week01-part01.pdf", body), courseCode))).To(ConsistOf("INVALID_PDF"))
		})
		It("INVALID_TEXT for NUL bytes and non UTF-8", func() {
			good := fx[grading]
			p := fx.clone()
			p[grading] = append(append([]byte{}, good...), 0)
			issues := run(p, courseCode)
			Expect(titles(issues)).To(ConsistOf("Invalid text content for file with .csv extension: " + grading))
			Expect(issues[0].Code).To(Equal("INVALID_TEXT"))

			p[grading] = append(append([]byte{}, good...), 0xE9, '\n') // latin-1 é
			Expect(codes(run(p, courseCode))).To(ConsistOf("INVALID_TEXT"))

			p[grading] = []byte{0xE0, 0xB8} // truncated multi-byte rune at EOF
			Expect(codes(run(p, courseCode))).To(ContainElement("INVALID_TEXT"))
		})
		It("checks .tex .txt .md as text too", func() {
			Expect(codes(run(fx.clone().with("README_FIRST.txt", "a\x00b"), courseCode))).To(ConsistOf("INVALID_TEXT"))
			Expect(titles(run(fx.clone().with("01-syllabus/syllabus-abet.tex", "\xff"), courseCode))).To(ConsistOf(
				"Invalid text content for file with .tex extension: 01-syllabus/syllabus-abet.tex"))
		})
		It("accepts a UTF-8 BOM and Thai text", func() {
			p := fx.clone()
			p[grading] = []byte("\xEF\xBB\xBFcomponent,weight_percent\nการบ้าน,40\nสอบปลายภาค,60\n")
			Expect(run(p, courseCode)).To(BeEmpty())
		})
		It("does not split multi-byte runes across read chunks", func() {
			p := fx.clone()
			p["README_FIRST.txt"] = []byte(strings.Repeat("ก", 100000))
			Expect(run(p, courseCode)).To(BeEmpty())
		})
		It("only reports the name for unrecognised extensions", func() {
			issues := run(fx.clone().with("10-lectures/week01-part01.pptx", "x"), courseCode)
			Expect(codes(issues)).To(ConsistOf("UNEXPECTED_FILE_NAME"))
		})
	})

	Describe("limits", func() {
		It("refuses more than 2000 entries", func() {
			p := pkg{}
			for i := 0; i <= validation.MaxEntries; i++ {
				p[fmt.Sprintf("c/f%04d.txt", i)] = []byte("x")
			}
			issues := run(p, courseCode)
			Expect(issues).To(HaveLen(1))
			Expect(issues[0].Code).To(Equal("ZIP_TOO_LARGE"))
			Expect(issues[0].Stage).To(Equal(domain.StageStructure))
		})
		It("does not refuse exactly 2000 entries", func() {
			p := fx.clone()
			for i := len(p); i < validation.MaxEntries; i++ {
				p[fmt.Sprintf("10-lectures/week%02d-part%02d.pdf", i%90+10, i/90)] = []byte("%PDF-1.4")
			}
			Expect(len(p)).To(Equal(validation.MaxEntries))
			Expect(codes(run(p, courseCode))).NotTo(ContainElement("ZIP_TOO_LARGE"))
		})
		It("refuses archives declaring more than 1 GB uncompressed", func() {
			var buf bytes.Buffer
			zw := zip.NewWriter(&buf)
			_, err := zw.CreateRaw(&zip.FileHeader{Name: "c/big.bin", Method: zip.Store, UncompressedSize64: 2 << 30, CompressedSize64: 0})
			Expect(err).NotTo(HaveOccurred())
			Expect(zw.Close()).To(Succeed())
			issues := validation.Validate(bytes.NewReader(buf.Bytes()), int64(buf.Len()), courseCode)
			Expect(issues).To(HaveLen(1))
			Expect(issues[0].Code).To(Equal("ZIP_TOO_LARGE"))
		})
		It("reports a non-zip as ZIP_INVALID", func() {
			b := []byte("definitely not a zip")
			issues := validation.Validate(bytes.NewReader(b), int64(len(b)), courseCode)
			Expect(codes(issues)).To(ConsistOf("ZIP_INVALID"))
		})
	})

	Describe("identity CSV", func() {
		const hdr = "course_name,course_code,course_instructors,course_instructors_short,program_name,school_name,university_name,semester_label\n"
		It("IDENTITY_ROW_COUNT for zero populated rows", func() {
			issues := run(fx.clone().with(identity, hdr), courseCode)
			Expect(titles(issues)).To(ConsistOf("Identity CSV must contain exactly one populated row"))
			Expect(issues[0].Stage).To(Equal(domain.StageCSV))
		})
		It("IDENTITY_ROW_COUNT for two rows; blank rows do not count", func() {
			row := "A,1234567,I,I,P,S,U,L\n"
			Expect(codes(run(fx.clone().with(identity, hdr+row+row), courseCode))).To(ConsistOf("IDENTITY_ROW_COUNT"))
			Expect(run(fx.clone().with(identity, hdr+"\n,,,,,,,\n"+row+"\n"), courseCode)).To(BeEmpty())
		})
		It("IDENTITY_MISSING_COLUMNS", func() {
			issues := run(fx.clone().with(identity, "course_name,course_code\nA,1234567\n"), courseCode)
			Expect(codes(issues)).To(ConsistOf("IDENTITY_MISSING_COLUMNS"))
			Expect(issues[0].Title).To(HavePrefix("Missing required columns in identity CSV: course_instructors"))
		})
		It("no header at all", func() {
			Expect(codes(run(fx.clone().with(identity, ""), courseCode))).To(ConsistOf("IDENTITY_NO_HEADER"))
		})
		It("ROW_MISSING_VALUES lists the blank columns with the worker's row number", func() {
			issues := run(fx.clone().with(identity, hdr+"A,1234567,,I,P,,U,L\n"), courseCode)
			Expect(titles(issues)).To(ConsistOf("Row 2 missing required values: course_instructors, school_name"))
			Expect(issues[0].Code).To(Equal("ROW_MISSING_VALUES"))
		})
		It("an empty course code is reported as a missing value", func() {
			issues := run(fx.clone().with(identity, hdr+"A,,I,I,P,S,U,L\n"), courseCode)
			Expect(titles(issues)).To(ConsistOf("Row 2 missing required values: course_code"))
		})
		It("IDENTITY_COURSE_MISMATCH", func() {
			issues := run(fx, "2301173")
			Expect(titles(issues)).To(ConsistOf("Identity CSV course code 1234567 does not match course 2301173"))
			Expect(issues[0].Code).To(Equal("IDENTITY_COURSE_MISMATCH"))
		})
		It("tolerates BOM, CRLF and quoted commas", func() {
			row := "\xEF\xBB\xBF" + strings.ReplaceAll(hdr, "\n", "\r\n") + "\"Name, with comma\",1234567,\"Dr. A, Ph.D.\",A,P,S,U,L\r\n"
			Expect(run(fx.clone().with(identity, row), courseCode)).To(BeEmpty())
		})
	})

	Describe("syllabus CSV", func() {
		It("SYLLABUS_NO_HEADER", func() {
			Expect(codes(run(fx.clone().with(syllabus, "\n\n"), courseCode))).To(ConsistOf("SYLLABUS_NO_HEADER"))
		})
		It("SYLLABUS_MISSING_COLUMNS", func() {
			issues := run(fx.clone().with(syllabus, "course_code,course_name\n1,2\n"), courseCode)
			Expect(codes(issues)).To(ConsistOf("SYLLABUS_MISSING_COLUMNS"))
		})
		It("SYLLABUS_TEMPLATE_ONLY for header-only and for placeholder-only files", func() {
			Expect(codes(run(fx.clone().with(syllabus, validSyllabusHeader+"\n"), courseCode))).To(ConsistOf("SYLLABUS_TEMPLATE_ONLY"))
			placeholder := validSyllabusHeader + "\nyour_course_code,your_course_name,required_elective,your_catalog_description,,,,,,,,,,,,,,,,\n"
			issues := run(fx.clone().with(syllabus, placeholder), courseCode)
			Expect(titles(issues)).To(ConsistOf("No non-template syllabus rows found in CSV"))
		})
		It("accepts a real row after the placeholder row", func() {
			ph := validSyllabusHeader + "\nyour_course_code,your_course_name,x,y,,,,,,,,,,,,,,,,\n1234567,Real,Required,desc,,,,,,,,,,,,,,,,\n"
			Expect(run(fx.clone().with(syllabus, ph), courseCode)).To(BeEmpty())
		})
	})

	Describe("schedule CSV", func() {
		It("SCHEDULE_NO_HEADER and SCHEDULE_MISSING_COLUMNS", func() {
			Expect(codes(run(fx.clone().with(schedule, ""), courseCode))).To(ConsistOf("SCHEDULE_NO_HEADER"))
			Expect(codes(run(fx.clone().with(schedule, "week\n1\n"), courseCode))).To(ConsistOf("SCHEDULE_MISSING_COLUMNS"))
		})
		It("SCHEDULE_WEEK_REQUIRED", func() {
			issues := run(fx.clone().with(schedule, "week,topic\n1,A\n,B\n"), courseCode)
			Expect(titles(issues)).To(ConsistOf("Row 3: week is required when topic is provided"))
			Expect(issues[0].Code).To(Equal("SCHEDULE_WEEK_REQUIRED"))
		})
		It("SCHEDULE_WEEK_INVALID", func() {
			issues := run(fx.clone().with(schedule, "week,topic\nWeek 1,A\n1.5,B\n"), courseCode)
			Expect(titles(issues)).To(ConsistOf(
				"Row 2: invalid week 'Week 1'. Use integers", "Row 3: invalid week '1.5'. Use integers"))
		})
		It("SCHEDULE_WEEK_NONPOSITIVE", func() {
			issues := run(fx.clone().with(schedule, "week,topic\n0,A\n-2,B\n"), courseCode)
			Expect(titles(issues)).To(ConsistOf("Row 2: week must be > 0", "Row 3: week must be > 0"))
			Expect(issues[0].Code).To(Equal("SCHEDULE_WEEK_NONPOSITIVE"))
		})
		It("SCHEDULE_EMPTY when no topic is filled in (the unfilled template)", func() {
			issues := run(fx.clone().with(schedule, "week,topic\n1,\n2,\n3,\n"), courseCode)
			Expect(titles(issues)).To(ConsistOf("No non-empty week/topic rows found in schedule CSV"))
			Expect(issues[0].Code).To(Equal("SCHEDULE_EMPTY"))
		})
		It("lets a week without topic pass when another row has a topic", func() {
			Expect(run(fx.clone().with(schedule, "week,topic\n1,\n2,Intro\n"), courseCode)).To(BeEmpty())
		})
	})

	Describe("grading CSV", func() {
		It("GRADING_NO_HEADER and GRADING_MISSING_COLUMNS", func() {
			Expect(codes(run(fx.clone().with(grading, ""), courseCode))).To(ConsistOf("GRADING_NO_HEADER"))
			Expect(codes(run(fx.clone().with(grading, "component\nLabs\n"), courseCode))).To(ConsistOf("GRADING_MISSING_COLUMNS"))
		})
		It("GRADING_BOTH_REQUIRED", func() {
			issues := run(fx.clone().with(grading, "component,weight_percent\nLabs,\n,10\n"), courseCode)
			Expect(titles(issues)).To(ConsistOf(
				"Row 2: both component and weight_percent are required", "Row 3: both component and weight_percent are required"))
		})
		It("GRADING_WEIGHT_NUMERIC", func() {
			issues := run(fx.clone().with(grading, "component,weight_percent\nLabs,15%\n"), courseCode)
			Expect(titles(issues)).To(ConsistOf("Row 2: weight_percent must be numeric"))
			Expect(issues[0].Code).To(Equal("GRADING_WEIGHT_NUMERIC"))
		})
		It("GRADING_WEIGHT_NEGATIVE", func() {
			issues := run(fx.clone().with(grading, "component,weight_percent\nLabs,-1\n"), courseCode)
			Expect(titles(issues)).To(ConsistOf("Row 2: weight_percent must be >= 0"))
			Expect(issues[0].Code).To(Equal("GRADING_WEIGHT_NEGATIVE"))
		})
		It("GRADING_EMPTY", func() {
			issues := run(fx.clone().with(grading, "component,weight_percent\n"), courseCode)
			Expect(titles(issues)).To(ConsistOf("No grading rows found in CSV"))
			Expect(issues[0].Code).To(Equal("GRADING_EMPTY"))
		})
		It("accepts decimals and zero", func() {
			Expect(run(fx.clone().with(grading, "component,weight_percent\nLabs,12.5\nBonus,0\n"), courseCode)).To(BeEmpty())
		})
	})

	Describe("incomplete documents (warnings only)", func() {
		It("warns about a missing part of an assignment, and still has no errors", func() {
			issues := run(fx.clone().without("20-assignments/asg02-low.pdf"), courseCode)
			Expect(issues).To(HaveLen(1))
			Expect(issues[0].Severity).To(Equal(domain.SeverityWarning))
			Expect(issues[0].Stage).To(Equal(domain.StageMissing))
			Expect(issues[0].Code).To(Equal("INCOMPLETE_DOCUMENT"))
			Expect(issues[0].Title).To(Equal("Incomplete document: asg02 is missing asg02-low.pdf"))
			Expect(issues[0].Description).To(ContainSubstring("incomplete"))
		})
		It("warns for quizzes, midterm and final", func() {
			p := fx.clone().without("30-quizzes/quiz03-high.pdf", "40-exams/midterm-average.pdf", "40-exams/final-instructions.pdf")
			Expect(titles(run(p, courseCode))).To(ConsistOf(
				"Incomplete document: quiz03 is missing quiz03-high.pdf",
				"Incomplete document: midterm is missing midterm-average.pdf",
				"Incomplete document: final is missing final-instructions.pdf"))
		})
		It("does not warn when an assignment is absent altogether", func() {
			p := fx.clone().without(
				"20-assignments/asg05-average.pdf", "20-assignments/asg05-high.pdf", "20-assignments/asg05-instructions.pdf",
				"20-assignments/asg05-low.pdf", "20-assignments/asg05-scores.pdf")
			Expect(run(p, courseCode)).To(BeEmpty())
		})
		It("warns about the syllabus pdf, overall grades and empty lectures", func() {
			p := fx.clone().without("01-syllabus/syllabus-instructor.pdf", "90-grades/overall-grades.pdf")
			for k := range p {
				if strings.HasPrefix(k, "10-lectures/") {
					delete(p, k)
				}
			}
			p["10-lectures/"] = nil
			issues := run(p, courseCode)
			Expect(errorsOf(issues)).To(BeEmpty())
			Expect(titles(issues)).To(ConsistOf(
				"Incomplete document: syllabus-instructor.pdf is missing",
				"Incomplete document: overall-grades.pdf is missing",
				"Incomplete document: no lecture files found in 10-lectures"))
		})
		It("lists warnings after errors", func() {
			p := fx.clone().without("20-assignments/asg02-low.pdf").with("notes.txt", "x")
			issues := run(p, courseCode)
			Expect(issues).To(HaveLen(2))
			Expect(issues[0].Severity).To(Equal(domain.SeverityError))
			Expect(issues[1].Severity).To(Equal(domain.SeverityWarning))
		})
	})

	Describe("issue catalogue", func() {
		It("fills description and suggested_fix for every issue", func() {
			p := fx.clone().with("notes.txt", "x").with("10-lectures/Week1.pdf", "no").with(schedule, "week,topic\nx,y\n").
				without("20-assignments/asg02-low.pdf", syllabus)
			issues := run(p, courseCode)
			Expect(len(issues)).To(BeNumerically(">=", 5))
			for _, i := range issues {
				Expect(i.Title).NotTo(BeEmpty(), i.Code)
				Expect(i.Description).NotTo(BeEmpty(), i.Code)
				Expect(i.SuggestedFix).NotTo(BeEmpty(), i.Code)
				Expect(i.Stage).NotTo(BeEmpty())
			}
		})
	})
})
