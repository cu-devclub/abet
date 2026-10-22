package validation

import (
	"fmt"
	"regexp"
	"sort"
)

// Per-item documents expected for an assignment, quiz, midterm or final (A9: reported as warnings only).
var itemKinds = []string{"instructions", "high", "average", "low"}

var (
	numberedItemRe = regexp.MustCompile(`^(asg\d{2}|quiz\d{2})-(average|high|instructions|low|scores)\.pdf$`)
	examItemRe     = regexp.MustCompile(`^(midterm|final)-(average|high|instructions|low|scores)\.pdf$`)
	lectureRe      = regexp.MustCompile(`^week\d{2}-part\d{2}\.pdf$`)
)

// checkCompleteness adds the INCOMPLETE_DOCUMENT warnings: partially present assignments/quizzes/exams and
// the optional documents every portfolio is expected to carry. Nothing here blocks a build.
func (v *validator) checkCompleteness(course *node) {
	for _, dir := range []string{"20-assignments", "30-quizzes"} {
		v.checkItems(course, dir, numberedItemRe)
	}
	v.checkItems(course, "40-exams", examItemRe)

	if course.hasDir("01-syllabus") && course.fileAt("01-syllabus/syllabus-instructor.pdf") == nil {
		v.addWarn(CodeIncompleteDocument, titleIncomplete("syllabus-instructor.pdf is missing"), "01-syllabus/syllabus-instructor.pdf")
	}
	if course.hasDir("90-grades") && course.fileAt("90-grades/overall-grades.pdf") == nil {
		v.addWarn(CodeIncompleteDocument, titleIncomplete("overall-grades.pdf is missing"), "90-grades/overall-grades.pdf")
	}
	if course.hasDir("10-lectures") {
		found := false
		for name, c := range course.children["10-lectures"].children {
			if !c.dir && lectureRe.MatchString(name) {
				found = true
				break
			}
		}
		if !found {
			v.addWarn(CodeIncompleteDocument, titleIncomplete("no lecture files found in 10-lectures"), "10-lectures")
		}
	}
}

// checkItems groups files like asg02-*.pdf and warns about every missing instructions/high/average/low part.
func (v *validator) checkItems(course *node, dir string, re *regexp.Regexp) {
	d, ok := course.children[dir]
	if !ok || !d.dir {
		return
	}
	present := map[string]map[string]bool{} // item -> kind -> present
	for name, c := range d.children {
		if c.dir {
			continue
		}
		m := re.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		if present[m[1]] == nil {
			present[m[1]] = map[string]bool{}
		}
		present[m[1]][m[2]] = true
	}
	items := make([]string, 0, len(present))
	for item := range present {
		items = append(items, item)
	}
	sort.Strings(items)
	for _, item := range items {
		for _, kind := range itemKinds {
			if !present[item][kind] {
				file := fmt.Sprintf("%s-%s.pdf", item, kind)
				v.addWarn(CodeIncompleteDocument, titleIncomplete(fmt.Sprintf("%s is missing %s", item, file)), dir+"/"+file)
			}
		}
	}
}
