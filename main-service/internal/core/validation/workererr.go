package validation

import (
	"fmt"
	"regexp"
	"strings"

	"abet/main-service/internal/core/domain"
)

const maxFailureDescription = 2000

var (
	jobPrefixRe = regexp.MustCompile(`^Pipeline processing failed for job [^:]*:\s*`)
	rowNumRe    = `(\d+)`

	// messagePatterns map the worker's own validation messages (runtime_pipeline.py and the generate_* scripts)
	// to the SRS issue catalogue. Order matters: first match wins.
	messagePatterns = []struct {
		re   *regexp.Regexp
		make func(m []string) domain.Issue
	}{
		{regexp.MustCompile(`^Missing required folder: (.+?)\.?$`), func(m []string) domain.Issue {
			return errIssue(CodeMissingFolder, titleMissingFolder(m[1]), m[1])
		}},
		{regexp.MustCompile(`^Missing required file: (.+?)\.?$`), func(m []string) domain.Issue {
			return errIssue(CodeMissingFile, titleMissingFile(m[1]), m[1])
		}},
		{regexp.MustCompile(`^Unexpected top-level folder: (.+?)\.?$`), func(m []string) domain.Issue {
			return errIssue(CodeUnexpectedTopFolder, titleTopFolder(m[1]), m[1])
		}},
		{regexp.MustCompile(`^Unexpected nested folder: (.+?)\.?$`), func(m []string) domain.Issue {
			return errIssue(CodeUnexpectedNested, titleNestedFolder(m[1]), m[1])
		}},
		{regexp.MustCompile(`^Unexpected top-level file: (.+?)\.?$`), func(m []string) domain.Issue {
			return errIssue(CodeUnexpectedTopFile, titleTopFile(m[1]), m[1])
		}},
		{regexp.MustCompile(`^Unexpected file outside required folders: (.+?)\.?$`), func(m []string) domain.Issue {
			return errIssue(CodeUnexpectedOutside, titleOutside(m[1]), m[1])
		}},
		{regexp.MustCompile(`^Unexpected nested file path: (.+?)\.?$`), func(m []string) domain.Issue {
			return errIssue(CodeUnexpectedNestedFil, titleNestedFile(m[1]), m[1])
		}},
		{regexp.MustCompile(`^Unexpected file name in ([^:]+): (.+?)\.?$`), func(m []string) domain.Issue {
			return errIssue(CodeUnexpectedFileName, titleFileName(m[1], m[2]), m[2])
		}},
		{regexp.MustCompile(`^Invalid PDF content for file with \.pdf extension: (.+?)\.?$`), func(m []string) domain.Issue {
			return errIssue(CodeInvalidPDF, titleInvalidPDF(m[1]), m[1])
		}},
		{regexp.MustCompile(`^Invalid text content for file with (\S+) extension: (.+?)\.?$`), func(m []string) domain.Issue {
			return errIssue(CodeInvalidText, titleInvalidText(m[1], m[2]), m[2])
		}},
		{regexp.MustCompile(`^Submission zip must contain exactly one top-level folder\. Found (\d+)\.?$`), func(m []string) domain.Issue {
			var n int
			_, _ = fmt.Sscanf(m[1], "%d", &n)
			return errIssue(CodeZipRoot, titleZipRoot(n), "")
		}},
		{regexp.MustCompile(`^Identity CSV must contain exactly one populated row\.?$`), func([]string) domain.Issue {
			return errIssue(CodeIdentityRowCount, "Identity CSV must contain exactly one populated row", identityPath)
		}},
		{regexp.MustCompile(`^Identity CSV course_code field is empty\.?$`), func([]string) domain.Issue {
			return errIssue(CodeIdentityCodeEmpty, "Identity CSV course code field is empty", identityPath)
		}},
		{regexp.MustCompile(`^Row ` + rowNumRe + ` missing required values: (.+?)\.?$`), func(m []string) domain.Issue {
			return errIssue(CodeRowMissingValues, fmt.Sprintf("Row %s missing required values: %s", m[1], m[2]), identityPath+": row "+m[1])
		}},
		{regexp.MustCompile(`^Missing required columns: (.+?)\.?$`), func(m []string) domain.Issue {
			return missingColumnsIssue(strings.Split(m[1], ", "))
		}},
		{regexp.MustCompile(`^No non-template syllabus rows found in CSV\.?$`), func([]string) domain.Issue {
			return errIssue(CodeSyllabusTemplateOnly, "No non-template syllabus rows found in CSV", syllabusPath)
		}},
		{regexp.MustCompile(`^Row ` + rowNumRe + `: week is required when topic is provided\.?$`), func(m []string) domain.Issue {
			return errIssue(CodeScheduleWeekRequired, fmt.Sprintf("Row %s: week is required when topic is provided", m[1]), schedulePath+": row "+m[1])
		}},
		{regexp.MustCompile(`^Row ` + rowNumRe + `: invalid week '(.*)'\. Use integers\.?$`), func(m []string) domain.Issue {
			return errIssue(CodeScheduleWeekInvalid, fmt.Sprintf("Row %s: invalid week '%s'. Use integers", m[1], m[2]), schedulePath+": row "+m[1])
		}},
		{regexp.MustCompile(`^Row ` + rowNumRe + `: week must be > 0\.?$`), func(m []string) domain.Issue {
			return errIssue(CodeScheduleWeekNonpos, fmt.Sprintf("Row %s: week must be > 0", m[1]), schedulePath+": row "+m[1])
		}},
		{regexp.MustCompile(`^Row ` + rowNumRe + `: both component and weight_percent are required\.?$`), func(m []string) domain.Issue {
			return errIssue(CodeGradingBothRequired, fmt.Sprintf("Row %s: both component and weight_percent are required", m[1]), gradingPath+": row "+m[1])
		}},
		{regexp.MustCompile(`^Row ` + rowNumRe + `: weight_percent must be numeric\.?$`), func(m []string) domain.Issue {
			return errIssue(CodeGradingWeightNumeric, fmt.Sprintf("Row %s: weight_percent must be numeric", m[1]), gradingPath+": row "+m[1])
		}},
		{regexp.MustCompile(`^Row ` + rowNumRe + `: weight_percent must be >= 0\.?$`), func(m []string) domain.Issue {
			return errIssue(CodeGradingWeightNegative, fmt.Sprintf("Row %s: weight_percent must be >= 0", m[1]), gradingPath+": row "+m[1])
		}},
		{regexp.MustCompile(`^No grading rows found in CSV\.?$`), func([]string) domain.Issue {
			return errIssue(CodeGradingEmpty, "No grading rows found in CSV", gradingPath)
		}},
		{regexp.MustCompile(`^No non-empty week/topic rows found in CSV\.?$`), func([]string) domain.Issue {
			return errIssue(CodeScheduleEmpty, "No non-empty week/topic rows found in schedule CSV", schedulePath)
		}},
		{regexp.MustCompile(`^CSV has no header row\.?$`), func([]string) domain.Issue {
			// The worker does not say which CSV; the structure error would have caught a missing file already.
			return errIssue(CodeBuildValidation, "A CSV file has no header row", "")
		}},
	}

	identityOnlyColumns = map[string]bool{
		"course_instructors": true, "course_instructors_short": true, "program_name": true,
		"school_name": true, "university_name": true, "semester_label": true,
	}
)

// missingColumnsIssue guesses which CSV a worker "Missing required columns: a, b" message is about from the column names.
func missingColumnsIssue(cols []string) domain.Issue {
	list := strings.Join(cols, ", ")
	allIn := func(set ...string) bool {
		for _, c := range cols {
			found := false
			for _, s := range set {
				if c == s {
					found = true
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
	switch {
	case allIn("week", "topic"):
		return errIssue(CodeScheduleMissingColumns, "Schedule CSV missing required columns (week, topic)", schedulePath)
	case allIn("component", "weight_percent"):
		return errIssue(CodeGradingMissingColumns, "Grading CSV missing required columns (component, weight_percent)", gradingPath)
	}
	for _, c := range cols {
		if identityOnlyColumns[c] {
			return errIssue(CodeIdentityMissingColumns, "Missing required columns in identity CSV: "+list, identityPath)
		}
	}
	return errIssue(CodeSyllabusMissingColumns, "Missing required columns in syllabus CSV: "+list, syllabusPath)
}

// classifyMessage maps one worker validation message to its SRS issue; ok is false for unknown messages.
func classifyMessage(msg string) (domain.Issue, bool) {
	msg = strings.TrimSpace(msg)
	for _, p := range messagePatterns {
		if m := p.re.FindStringSubmatch(msg); m != nil {
			return p.make(m), true
		}
	}
	return domain.Issue{}, false
}

// ParseWorkerFailure turns the error text of a failed worker job into issues (submission-pipeline.md section 5):
//  1. bullet lines below "Errors:" (until the first blank line) become one issue each, mapped to the SRS catalogue
//     when they match a known message, else BUILD_VALIDATION with the message as title;
//  2. a bare known message (e.g. the zip-root error, which the worker prints without the "Errors:" block) becomes that issue;
//  3. anything else is one BUILD_FAILED "PDF generation failed" issue carrying the last 2000 characters of the text
//     (a Python traceback, if present, is cut off).
func ParseWorkerFailure(errText string) []domain.Issue {
	text := strings.ReplaceAll(errText, "\r\n", "\n")
	body := strings.TrimSpace(jobPrefixRe.ReplaceAllString(strings.TrimSpace(text), ""))

	if i := strings.Index(body, "\nErrors:\n"); i >= 0 || strings.HasPrefix(body, "Errors:\n") {
		start := i + len("\nErrors:\n")
		if i < 0 {
			start = len("Errors:\n")
		}
		var issues []domain.Issue
		for _, line := range strings.Split(body[start:], "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				break
			}
			msg, isBullet := trimBullet(line)
			if !isBullet {
				continue
			}
			if is, ok := classifyMessage(msg); ok {
				issues = append(issues, is)
			} else {
				issues = append(issues, errIssue(CodeBuildValidation, msg, ""))
			}
		}
		if len(issues) > 0 {
			return issues
		}
	}

	firstLine := body
	if i := strings.Index(firstLine, "\n"); i >= 0 {
		firstLine = firstLine[:i]
	}
	if is, ok := classifyMessage(firstLine); ok {
		return []domain.Issue{is}
	}

	desc := body
	if i := strings.Index(desc, "Traceback (most recent call last)"); i >= 0 {
		desc = strings.TrimSpace(desc[:i])
	}
	// The worker prints its log before the real message; for a LaTeX failure the useful part starts at "latexmk failed".
	if i := strings.LastIndex(desc, "latexmk failed for"); i >= 0 {
		desc = desc[i:]
		if r := []rune(desc); len(r) > maxFailureDescription {
			desc = string(r[:maxFailureDescription])
		}
	} else if r := []rune(desc); len(r) > maxFailureDescription {
		desc = string(r[len(r)-maxFailureDescription:])
	}
	is := errIssue(CodeBuildFailed, "PDF generation failed", "")
	if desc != "" {
		is.Description = desc
	}
	return []domain.Issue{is}
}

func trimBullet(line string) (string, bool) {
	for _, b := range []string{"•", "-", "*"} {
		if strings.HasPrefix(line, b) {
			return strings.TrimSpace(strings.TrimPrefix(line, b)), true
		}
	}
	return "", false
}
