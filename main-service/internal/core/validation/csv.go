package validation

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

var identityColumns = []string{
	"course_name", "course_code", "course_instructors", "course_instructors_short",
	"program_name", "school_name", "university_name", "semester_label",
}

var syllabusColumns = []string{
	"course_code", "course_name", "required_or_elective", "catalog_description", "prerequisites",
	"categorization_of_credits", "instructors", "class_laboratory_schedule", "prepared_by", "prepared_date",
	"textbooks_and_other_required_material", "course_learning_outcomes_expected_performance_criteria", "topics_covered",
	"ice_outcome_1", "ice_outcome_2", "ice_outcome_3", "ice_outcome_4", "ice_outcome_5", "ice_outcome_6", "ice_outcome_7",
}

// syllabusSentinels are the placeholder values of the template row (generate_syllabus_abet_from_csv.TEMPLATE_SENTINELS).
var syllabusSentinels = map[string]bool{
	"your_course_code": true, "your_course_name": true, "required_elective": true, "your_catalog_description": true,
}

// table is a parsed CSV in the shape of Python's csv.DictReader.
type table struct {
	header []string
	rows   []tableRow
}

type tableRow struct {
	n      int // 1-based, header = row 1, blank lines not counted (enumerate(reader, start=2))
	values map[string]string
}

func (r tableRow) get(col string) string { return strings.TrimSpace(r.values[col]) }

func (r tableRow) empty() bool {
	for _, v := range r.values {
		if strings.TrimSpace(v) != "" {
			return false
		}
	}
	return true
}

// parseTable parses data like csv.DictReader after opening the file with encoding "utf-8-sig".
// ok is false when the file has no header row.
func parseTable(data []byte) (t table, ok bool) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	header, err := r.Read()
	if err != nil {
		return t, false
	}
	t.header = header
	n := 2
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			// A malformed record cannot be recovered; stop here (rows read so far are still checked).
			break
		}
		vals := make(map[string]string, len(header))
		for i, h := range header {
			if i < len(rec) {
				vals[h] = rec[i]
			}
		}
		t.rows = append(t.rows, tableRow{n: n, values: vals})
		n++
	}
	return t, true
}

func (t table) missing(required []string) []string {
	have := map[string]bool{}
	for _, h := range t.header {
		have[h] = true
	}
	var out []string
	for _, c := range required {
		if !have[c] {
			out = append(out, c)
		}
	}
	return out
}

// checkCSVs runs the four CSV content checks on the files that exist and are valid UTF-8 text.
func (v *validator) checkCSVs(course *node) {
	load := func(rel string) (table, bool, bool) { // table, fileUsable, hasHeader
		f := course.fileAt(rel)
		if f == nil || v.badText[rel] {
			return table{}, false, false
		}
		data, ok := v.readCSVBytes(f)
		if !ok {
			return table{}, false, false
		}
		t, hasHeader := parseTable(data)
		return t, true, hasHeader
	}

	if t, usable, hasHeader := load(identityPath); usable {
		v.checkIdentity(t, hasHeader)
	}
	if t, usable, hasHeader := load(syllabusPath); usable {
		v.checkSyllabus(t, hasHeader)
	}
	if t, usable, hasHeader := load(schedulePath); usable {
		v.checkSchedule(t, hasHeader)
	}
	if t, usable, hasHeader := load(gradingPath); usable {
		v.checkGrading(t, hasHeader)
	}
}

// checkIdentity ports generate_course_meta_from_csv.load_rows + the course-code logic of validate_submission.
func (v *validator) checkIdentity(t table, hasHeader bool) {
	if !hasHeader {
		v.addErr(CodeIdentityNoHeader, "Identity CSV has no header row", identityPath)
		return
	}
	if miss := t.missing(identityColumns); len(miss) > 0 {
		v.addErr(CodeIdentityMissingColumns, "Missing required columns in identity CSV: "+strings.Join(miss, ", "), identityPath)
		return
	}
	var populated []tableRow
	badRow := false
	for _, row := range t.rows {
		if row.empty() {
			continue
		}
		var missing []string
		for _, c := range identityColumns {
			if row.get(c) == "" {
				missing = append(missing, c)
			}
		}
		if len(missing) > 0 {
			badRow = true
			v.addErr(CodeRowMissingValues, titleRowMissing(row.n, missing), fmt.Sprintf("%s: row %d", identityPath, row.n))
			continue
		}
		populated = append(populated, row)
	}
	if !badRow && len(populated) != 1 {
		v.addErr(CodeIdentityRowCount, "Identity CSV must contain exactly one populated row", identityPath)
		return
	}
	if len(populated) == 1 && v.expectedCode != "" {
		code := populated[0].get("course_code")
		if !strings.EqualFold(code, strings.TrimSpace(v.expectedCode)) {
			v.addErr(CodeIdentityCourseMismatch,
				fmt.Sprintf("Identity CSV course code %s does not match course %s", code, strings.TrimSpace(v.expectedCode)),
				identityPath+": row "+strconv.Itoa(populated[0].n))
		}
	}
}

// checkSyllabus ports generate_syllabus_abet_from_csv.load_rows.
func (v *validator) checkSyllabus(t table, hasHeader bool) {
	if !hasHeader {
		v.addErr(CodeSyllabusNoHeader, "Syllabus CSV has no header row", syllabusPath)
		return
	}
	if miss := sortedCopy(t.missing(syllabusColumns)); len(miss) > 0 {
		v.addErr(CodeSyllabusMissingColumns, "Missing required columns in syllabus CSV: "+strings.Join(miss, ", "), syllabusPath)
		return
	}
	for _, row := range t.rows {
		if row.empty() {
			continue
		}
		if syllabusSentinels[row.get("course_code")] || syllabusSentinels[row.get("course_name")] {
			continue
		}
		return // at least one real row
	}
	v.addErr(CodeSyllabusTemplateOnly, "No non-template syllabus rows found in CSV", syllabusPath)
}

// checkSchedule ports generate_course_schedule_from_csv.parse_rows (+ the "empty" rule of its CLI, per SRS 7.4).
func (v *validator) checkSchedule(t table, hasHeader bool) {
	if !hasHeader {
		v.addErr(CodeScheduleNoHeader, "Schedule CSV has no header row", schedulePath)
		return
	}
	if miss := sortedCopy(t.missing([]string{"week", "topic"})); len(miss) > 0 {
		v.addErr(CodeScheduleMissingColumns, "Schedule CSV missing required columns (week, topic)", schedulePath)
		return
	}
	valid, bad := 0, false
	for _, row := range t.rows {
		weekRaw, topic := row.get("week"), row.get("topic")
		if weekRaw == "" && topic == "" {
			continue
		}
		loc := fmt.Sprintf("%s: row %d", schedulePath, row.n)
		if weekRaw == "" {
			bad = true
			v.addErr(CodeScheduleWeekRequired, fmt.Sprintf("Row %d: week is required when topic is provided", row.n), loc)
			continue
		}
		week, err := strconv.Atoi(weekRaw)
		if err != nil {
			bad = true
			v.addErr(CodeScheduleWeekInvalid, fmt.Sprintf("Row %d: invalid week '%s'. Use integers", row.n, weekRaw), loc)
			continue
		}
		if week <= 0 {
			bad = true
			v.addErr(CodeScheduleWeekNonpos, fmt.Sprintf("Row %d: week must be > 0", row.n), loc)
			continue
		}
		if topic != "" {
			valid++
		}
	}
	if valid == 0 && !bad {
		v.addErr(CodeScheduleEmpty, "No non-empty week/topic rows found in schedule CSV", schedulePath)
	}
}

// checkGrading ports generate_course_grading_from_csv.parse_rows.
func (v *validator) checkGrading(t table, hasHeader bool) {
	if !hasHeader {
		v.addErr(CodeGradingNoHeader, "Grading CSV has no header row", gradingPath)
		return
	}
	if miss := sortedCopy(t.missing([]string{"component", "weight_percent"})); len(miss) > 0 {
		v.addErr(CodeGradingMissingColumns, "Grading CSV missing required columns (component, weight_percent)", gradingPath)
		return
	}
	valid, bad := 0, false
	for _, row := range t.rows {
		comp, pct := row.get("component"), row.get("weight_percent")
		if comp == "" && pct == "" {
			continue
		}
		loc := fmt.Sprintf("%s: row %d", gradingPath, row.n)
		if comp == "" || pct == "" {
			bad = true
			v.addErr(CodeGradingBothRequired, fmt.Sprintf("Row %d: both component and weight_percent are required", row.n), loc)
			continue
		}
		f, err := strconv.ParseFloat(pct, 64)
		if err != nil {
			bad = true
			v.addErr(CodeGradingWeightNumeric, fmt.Sprintf("Row %d: weight_percent must be numeric", row.n), loc)
			continue
		}
		if f < 0 {
			bad = true
			v.addErr(CodeGradingWeightNegative, fmt.Sprintf("Row %d: weight_percent must be >= 0", row.n), loc)
			continue
		}
		valid++
	}
	if valid == 0 && !bad {
		v.addErr(CodeGradingEmpty, "No grading rows found in CSV", gradingPath)
	}
}

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}
