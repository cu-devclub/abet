package importplan

import (
	"sort"

	"abet/main-service/internal/core/domain"
)

type offeringKey struct {
	code     string
	semester int
}

// BuildImportPlan computes what the import would do (see api.md "Mapping CSV import", D4/D8/D9/D10).
// rows must already be validated (ParseCSV); state describes the current offerings of the
// (curriculum, year) scope and the users mentioned in rows. The function is pure and deterministic.
func BuildImportPlan(rows []domain.ImportRow, state domain.ImportState) domain.ImportPlan {
	plan := domain.ImportPlan{
		Courses: []domain.ImportCoursePlan{},
		Access:  []domain.ImportAccessPlan{},
		Users:   []domain.ImportUserPlan{},
		Errors:  []domain.ImportError{},
	}
	plan.Summary.Rows = len(rows)

	// --- 1. courses in the file (first-appearance order); duplicate rows: last wins (also for the name).
	var fileKeys []offeringKey
	inFile := map[offeringKey]bool{}
	nameByCode := map[string]string{}
	desired := map[offeringKey]map[string]domain.ImportRow{} // key -> email -> last row
	emailOrder := map[offeringKey][]string{}
	for _, r := range rows {
		k := offeringKey{r.CourseCode, r.Semester}
		if !inFile[k] {
			inFile[k] = true
			fileKeys = append(fileKeys, k)
			desired[k] = map[string]domain.ImportRow{}
		}
		nameByCode[r.CourseCode] = r.CourseName
		if r.Email == "" {
			continue
		}
		if _, seen := desired[k][r.Email]; !seen {
			emailOrder[k] = append(emailOrder[k], r.Email)
		}
		desired[k][r.Email] = r
	}

	current := map[offeringKey]domain.ImportOfferingState{}
	for _, o := range state.Offerings {
		current[offeringKey{o.CourseCode, o.Semester}] = o
	}

	// --- 2. users: highest CSV role over the rows that actually grant something (both-No rows are ignored).
	csvRole := map[string]domain.Role{}
	var userOrder []string
	for _, k := range fileKeys {
		for _, email := range emailOrder[k] {
			r := desired[k][email]
			if !r.CanUpload && !r.CanDownload {
				continue
			}
			if _, seen := csvRole[email]; !seen {
				userOrder = append(userOrder, email)
			}
			if r.Role.Rank() > csvRole[email].Rank() {
				csvRole[email] = r.Role
			}
		}
	}
	effectiveRole := map[string]domain.Role{} // role of the user once the plan is applied
	for email, u := range state.Users {
		effectiveRole[email] = u.Role
	}
	for _, email := range userOrder {
		existing, ok := state.Users[email]
		switch {
		case !ok:
			plan.Users = append(plan.Users, domain.ImportUserPlan{Action: domain.UserCreate, Email: email, Role: csvRole[email]})
			effectiveRole[email] = csvRole[email]
			plan.Summary.UsersCreate++
		case existing.Role == domain.RoleStaff && csvRole[email] == domain.RoleInstructor:
			plan.Users = append(plan.Users, domain.ImportUserPlan{Action: domain.UserRaiseRole, Email: email, Role: domain.RoleInstructor})
			effectiveRole[email] = domain.RoleInstructor
			plan.Summary.UsersRoleRaised++
		}
	}

	// --- 3. offerings in the file: create / update / keep + access mirror.
	for _, k := range fileKeys {
		name := nameByCode[k.code]
		cur, exists := current[k]
		action := domain.CourseCreate
		if exists {
			action = domain.CourseKeep
			if cur.CourseName != name {
				action = domain.CourseUpdate
			}
		}
		plan.Courses = append(plan.Courses, domain.ImportCoursePlan{Action: action, CourseCode: k.code, CourseName: name, Semester: k.semester})
		switch action {
		case domain.CourseCreate:
			plan.Summary.CoursesCreate++
		case domain.CourseUpdate:
			plan.Summary.CoursesUpdate++
		default:
			plan.Summary.CoursesKeep++
		}

		curAccess := map[string]domain.ImportAccessState{}
		for _, a := range cur.Access {
			curAccess[a.Email] = a
		}
		want := map[string]domain.ImportRow{}
		var wantOrder []string
		for _, email := range emailOrder[k] {
			r := desired[k][email]
			if r.CanUpload || r.CanDownload {
				want[email] = r
				wantOrder = append(wantOrder, email)
			}
		}
		for _, email := range wantOrder {
			r := want[email]
			item := domain.ImportAccessPlan{CourseCode: k.code, Semester: k.semester, Email: email, Role: effectiveRole[email], CanUpload: r.CanUpload, CanDownload: r.CanDownload}
			old, has := curAccess[email]
			switch {
			case !has:
				item.Action = domain.AccessAdd
				plan.Summary.AccessAdd++
			case old.CanUpload != r.CanUpload || old.CanDownload != r.CanDownload:
				item.Action = domain.AccessUpdate
				plan.Summary.AccessUpdate++
			default:
				continue
			}
			plan.Access = append(plan.Access, item)
		}
		for _, a := range sortedAccess(cur.Access) {
			if _, keep := want[a.Email]; keep {
				continue
			}
			plan.Access = append(plan.Access, removeItem(k, a))
			plan.Summary.AccessRemove++
		}
	}

	// --- 4. offerings in scope but not in the file: keep (has uploads, D8) with access cleared (D9), or delete (D4).
	var notInFile []domain.ImportOfferingState
	for _, o := range state.Offerings {
		if !inFile[offeringKey{o.CourseCode, o.Semester}] {
			notInFile = append(notInFile, o)
		}
	}
	sort.SliceStable(notInFile, func(i, j int) bool {
		if notInFile[i].CourseCode != notInFile[j].CourseCode {
			return notInFile[i].CourseCode < notInFile[j].CourseCode
		}
		return notInFile[i].Semester < notInFile[j].Semester
	})
	for _, o := range notInFile {
		k := offeringKey{o.CourseCode, o.Semester}
		item := domain.ImportCoursePlan{CourseCode: o.CourseCode, CourseName: o.CourseName, Semester: o.Semester}
		if o.HasSubmissions {
			item.Action = domain.CourseKeepHasUpload
			item.Note = "has uploads: kept, access cleared"
			plan.Summary.CoursesKeptWithUploads++
			for _, a := range sortedAccess(o.Access) {
				plan.Access = append(plan.Access, removeItem(k, a))
				plan.Summary.AccessRemove++
			}
		} else {
			item.Action = domain.CourseDelete
			item.Note = "not in file and never uploaded"
			plan.Summary.CoursesDelete++
		}
		plan.Courses = append(plan.Courses, item)
	}
	return plan
}

func removeItem(k offeringKey, a domain.ImportAccessState) domain.ImportAccessPlan {
	return domain.ImportAccessPlan{
		Action: domain.AccessRemove, CourseCode: k.code, Semester: k.semester,
		Email: a.Email, Role: a.Role, CanUpload: a.CanUpload, CanDownload: a.CanDownload,
	}
}

func sortedAccess(in []domain.ImportAccessState) []domain.ImportAccessState {
	out := append([]domain.ImportAccessState(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Email < out[j].Email })
	return out
}
