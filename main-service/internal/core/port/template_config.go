package port

import (
	"context"
	"io"
	"time"

	"abet/main-service/internal/core/domain"
)

// CourseTemplateKey is the object that replaces the built-in course template zip (System Config, D29).
const CourseTemplateKey = "config/course-template.zip"

// CourseTemplateMetaUploader is the user-metadata key holding the uploader's e-mail.
const CourseTemplateMetaUploader = "uploaded-by"

// CourseTemplateInfo describes the template users download: the built-in default or the uploaded custom one.
type CourseTemplateInfo struct {
	Source     string // "default" | "custom"
	SizeBytes  *int64
	UploadedAt *time.Time
	UploadedBy *string
}

type GetCourseTemplateUseCase interface {
	Execute(ctx context.Context, actor *domain.Actor) (*CourseTemplateInfo, error)
}

type PutCourseTemplateUseCase interface {
	// Execute validates the zip read from file and stores it as the custom template.
	Execute(ctx context.Context, actor *domain.Actor, file io.Reader) (*CourseTemplateInfo, error)
}

type DeleteCourseTemplateUseCase interface {
	// Execute removes the custom template (idempotent) and returns the default info.
	Execute(ctx context.Context, actor *domain.Actor) (*CourseTemplateInfo, error)
}
