package domain

import "github.com/google/uuid"

// ImportRow is one validated data row of the mapping CSV. Row is the 1-based row number (header = 1).
// Email == "" marks a course-only row (Role and permissions are then zero values).
type ImportRow struct {
	Row         int
	CourseCode  string
	CourseName  string
	Semester    int
	Email       string
	Role        Role
	CanUpload   bool
	CanDownload bool
}

// ImportError is a row-level problem; Row 1 is the header.
type ImportError struct {
	Row     int
	Message string
}

// ImportAccessState is a current course_access row of an offering in scope.
type ImportAccessState struct {
	UserID      uuid.UUID
	Email       string
	Role        Role
	CanUpload   bool
	CanDownload bool
	Source      string
}

// ImportOfferingState is a current offering of the (curriculum, year) scope.
type ImportOfferingState struct {
	OfferingID     uuid.UUID
	CourseID       uuid.UUID
	CourseCode     string
	CourseName     string
	Semester       int
	HasSubmissions bool
	Access         []ImportAccessState
}

// ImportUserState is an existing user referenced by the CSV.
type ImportUserState struct {
	UserID uuid.UUID
	Email  string
	Role   Role
}

// ImportState is everything the pure planner needs to know about the database.
type ImportState struct {
	Offerings []ImportOfferingState
	// Users maps lower-case e-mail -> existing user (only users mentioned in the CSV are required).
	Users map[string]ImportUserState
}

// Course plan actions.
const (
	CourseCreate        = "create"
	CourseUpdate        = "update"
	CourseKeep          = "keep"
	CourseDelete        = "delete"
	CourseKeepHasUpload = "keep_has_uploads"
)

// Access plan actions.
const (
	AccessAdd    = "add"
	AccessUpdate = "update"
	AccessRemove = "remove"
)

// User plan actions.
const (
	UserCreate    = "create"
	UserRaiseRole = "raise_role"
)

type ImportCoursePlan struct {
	Action     string
	CourseCode string
	CourseName string
	Semester   int
	Note       string
}

type ImportAccessPlan struct {
	Action      string
	CourseCode  string
	Semester    int
	Email       string
	Role        Role
	CanUpload   bool
	CanDownload bool
}

type ImportUserPlan struct {
	Action string
	Email  string
	Role   Role
}

type ImportSummary struct {
	Rows                   int
	CoursesCreate          int
	CoursesUpdate          int
	CoursesKeep            int
	CoursesDelete          int
	CoursesKeptWithUploads int
	AccessAdd              int
	AccessUpdate           int
	AccessRemove           int
	UsersCreate            int
	UsersRoleRaised        int
}

// ImportPlan is the outcome of the mapping CSV algorithm (preview, or what apply executed).
type ImportPlan struct {
	Applied bool
	Summary ImportSummary
	Courses []ImportCoursePlan
	Access  []ImportAccessPlan
	Users   []ImportUserPlan
	Errors  []ImportError
}
