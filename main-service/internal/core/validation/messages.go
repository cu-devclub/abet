package validation

import (
	"fmt"
	"strings"

	"abet/main-service/internal/core/domain"
)

// User-action texts of SRS section 7 (one per sub-section).
const (
	fixMissing   = "Ensure all 9 required subdirectories exist and that all 4 mandatory CSV templates are included."
	fixStructure = "Remove extra folders and move loose files into their designated category folders. Maintain a maximum depth of two levels (e.g., 10-lectures/week01-part01.pdf)."
	fixNaming    = "Rename files using the required pattern (e.g., weekNN-partNN.pdf, asgNN-instructions.pdf) and re-export corrupted PDF or CSV files as UTF-8 text."
	fixCSV       = "Open the affected CSV file, restore missing header columns, fill in required fields, and verify data types."
)

// Issue codes (stable identifiers, also used by the worker-failure parser).
const (
	CodeMissingFolder       = "MISSING_FOLDER"
	CodeMissingFile         = "MISSING_FILE"
	CodeIncompleteDocument  = "INCOMPLETE_DOCUMENT"
	CodeUnexpectedTopFolder = "UNEXPECTED_TOP_LEVEL_FOLDER"
	CodeUnexpectedNested    = "UNEXPECTED_NESTED_FOLDER"
	CodeUnexpectedTopFile   = "UNEXPECTED_TOP_LEVEL_FILE"
	CodeUnexpectedOutside   = "UNEXPECTED_FILE_OUTSIDE"
	CodeUnexpectedNestedFil = "UNEXPECTED_NESTED_FILE"
	CodeZipRoot             = "ZIP_ROOT"
	CodeZipTooLarge         = "ZIP_TOO_LARGE"
	CodeZipInvalid          = "ZIP_INVALID"
	CodeUnexpectedFileName  = "UNEXPECTED_FILE_NAME"
	CodeInvalidPDF          = "INVALID_PDF"
	CodeInvalidText         = "INVALID_TEXT"

	CodeIdentityNoHeader       = "IDENTITY_NO_HEADER"
	CodeIdentityRowCount       = "IDENTITY_ROW_COUNT"
	CodeIdentityCodeEmpty      = "IDENTITY_CODE_EMPTY"
	CodeIdentityMissingColumns = "IDENTITY_MISSING_COLUMNS"
	CodeIdentityCourseMismatch = "IDENTITY_COURSE_MISMATCH"
	CodeRowMissingValues       = "ROW_MISSING_VALUES"
	CodeSyllabusNoHeader       = "SYLLABUS_NO_HEADER"
	CodeSyllabusMissingColumns = "SYLLABUS_MISSING_COLUMNS"
	CodeSyllabusTemplateOnly   = "SYLLABUS_TEMPLATE_ONLY"
	CodeScheduleNoHeader       = "SCHEDULE_NO_HEADER"
	CodeScheduleMissingColumns = "SCHEDULE_MISSING_COLUMNS"
	CodeScheduleWeekRequired   = "SCHEDULE_WEEK_REQUIRED"
	CodeScheduleWeekInvalid    = "SCHEDULE_WEEK_INVALID"
	CodeScheduleWeekNonpos     = "SCHEDULE_WEEK_NONPOSITIVE"
	CodeScheduleEmpty          = "SCHEDULE_EMPTY"
	CodeGradingNoHeader        = "GRADING_NO_HEADER"
	CodeGradingMissingColumns  = "GRADING_MISSING_COLUMNS"
	CodeGradingBothRequired    = "GRADING_BOTH_REQUIRED"
	CodeGradingWeightNumeric   = "GRADING_WEIGHT_NUMERIC"
	CodeGradingWeightNegative  = "GRADING_WEIGHT_NEGATIVE"
	CodeGradingEmpty           = "GRADING_EMPTY"

	CodeBuildFailed      = "BUILD_FAILED"
	CodeBuildValidation  = "BUILD_VALIDATION"
	CodeBuildNoOutput    = "BUILD_NO_OUTPUT"
	CodeQueueUnavailable = "QUEUE_UNAVAILABLE"
	CodeBuildTimeout     = "BUILD_TIMEOUT"
)

// causeAndFix holds the SRS "Cause / Explanation" and "User Action Required" texts of a code.
type causeAndFix struct {
	stage domain.Stage
	cause string
	fix   string
}

var catalog = map[string]causeAndFix{
	CodeMissingFolder:      {domain.StageMissing, "One or more of the 9 required course folders (00-meta, 01-syllabus, 02-textbook, 03-schedule, 10-lectures, 20-assignments, 30-quizzes, 40-exams, 90-grades) are missing from the course package.", fixMissing},
	CodeMissingFile:        {domain.StageMissing, "One of the 4 core CSV templates (instructor_course_identity_template.csv, instructor_syllabus_abet_reference.csv, instructor_course_schedule_template.csv, or instructor_course_grading_template.csv) is missing.", fixMissing},
	CodeIncompleteDocument: {domain.StageMissing, "Expected course documents (such as lecture slides, assignments, quizzes, exam papers, or grade sheets) are incomplete.", fixMissing},

	CodeUnexpectedTopFolder: {domain.StageStructure, "An unrecognized folder exists at the root of the submission.", fixStructure},
	CodeUnexpectedNested:    {domain.StageStructure, "A subfolder was created inside a category directory (category folders must remain flat).", fixStructure},
	CodeUnexpectedTopFile:   {domain.StageStructure, "A file was placed in the root directory instead of inside a category folder.", fixStructure},
	CodeUnexpectedOutside:   {domain.StageStructure, "Files reside in unauthorized subdirectories.", fixStructure},
	CodeUnexpectedNestedFil: {domain.StageStructure, "Files are nested deeper than 2 directory levels.", fixStructure},
	CodeZipRoot:             {domain.StageStructure, "The zip must contain the course folder (or the 9 required folders) directly; several unrelated top-level folders were found.", fixStructure},
	CodeZipTooLarge:         {domain.StageStructure, "The archive contains too many files or expands to an unreasonable size.", "Remove unnecessary files and upload a smaller package (at most 2000 files and 1 GB uncompressed)."},
	CodeZipInvalid:          {domain.StageStructure, "The uploaded file could not be opened as a zip archive.", "Re-create the zip (for example by zipping the course folder again) and upload it again."},

	CodeUnexpectedFileName: {domain.StageNaming, "The filename does not match the strict naming contract defined in the specifications.", fixNaming},
	CodeInvalidPDF:         {domain.StageNaming, "The file has a .pdf extension but is corrupted, empty, or not a valid PDF.", fixNaming},
	CodeInvalidText:        {domain.StageNaming, "The CSV or text file contains binary corruption or is not encoded in UTF-8.", fixNaming},

	CodeIdentityNoHeader:       {domain.StageCSV, "Header row was removed.", fixCSV},
	CodeIdentityRowCount:       {domain.StageCSV, "The CSV is blank or contains multiple rows.", fixCSV},
	CodeIdentityCodeEmpty:      {domain.StageCSV, "The course id/course code field is missing or empty.", fixCSV},
	CodeIdentityMissingColumns: {domain.StageCSV, "Header columns were deleted or renamed.", fixCSV},
	CodeIdentityCourseMismatch: {domain.StageCSV, "The course code in the identity CSV differs from the course this portfolio is being uploaded to.", "Upload the portfolio to the matching course, or correct course_code in 00-meta/instructor_course_identity_template.csv."},
	CodeRowMissingValues:       {domain.StageCSV, "Mandatory fields in the data row are blank.", fixCSV},
	CodeSyllabusNoHeader:       {domain.StageCSV, "Header row was removed.", fixCSV},
	CodeSyllabusMissingColumns: {domain.StageCSV, "Missing core columns or outcome columns (SO1-SO7).", fixCSV},
	CodeSyllabusTemplateOnly:   {domain.StageCSV, "The CSV still contains placeholder text instead of real syllabus data.", fixCSV},
	CodeScheduleNoHeader:       {domain.StageCSV, "Header row was removed.", fixCSV},
	CodeScheduleMissingColumns: {domain.StageCSV, "Missing week or topic column headers.", fixCSV},
	CodeScheduleWeekRequired:   {domain.StageCSV, "A topic description exists without a week number.", fixCSV},
	CodeScheduleWeekInvalid:    {domain.StageCSV, "Week values must be positive integers (e.g., 1, 2, not \"Week 1\").", fixCSV},
	CodeScheduleWeekNonpos:     {domain.StageCSV, "Week value is zero or negative.", fixCSV},
	CodeScheduleEmpty:          {domain.StageCSV, "No valid schedule entries exist in the file.", fixCSV},
	CodeGradingNoHeader:        {domain.StageCSV, "Header row was removed.", fixCSV},
	CodeGradingMissingColumns:  {domain.StageCSV, "Missing component or weight_percent column headers.", fixCSV},
	CodeGradingBothRequired:    {domain.StageCSV, "Either component or weight_percent is missing in an active row.", fixCSV},
	CodeGradingWeightNumeric:   {domain.StageCSV, "Weight percentage contains non-numeric text (e.g., \"15%\" instead of 15.0).", fixCSV},
	CodeGradingWeightNegative:  {domain.StageCSV, "Weight percentage is negative.", fixCSV},
	CodeGradingEmpty:           {domain.StageCSV, "At least one grading component must be specified.", fixCSV},

	CodeBuildFailed:      {domain.StageBuild, "The portfolio build service could not generate the PDF.", "Check the files in your package against the instructor package template and upload again. If the problem persists, contact an administrator."},
	CodeBuildValidation:  {domain.StageBuild, "The portfolio build service rejected the package during its own validation.", "Fix the problem described above and upload the package again."},
	CodeBuildNoOutput:    {domain.StageBuild, "The portfolio build finished without producing a PDF.", "Upload the package again. If the problem persists, contact an administrator."},
	CodeQueueUnavailable: {domain.StageBuild, "The portfolio build queue could not be reached.", "Please try again later."},
	CodeBuildTimeout:     {domain.StageBuild, "The portfolio build service did not finish in time.", "Upload the folder again. If this keeps happening, contact the system administrator."},
}

// NewIssue builds an issue of the given code with the SRS cause/fix filled in. Unknown codes yield a bare error issue.
func NewIssue(severity domain.Severity, code, title, location string) domain.Issue {
	c, ok := catalog[code]
	if !ok {
		return domain.Issue{Severity: severity, Stage: domain.StageBuild, Code: code, Title: title, Location: location}
	}
	return domain.Issue{
		Severity: severity, Stage: c.stage, Code: code, Title: title,
		Description: c.cause, Location: location, SuggestedFix: c.fix,
	}
}

func errIssue(code, title, location string) domain.Issue {
	return NewIssue(domain.SeverityError, code, title, location)
}

func warnIssue(code, title, location string) domain.Issue {
	return NewIssue(domain.SeverityWarning, code, title, location)
}

// Titles (SRS section 7, placeholders filled).

func titleMissingFolder(subdir string) string { return "Missing required folder: " + subdir }
func titleMissingFile(relpath string) string  { return "Missing required file: " + relpath }
func titleTopFolder(relpath string) string    { return "Unexpected top-level folder: " + relpath }
func titleNestedFolder(relpath string) string { return "Unexpected nested folder: " + relpath }
func titleTopFile(relpath string) string      { return "Unexpected top-level file: " + relpath }
func titleOutside(relpath string) string {
	return "Unexpected file outside required folders: " + relpath
}
func titleNestedFile(relpath string) string { return "Unexpected nested file path: " + relpath }
func titleFileName(category, rel string) string {
	return fmt.Sprintf("Unexpected file name in %s: %s", category, rel)
}
func titleInvalidPDF(relpath string) string {
	return "Invalid PDF content for file with .pdf extension: " + relpath
}
func titleInvalidText(suffix, relpath string) string {
	return fmt.Sprintf("Invalid text content for file with %s extension: %s", suffix, relpath)
}
func titleZipRoot(n int) string {
	return fmt.Sprintf("Submission zip must contain exactly one top-level folder. Found %d.", n)
}
func titleRowMissing(n int, cols []string) string {
	return fmt.Sprintf("Row %d missing required values: %s", n, strings.Join(cols, ", "))
}
func titleIncomplete(what string) string { return "Incomplete document: " + what }
