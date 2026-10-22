package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	MinAcademicYear = 2000
	MaxAcademicYear = 2200
)

// Offering is a course offered in one academic year and semester (course_offerings joined with its course).
type Offering struct {
	ID              uuid.UUID
	CourseID        uuid.UUID
	CourseCode      string
	CourseName      string
	AcademicYear    int
	Semester        int
	Curriculum      Ref
	Department      Ref
	InstructorCount int
	StaffCount      int
	HasSubmissions  bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Scope returns the organisational position of the offering (for authorization checks).
func (o *Offering) Scope() OfferingScope {
	return OfferingScope{CurriculumID: o.Curriculum.ID, DepartmentID: o.Department.ID}
}

// ValidateAcademicYear checks the CE year range accepted by the schema.
func ValidateAcademicYear(year int) error {
	if year < MinAcademicYear || year > MaxAcademicYear {
		return fmt.Errorf("%w: academic_year must be between %d and %d", ErrInvalidInput, MinAcademicYear, MaxAcademicYear)
	}
	return nil
}

// ValidateSemester checks the semester (3 = Summer).
func ValidateSemester(semester int) error {
	if semester < 1 || semester > 3 {
		return fmt.Errorf("%w: semester must be 1, 2 or 3", ErrInvalidInput)
	}
	return nil
}

// NormalizeCourse trims code and name and rejects empty values.
func NormalizeCourse(code, name string) (string, string, error) {
	code, name = strings.TrimSpace(code), strings.TrimSpace(name)
	if code == "" {
		return "", "", fmt.Errorf("%w: course_code cannot be empty", ErrInvalidInput)
	}
	if name == "" {
		return "", "", fmt.Errorf("%w: course_name cannot be empty", ErrInvalidInput)
	}
	return code, name, nil
}

// Access sources stored in course_access.source (and "default" for derived entries).
const (
	AccessSourceManual  = "manual"
	AccessSourceCSV     = "csv"
	AccessSourceDefault = "default"
)

// AccessEntry is one row of the access list of an offering (explicit or derived default).
type AccessEntry struct {
	UserID        uuid.UUID
	Name          string
	Email         string
	Role          Role
	CanUpload     bool
	CanDownload   bool
	IsDefault     bool
	DefaultReason *string
	Source        string
}

// DefaultAccessReason is the explanation shown for the staff default (A3).
func DefaultAccessReason(departmentCode string) string {
	return fmt.Sprintf("Staff in department %s (default download)", departmentCode)
}
