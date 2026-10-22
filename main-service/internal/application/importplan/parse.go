// Package importplan implements the mapping CSV import (D2-D4, D8-D10) as pure functions:
// ParseCSV turns the file into validated rows and BuildImportPlan diffs them against the current state.
// Nothing in here touches the database or the file system.
package importplan

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"strconv"
	"strings"

	"abet/main-service/internal/core/domain"
)

// ParseResult is the outcome of ParseCSV. RowCount is the number of non-blank data rows (valid or not).
type ParseResult struct {
	Rows     []domain.ImportRow
	Errors   []domain.ImportError
	RowCount int
}

const (
	colCourseID = iota
	colCourseName
	colSemester
	colEmail
	colRole
	colUpload
	colDownload
	numCols
)

var colLabels = [numCols]string{"course id", "course name", "semester", "user email", "user role", "upload permission", "download permission"}

// headerAliases maps a normalised header (lower case, no spaces/underscores/hyphens) to its column.
var headerAliases = map[string]int{
	"courseid": colCourseID, "coursecode": colCourseID,
	"coursename": colCourseName,
	"semester":   colSemester,
	"useremail":  colEmail, "email": colEmail,
	"userrole": colRole, "role": colRole,
	"uploadpermission": colUpload, "upload": colUpload,
	"downloadpermission": colDownload, "download": colDownload,
}

func normalizeHeader(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.NewReplacer(" ", "", "_", "", "-", "", " ", "").Replace(s)
}

// ParseCSV validates the whole file and returns every problem it finds (row 1 is the header).
// Valid rows are returned even when other rows have errors so that a preview can still show the plan.
func ParseCSV(data []byte, allowedDomains []string) ParseResult {
	var res ParseResult
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))

	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	r.TrimLeadingSpace = true

	header, err := r.Read()
	if errors.Is(err, io.EOF) {
		res.Errors = append(res.Errors, domain.ImportError{Row: 1, Message: "file is empty"})
		return res
	}
	if err != nil {
		res.Errors = append(res.Errors, domain.ImportError{Row: errRow(err, 1), Message: "cannot read CSV: " + errText(err)})
		return res
	}

	colIdx := [numCols]int{-1, -1, -1, -1, -1, -1, -1}
	for i, h := range header {
		if c, ok := headerAliases[normalizeHeader(h)]; ok && colIdx[c] < 0 {
			colIdx[c] = i
		}
	}
	for _, c := range []int{colCourseID, colCourseName, colSemester} {
		if colIdx[c] < 0 {
			res.Errors = append(res.Errors, domain.ImportError{Row: 1, Message: "missing column: " + colLabels[c]})
		}
	}
	if len(res.Errors) > 0 {
		return res
	}

	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			res.Errors = append(res.Errors, domain.ImportError{Row: errRow(err, 0), Message: "cannot read CSV: " + errText(err)})
			break
		}
		line, _ := r.FieldPos(0)
		get := func(c int) string {
			if colIdx[c] < 0 || colIdx[c] >= len(rec) {
				return ""
			}
			return strings.TrimSpace(rec[colIdx[c]])
		}
		blank := true
		for _, f := range rec {
			if strings.TrimSpace(f) != "" {
				blank = false
				break
			}
		}
		if blank {
			continue
		}
		res.RowCount++
		row, errs := parseRow(line, get, allowedDomains)
		if len(errs) > 0 {
			res.Errors = append(res.Errors, errs...)
			continue
		}
		res.Rows = append(res.Rows, row)
	}
	if res.RowCount == 0 {
		// A header-only file would otherwise delete every offering without uploads.
		res.Errors = append(res.Errors, domain.ImportError{Row: 2, Message: "the file has no data rows"})
	}
	return res
}

func parseRow(line int, get func(int) string, allowedDomains []string) (domain.ImportRow, []domain.ImportError) {
	var errs []domain.ImportError
	bad := func(msg string) { errs = append(errs, domain.ImportError{Row: line, Message: msg}) }

	row := domain.ImportRow{Row: line, CourseCode: get(colCourseID), CourseName: get(colCourseName)}
	if row.CourseCode == "" {
		bad("course id is required")
	}
	if row.CourseName == "" {
		bad("course name is required")
	}
	switch sem := get(colSemester); sem {
	case "1", "2", "3":
		row.Semester, _ = strconv.Atoi(sem)
	default:
		bad("semester must be 1, 2 or 3")
	}

	email, roleRaw, upRaw, downRaw := get(colEmail), get(colRole), get(colUpload), get(colDownload)
	if email == "" {
		if roleRaw != "" || upRaw != "" || downRaw != "" {
			bad("user email is required when user role or permissions are given")
		}
		return row, errs
	}

	if norm, err := domain.ValidateEmail(email, allowedDomains); err != nil {
		bad(strings.TrimPrefix(err.Error(), domain.ErrInvalidInput.Error()+": "))
	} else {
		row.Email = norm
	}
	switch strings.ToLower(roleRaw) {
	case "instructor":
		row.Role = domain.RoleInstructor
	case "staff":
		row.Role = domain.RoleStaff
	default:
		bad("user role must be Instructor or Staff")
	}
	if v, ok := parseBool(upRaw); ok {
		row.CanUpload = v
	} else {
		bad("upload permission must be Yes or No")
	}
	if v, ok := parseBool(downRaw); ok {
		row.CanDownload = v
	} else {
		bad("download permission must be Yes or No")
	}
	return row, errs
}

func parseBool(s string) (value, ok bool) {
	switch strings.ToLower(s) {
	case "yes", "true", "y", "1":
		return true, true
	case "no", "false", "n", "0":
		return false, true
	}
	return false, false
}

func errRow(err error, fallback int) int {
	var pe *csv.ParseError
	if errors.As(err, &pe) && pe.StartLine > 0 {
		return pe.StartLine
	}
	return fallback
}

func errText(err error) string {
	var pe *csv.ParseError
	if errors.As(err, &pe) && pe.Err != nil {
		return pe.Err.Error()
	}
	return err.Error()
}
