// Package validation is the immediate (pre-queue) validator of a portfolio zip. It is pure: no database, no
// disk, only an io.ReaderAt. The rules are a port of portfolio-service/scripts/runtime_pipeline.py
// (validate_submission, validate_structure, validate_course_files) and of the four CSV generator scripts,
// so that a package accepted here is also accepted by the worker; the messages are the SRS section 7 titles.
package validation

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"abet/main-service/internal/core/domain"
)

const (
	// MaxEntries and MaxUncompressedBytes guard against zip bombs.
	MaxEntries           = 2000
	MaxUncompressedBytes = 1 << 30

	maxCSVBytes = 32 << 20
	pdfProbe    = 1024
)

var requiredSubdirs = []string{
	"00-meta", "01-syllabus", "02-textbook", "03-schedule", "10-lectures", "20-assignments", "30-quizzes", "40-exams", "90-grades",
}

var requiredFiles = []string{
	identityPath, syllabusPath, schedulePath, gradingPath,
}

const (
	identityPath = "00-meta/instructor_course_identity_template.csv"
	syllabusPath = "01-syllabus/instructor_syllabus_abet_reference.csv"
	schedulePath = "03-schedule/instructor_course_schedule_template.csv"
	gradingPath  = "90-grades/instructor_course_grading_template.csv"
)

var allowedRootFiles = map[string]bool{
	"README_FIRST.txt":                         true,
	"INSTRUCTOR_DETAILED_SUBMISSION_GUIDE.md":  true,
	"INSTRUCTOR_DETAILED_SUBMISSION_GUIDE.pdf": true,
}

var allowedFilePatterns = map[string][]*regexp.Regexp{
	"00-meta": {
		regexp.MustCompile(`^course-meta\.tex$`),
		regexp.MustCompile(`^instructor_course_identity_template\.csv$`),
	},
	"01-syllabus": {
		regexp.MustCompile(`^instructor_syllabus_abet_reference\.csv$`),
		regexp.MustCompile(`^syllabus-abet\.tex$`),
		regexp.MustCompile(`^syllabus-instructor\.pdf$`),
	},
	"02-textbook": {regexp.MustCompile(`^textbook-part\d{2}\.pdf$`)},
	"03-schedule": {
		regexp.MustCompile(`^course-schedule\.tex$`),
		regexp.MustCompile(`^instructor_course_schedule_template\.csv$`),
		regexp.MustCompile(`^lecture_schedule_mapping\.csv$`),
	},
	"10-lectures": {regexp.MustCompile(`^week\d{2}-part\d{2}\.pdf$`)},
	"20-assignments": {
		regexp.MustCompile(`^asg-all-scores\.pdf$`),
		regexp.MustCompile(`^asg\d{2}-(average|high|instructions|low|scores)\.pdf$`),
	},
	"30-quizzes": {
		regexp.MustCompile(`^quiz-all-scores\.pdf$`),
		regexp.MustCompile(`^quiz\d{2}-(average|high|instructions|low|scores)\.pdf$`),
	},
	"40-exams": {regexp.MustCompile(`^(midterm|final)-(average|high|instructions|low|scores)\.pdf$`)},
	"90-grades": {
		regexp.MustCompile(`^course-grading\.tex$`),
		regexp.MustCompile(`^instructor_course_grading_template\.csv$`),
		regexp.MustCompile(`^overall-grades\.pdf$`),
	},
}

func isRequiredSubdir(name string) bool {
	for _, s := range requiredSubdirs {
		if s == name {
			return true
		}
	}
	return false
}

// Validate checks the zip behind r and returns every finding (errors first-found order, then warnings).
// expectedCourseCode is the code of the offering's course; when non-empty the identity CSV must carry the same one.
// A nil/empty result means the package is acceptable. The zip itself must be openable (the caller checks that first);
// if it is not, a single ZIP_ROOT-free structure error is returned.
func Validate(r io.ReaderAt, size int64, expectedCourseCode string) []domain.Issue {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return []domain.Issue{errIssue(CodeZipInvalid, "The file is not a readable zip archive.", "")}
	}
	if len(zr.File) > MaxEntries {
		return []domain.Issue{errIssue(CodeZipTooLarge, "Zip contains too many files: more than 2000 entries.", "")}
	}
	var total uint64
	for _, f := range zr.File {
		total += f.UncompressedSize64
	}
	if total > MaxUncompressedBytes {
		return []domain.Issue{errIssue(CodeZipTooLarge, "Zip expands to more than 1 GB when extracted.", "")}
	}

	v := &validator{budget: MaxUncompressedBytes, expectedCode: expectedCourseCode}
	course, issue := locateCourse(buildTree(zr.File))
	if issue != nil {
		return []domain.Issue{*issue}
	}
	v.checkStructure(course)
	v.checkFiles(course, "")
	v.checkCSVs(course)
	v.checkCompleteness(course)
	return append(v.errors, v.warnings...)
}

// locateCourse ports copy_source_to_job_dir + normalize_wrapped_course_layout: it returns the node that plays
// the role of the course folder.
func locateCourse(root *node) (*node, *domain.Issue) {
	var topDirs, items []*node
	for _, c := range root.sortedChildren() {
		items = append(items, c)
		if c.dir {
			topDirs = append(topDirs, c)
		}
	}
	if len(topDirs) == 1 {
		return findWrapped(topDirs[0]), nil
	}
	rootLayout := len(topDirs) > 0
	for _, d := range topDirs {
		if !isRequiredSubdir(d.name) && d.name != "99-abet" {
			rootLayout = false
		}
	}
	if (len(topDirs) == 0 || rootLayout) && len(items) > 0 {
		return findWrapped(root), nil
	}
	is := errIssue(CodeZipRoot, titleZipRoot(len(topDirs)), "")
	return nil, &is
}

type validator struct {
	errors       []domain.Issue
	warnings     []domain.Issue
	badText      map[string]bool // relpaths already reported as invalid text (no CSV parsing for them)
	budget       int64           // bytes still allowed to be read from the archive
	expectedCode string
}

func (v *validator) addErr(code, title, location string) {
	v.errors = append(v.errors, errIssue(code, title, location))
}

func (v *validator) addWarn(code, title, location string) {
	v.warnings = append(v.warnings, warnIssue(code, title, location))
}

// checkStructure ports validate_structure.
func (v *validator) checkStructure(course *node) {
	for _, s := range requiredSubdirs {
		if !course.hasDir(s) {
			v.addErr(CodeMissingFolder, titleMissingFolder(s), s)
		}
	}
	for _, rel := range requiredFiles {
		if course.fileAt(rel) == nil {
			v.addErr(CodeMissingFile, titleMissingFile(rel), rel)
		}
	}
}

// checkFiles ports validate_course_files: walk in sorted (depth-first) order like Python's sorted(rglob).
func (v *validator) checkFiles(n *node, prefix string) {
	for _, c := range n.sortedChildren() {
		rel := c.name
		if prefix != "" {
			rel = prefix + "/" + c.name
		}
		parts := strings.Split(rel, "/")
		if c.dir {
			if len(parts) == 1 {
				if !isRequiredSubdir(parts[0]) {
					v.addErr(CodeUnexpectedTopFolder, titleTopFolder(rel), rel)
				}
			} else {
				v.addErr(CodeUnexpectedNested, titleNestedFolder(rel), rel)
			}
			v.checkFiles(c, rel)
			continue
		}

		if len(parts) == 1 {
			if !allowedRootFiles[c.name] {
				v.addErr(CodeUnexpectedTopFile, titleTopFile(rel), rel)
			}
		} else {
			category := parts[0]
			switch {
			case !isRequiredSubdir(category):
				v.addErr(CodeUnexpectedOutside, titleOutside(rel), rel)
			case len(parts) != 2:
				v.addErr(CodeUnexpectedNestedFil, titleNestedFile(rel), rel)
			default:
				ok := false
				for _, re := range allowedFilePatterns[category] {
					if re.MatchString(c.name) {
						ok = true
						break
					}
				}
				if !ok {
					v.addErr(CodeUnexpectedFileName, titleFileName(category, rel), rel)
				}
			}
		}

		suffix := pySuffix(c.name)
		switch suffix {
		case ".pdf":
			if !v.validPDF(c) {
				v.addErr(CodeInvalidPDF, titleInvalidPDF(rel), rel)
			}
		case ".csv", ".tex", ".txt", ".md":
			if !v.validText(c) {
				v.addErr(CodeInvalidText, titleInvalidText(suffix, rel), rel)
				if v.badText == nil {
					v.badText = map[string]bool{}
				}
				v.badText[rel] = true
			}
		}
	}
}

// pySuffix mimics pathlib.PurePath.suffix(...).lower().
func pySuffix(name string) string {
	i := strings.LastIndex(name, ".")
	if i <= 0 || i == len(name)-1 {
		return ""
	}
	return strings.ToLower(name[i:])
}

// validPDF ports is_valid_pdf: "%PDF-" must occur in the first 1024 bytes.
func (v *validator) validPDF(n *node) bool {
	rc, err := n.file.Open()
	if err != nil {
		return false
	}
	defer rc.Close()
	head, _ := io.ReadAll(io.LimitReader(rc, pdfProbe))
	v.budget -= int64(len(head))
	return bytes.Contains(head, []byte("%PDF-"))
}

// validText ports is_valid_text_file: no NUL byte and valid UTF-8 (a BOM is valid UTF-8).
func (v *validator) validText(n *node) bool {
	rc, err := n.file.Open()
	if err != nil {
		return false
	}
	defer rc.Close()
	buf := make([]byte, 64<<10)
	var carry []byte
	for {
		if v.budget <= 0 {
			return false
		}
		k, rerr := rc.Read(buf)
		v.budget -= int64(k)
		chunk := append(carry, buf[:k]...)
		if bytes.IndexByte(chunk[len(carry):], 0) >= 0 {
			return false
		}
		// keep an incomplete trailing rune for the next round
		cut := len(chunk)
		if rerr == nil {
			for back := 1; back <= utf8.UTFMax-1 && back <= len(chunk); back++ {
				b := chunk[len(chunk)-back]
				if utf8.RuneStart(b) {
					if !utf8.FullRune(chunk[len(chunk)-back:]) {
						cut = len(chunk) - back
					}
					break
				}
			}
		}
		if !utf8.Valid(chunk[:cut]) {
			return false
		}
		carry = append([]byte(nil), chunk[cut:]...)
		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				return utf8.Valid(carry)
			}
			return false
		}
	}
}

// readCSVBytes reads a (valid-text) CSV from the archive, capped at maxCSVBytes.
func (v *validator) readCSVBytes(n *node) ([]byte, bool) {
	rc, err := n.file.Open()
	if err != nil {
		return nil, false
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, maxCSVBytes))
	v.budget -= int64(len(data))
	return data, err == nil
}
