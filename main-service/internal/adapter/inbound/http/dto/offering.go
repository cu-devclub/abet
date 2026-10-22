package dto

import (
	"github.com/google/uuid"

	"abet/main-service/internal/core/domain"
	"abet/main-service/internal/core/port"
)

type OfferingResponse struct {
	ID              uuid.UUID   `json:"id"`
	CourseID        uuid.UUID   `json:"course_id"`
	CourseCode      string      `json:"course_code"`
	CourseName      string      `json:"course_name"`
	AcademicYear    int         `json:"academic_year"`
	Semester        int         `json:"semester"`
	Curriculum      RefResponse `json:"curriculum"`
	Department      RefResponse `json:"department"`
	InstructorCount int         `json:"instructor_count"`
	StaffCount      int         `json:"staff_count"`
	HasSubmissions  bool        `json:"has_submissions"`
	CreatedAt       string      `json:"created_at"`
	UpdatedAt       string      `json:"updated_at"`
}

func OfferingFromDomain(o *domain.Offering) OfferingResponse {
	return OfferingResponse{
		ID: o.ID, CourseID: o.CourseID, CourseCode: o.CourseCode, CourseName: o.CourseName,
		AcademicYear: o.AcademicYear, Semester: o.Semester,
		Curriculum: RefFromDomain(o.Curriculum), Department: RefFromDomain(o.Department),
		InstructorCount: o.InstructorCount, StaffCount: o.StaffCount, HasSubmissions: o.HasSubmissions,
		CreatedAt: FormatTime(o.CreatedAt), UpdatedAt: FormatTime(o.UpdatedAt),
	}
}

func OfferingsFromDomain(os []domain.Offering) []OfferingResponse {
	out := make([]OfferingResponse, 0, len(os))
	for i := range os {
		out = append(out, OfferingFromDomain(&os[i]))
	}
	return out
}

// YearsResponse is the body of GET /offerings/years.
type YearsResponse struct {
	Items   []int `json:"items"`
	Current int   `json:"current"`
}

type CreateOfferingRequest struct {
	CourseCode   string    `json:"course_code" validate:"required,max=50"`
	CourseName   string    `json:"course_name" validate:"required,max=300"`
	CurriculumID uuid.UUID `json:"curriculum_id" validate:"required"`
	AcademicYear int       `json:"academic_year" validate:"required,min=2000,max=2200"`
	Semester     int       `json:"semester" validate:"required,min=1,max=3"`
}

func (r CreateOfferingRequest) Validate() error { return Validate(r) }

func (r CreateOfferingRequest) ToInput() port.CreateOfferingInput {
	return port.CreateOfferingInput{
		CourseCode: r.CourseCode, CourseName: r.CourseName, CurriculumID: r.CurriculumID,
		AcademicYear: r.AcademicYear, Semester: r.Semester,
	}
}

type UpdateOfferingRequest struct {
	CourseCode   *string    `json:"course_code" validate:"omitempty,max=50"`
	CourseName   *string    `json:"course_name" validate:"omitempty,max=300"`
	CurriculumID *uuid.UUID `json:"curriculum_id"`
	AcademicYear *int       `json:"academic_year" validate:"omitempty,min=2000,max=2200"`
	Semester     *int       `json:"semester" validate:"omitempty,min=1,max=3"`
}

func (r UpdateOfferingRequest) Validate() error { return Validate(r) }

func (r UpdateOfferingRequest) ToInput(id uuid.UUID) port.UpdateOfferingInput {
	return port.UpdateOfferingInput{
		ID: id, CourseCode: r.CourseCode, CourseName: r.CourseName, CurriculumID: r.CurriculumID,
		AcademicYear: r.AcademicYear, Semester: r.Semester,
	}
}

type BulkDeleteRequest struct {
	IDs []uuid.UUID `json:"ids" validate:"required,min=1,max=1000"`
}

func (r BulkDeleteRequest) Validate() error { return Validate(r) }

type BulkDeleteResponse struct {
	Deleted int `json:"deleted"`
}

// ---- access ----

type AccessEntryResponse struct {
	UserID        uuid.UUID `json:"user_id"`
	Name          string    `json:"name"`
	Email         string    `json:"email"`
	Role          string    `json:"role"`
	CanUpload     bool      `json:"can_upload"`
	CanDownload   bool      `json:"can_download"`
	IsDefault     bool      `json:"is_default"`
	DefaultReason *string   `json:"default_reason"`
	Source        string    `json:"source"`
}

func AccessEntryFromDomain(e domain.AccessEntry) AccessEntryResponse {
	return AccessEntryResponse{
		UserID: e.UserID, Name: e.Name, Email: e.Email, Role: string(e.Role),
		CanUpload: e.CanUpload, CanDownload: e.CanDownload, IsDefault: e.IsDefault,
		DefaultReason: e.DefaultReason, Source: e.Source,
	}
}

func AccessEntriesFromDomain(es []domain.AccessEntry) []AccessEntryResponse {
	out := make([]AccessEntryResponse, 0, len(es))
	for _, e := range es {
		out = append(out, AccessEntryFromDomain(e))
	}
	return out
}

type SetAccessRequest struct {
	CanUpload   bool `json:"can_upload"`
	CanDownload bool `json:"can_download"`
}

func (r SetAccessRequest) Validate() error { return nil }

// ---- mapping CSV import ----

type ImportSummaryResponse struct {
	Rows                   int `json:"rows"`
	CoursesCreate          int `json:"courses_create"`
	CoursesUpdate          int `json:"courses_update"`
	CoursesKeep            int `json:"courses_keep"`
	CoursesDelete          int `json:"courses_delete"`
	CoursesKeptWithUploads int `json:"courses_kept_with_uploads"`
	AccessAdd              int `json:"access_add"`
	AccessUpdate           int `json:"access_update"`
	AccessRemove           int `json:"access_remove"`
	UsersCreate            int `json:"users_create"`
	UsersRoleRaised        int `json:"users_role_raised"`
}

type ImportCourseResponse struct {
	Action     string `json:"action"`
	CourseCode string `json:"course_code"`
	CourseName string `json:"course_name"`
	Semester   int    `json:"semester"`
	Note       string `json:"note"`
}

type ImportAccessResponse struct {
	Action      string `json:"action"`
	CourseCode  string `json:"course_code"`
	Semester    int    `json:"semester"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	CanUpload   bool   `json:"can_upload"`
	CanDownload bool   `json:"can_download"`
}

type ImportUserResponse struct {
	Action string `json:"action"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

type ImportErrorResponse struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

type ImportPlanResponse struct {
	Applied bool                   `json:"applied"`
	Summary ImportSummaryResponse  `json:"summary"`
	Courses []ImportCourseResponse `json:"courses"`
	Access  []ImportAccessResponse `json:"access"`
	Users   []ImportUserResponse   `json:"users"`
	Errors  []ImportErrorResponse  `json:"errors"`
}

func ImportPlanFromDomain(p *domain.ImportPlan) ImportPlanResponse {
	s := p.Summary
	out := ImportPlanResponse{
		Applied: p.Applied,
		Summary: ImportSummaryResponse{
			Rows: s.Rows, CoursesCreate: s.CoursesCreate, CoursesUpdate: s.CoursesUpdate, CoursesKeep: s.CoursesKeep,
			CoursesDelete: s.CoursesDelete, CoursesKeptWithUploads: s.CoursesKeptWithUploads,
			AccessAdd: s.AccessAdd, AccessUpdate: s.AccessUpdate, AccessRemove: s.AccessRemove,
			UsersCreate: s.UsersCreate, UsersRoleRaised: s.UsersRoleRaised,
		},
		Courses: make([]ImportCourseResponse, 0, len(p.Courses)),
		Access:  make([]ImportAccessResponse, 0, len(p.Access)),
		Users:   make([]ImportUserResponse, 0, len(p.Users)),
		Errors:  make([]ImportErrorResponse, 0, len(p.Errors)),
	}
	for _, c := range p.Courses {
		out.Courses = append(out.Courses, ImportCourseResponse{Action: c.Action, CourseCode: c.CourseCode, CourseName: c.CourseName, Semester: c.Semester, Note: c.Note})
	}
	for _, a := range p.Access {
		out.Access = append(out.Access, ImportAccessResponse{
			Action: a.Action, CourseCode: a.CourseCode, Semester: a.Semester, Email: a.Email,
			Role: string(a.Role), CanUpload: a.CanUpload, CanDownload: a.CanDownload,
		})
	}
	for _, u := range p.Users {
		out.Users = append(out.Users, ImportUserResponse{Action: u.Action, Email: u.Email, Role: string(u.Role)})
	}
	for _, e := range p.Errors {
		out.Errors = append(out.Errors, ImportErrorResponse{Row: e.Row, Message: e.Message})
	}
	return out
}
