package validation_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/validation"
)

var _ = Describe("ParseWorkerFailure", func() {
	It("extracts bullet lines after Errors: and maps known messages", func() {
		text := "Pipeline processing failed for job 6f1c2c1e-1111-4a2b-9c3d-000000000001: Validation failed for 2301173-Calculus.\n" +
			"Errors:\n" +
			"  • Unexpected file name in 10-lectures: 10-lectures/Week1.pdf\n" +
			"  • Invalid PDF content for file with .pdf extension: 20-assignments/asg01-high.pdf\n" +
			"  • Row 3: invalid week 'Week 1'. Use integers.\n" +
			"  • Some brand new worker complaint\n" +
			"\n--- Validation Report (/tmp/x/report.md) ---\n# Report\n  • not an error line\n"
		issues := validation.ParseWorkerFailure(text)
		Expect(titles(issues)).To(Equal([]string{
			"Unexpected file name in 10-lectures: 10-lectures/Week1.pdf",
			"Invalid PDF content for file with .pdf extension: 20-assignments/asg01-high.pdf",
			"Row 3: invalid week 'Week 1'. Use integers",
			"Some brand new worker complaint",
		}))
		Expect(codes(issues)).To(Equal([]string{"UNEXPECTED_FILE_NAME", "INVALID_PDF", "SCHEDULE_WEEK_INVALID", "BUILD_VALIDATION"}))
		Expect(issues[0].Stage).To(Equal(domain.StageNaming))
		Expect(issues[0].Location).To(Equal("10-lectures/Week1.pdf"))
		Expect(issues[0].SuggestedFix).NotTo(BeEmpty())
		Expect(issues[3].Stage).To(Equal(domain.StageBuild))
		Expect(issues[3].Severity).To(Equal(domain.SeverityError))
	})

	It("accepts '-' bullets and CRLF", func() {
		text := "Pipeline processing failed for job x: Validation failed for c.\r\nErrors:\r\n  - Missing required folder: 02-textbook\r\n  - Missing required file: 90-grades/instructor_course_grading_template.csv\r\n"
		Expect(titles(validation.ParseWorkerFailure(text))).To(Equal([]string{
			"Missing required folder: 02-textbook",
			"Missing required file: 90-grades/instructor_course_grading_template.csv",
		}))
	})

	It("maps a bare zip-root failure", func() {
		issues := validation.ParseWorkerFailure("Pipeline processing failed for job abc: Submission zip must contain exactly one top-level folder. Found 3.")
		Expect(issues).To(HaveLen(1))
		Expect(issues[0].Code).To(Equal("ZIP_ROOT"))
		Expect(issues[0].Title).To(Equal("Submission zip must contain exactly one top-level folder. Found 3."))
	})

	It("maps a bare missing-columns failure to the right CSV", func() {
		cases := map[string]string{
			"Missing required columns: topic":                                "SCHEDULE_MISSING_COLUMNS",
			"Missing required columns: component, weight_percent":            "GRADING_MISSING_COLUMNS",
			"Missing required columns: program_name, school_name":            "IDENTITY_MISSING_COLUMNS",
			"Missing required columns: ice_outcome_7, prepared_by":           "SYLLABUS_MISSING_COLUMNS",
			"Missing required columns: course_name":                          "SYLLABUS_MISSING_COLUMNS",
			"Missing required columns: course_name, semester_label":          "IDENTITY_MISSING_COLUMNS",
			"No grading rows found in CSV.":                                  "GRADING_EMPTY",
			"No non-template syllabus rows found in CSV.":                    "SYLLABUS_TEMPLATE_ONLY",
			"Row 2: weight_percent must be numeric.":                         "GRADING_WEIGHT_NUMERIC",
			"Identity CSV must contain exactly one populated row.":           "IDENTITY_ROW_COUNT",
			"Row 2 missing required values: course_instructors, school_name": "ROW_MISSING_VALUES",
		}
		for msg, code := range cases {
			issues := validation.ParseWorkerFailure("Pipeline processing failed for job j: " + msg)
			Expect(codes(issues)).To(Equal([]string{code}), msg)
		}
	})

	It("falls back to BUILD_FAILED with the tail of the text and no traceback", func() {
		latex := "latexmk failed for /tmp/x/Course-main-tex/main.tex (exit code 12).\n--- LaTeX Log Errors ---\n! Undefined control sequence.\nl.42 \\foo\n"
		issues := validation.ParseWorkerFailure("Pipeline processing failed for job j: " + latex)
		Expect(issues).To(HaveLen(1))
		Expect(issues[0].Code).To(Equal("BUILD_FAILED"))
		Expect(issues[0].Title).To(Equal("PDF generation failed"))
		Expect(issues[0].Stage).To(Equal(domain.StageBuild))
		Expect(issues[0].Description).To(ContainSubstring("Undefined control sequence"))
	})

	It("cuts a Python traceback and caps the description at 2000 characters", func() {
		long := strings.Repeat("x", 3000)
		text := "Pipeline processing failed for job j: " + long + "\nTraceback (most recent call last):\n  File \"a.py\", line 1\nRuntimeError: boom"
		issues := validation.ParseWorkerFailure(text)
		Expect(issues).To(HaveLen(1))
		Expect([]rune(issues[0].Description)).To(HaveLen(2000))
		Expect(issues[0].Description).NotTo(ContainSubstring("Traceback"))
	})

	It("handles empty text and Errors: without bullets", func() {
		Expect(codes(validation.ParseWorkerFailure(""))).To(Equal([]string{"BUILD_FAILED"}))
		Expect(codes(validation.ParseWorkerFailure("Validation failed for c.\nErrors:\n\nnothing"))).To(Equal([]string{"BUILD_FAILED"}))
	})
})

// Captured from the real worker (abet-portfolio-service) during the T05 end-to-end run: the error text starts with the
// pipeline's log output, then the SystemExit message with the "Errors:" block, then the validation report.
var _ = Describe("ParseWorkerFailure with real worker output", func() {
	It("parses the validation failure of a package the worker rejects", func() {
		text := "Pipeline processing failed for job probe-0001: 2026-10-09 15:39:13,111 [INFO] [Config] Connecting to MariaDB at abet-mariadb:3306 (DB: 'abet_db')...\n" +
			"2026-10-09 15:39:13,158 [WARNING] [Config] MariaDB unavailable. Falling back to settings.json...\n" +
			"2026-10-09 15:39:13,195 [INFO] [Config] Initial validation failed for probe with 4 error(s). Attempting auto-repair...\n" +
			"2026-10-09 15:39:13,224 [ERROR] [Config] Validation failed for probe with 4 error(s):\n" +
			"  • Missing required folder: 90-grades\n" +
			"  • Missing required file: 90-grades/instructor_course_grading_template.csv\n" +
			"2026-10-09 15:39:13,224 [ERROR] [Config] --- Validation Report (probe-validation.md) ---\n" +
			"# Validation Report: probe\n## Errors\n- Missing required folder: 90-grades\n" +
			"Validation failed for probe.\nErrors:\n" +
			"  • Missing required folder: 90-grades\n" +
			"  • Missing required file: 90-grades/instructor_course_grading_template.csv\n" +
			"  • Unexpected file name in 10-lectures: 10-lectures/Week1.pdf\n" +
			"  • Invalid PDF content for file with .pdf extension: 10-lectures/Week1.pdf\n\n" +
			"--- Validation Report (/tmp/x/probe-validation.md) ---\n# Validation Report: probe\n## Errors\n- Missing required folder: 90-grades\n"
		issues := validation.ParseWorkerFailure(text)
		Expect(titles(issues)).To(Equal([]string{
			"Missing required folder: 90-grades",
			"Missing required file: 90-grades/instructor_course_grading_template.csv",
			"Unexpected file name in 10-lectures: 10-lectures/Week1.pdf",
			"Invalid PDF content for file with .pdf extension: 10-lectures/Week1.pdf",
		}))
		Expect(codes(issues)).To(Equal([]string{"MISSING_FOLDER", "MISSING_FILE", "UNEXPECTED_FILE_NAME", "INVALID_PDF"}))
	})

	It("keeps the latexmk message of a LaTeX failure, not the log noise before it", func() {
		text := "Pipeline processing failed for job j: 2026-10-09 15:38:52,958 [INFO] [Config] Auto-registered missing course\n" +
			"2026-10-09 15:38:55,155 [ERROR] [Config] latexmk failed for /tmp/x/main.tex (exit code 12).\n--- LaTeX Log Errors ---\nlibxpdf: noise\n" +
			"latexmk failed for /tmp/x/main.tex (exit code 12).\n--- LaTeX Log Errors ---\n==> Fatal error occurred, no output PDF file produced!\n--- latexmk stderr ---\nlibxpdf: Syntax Error: Couldn't read page catalog\n"
		issues := validation.ParseWorkerFailure(text)
		Expect(issues).To(HaveLen(1))
		Expect(issues[0].Code).To(Equal("BUILD_FAILED"))
		Expect(issues[0].Description).To(HavePrefix("latexmk failed for /tmp/x/main.tex"))
		Expect(issues[0].Description).To(ContainSubstring("Couldn't read page catalog"))
		Expect(issues[0].Description).NotTo(ContainSubstring("Auto-registered"))
	})
})
