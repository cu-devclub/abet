# Business rules

Back to the [documentation index](README.md). Labels like D2 / A1 are the user's decisions and the CTO's assumptions from the project plan; the rule text here is what the code does today
(`main-service/internal/application/importplan`, `internal/core/validation`, `internal/core/domain`).

## Conventions

| Topic | Rule |
|---|---|
| Semester (A1) | stored as `1`, `2`, `3` (DB check). The UI shows `3` as **"Summer"** |
| Academic year (A2) | stored as a **CE integer** (e.g. `2026`, allowed 2000..2200), not Buddhist Era. `GET /offerings/years` returns the years that exist plus the current calendar year (`current`) |
| Uniqueness | course = (curriculum, code); offering = (course, academic year, semester); access = (offering, user) |
| Codes | department/curriculum `code` is trimmed and stored UPPERCASE for departments; user e-mails lower-case. Course codes are upper-cased by the UI |
| Staff default access (A3) | a `staff` user can **download** every offering whose department equals their own department, without any access row (shown "(default)", read-only in the UI; listed in `GET /offerings/{id}/access` with `is_default: true`). Instructors have **no** default: only explicit assignments. Admins: everything in scope. Super admin: everything |
| Single role | one global role per user, highest applies |
| Cover | exactly one PDF per curriculum, max 20 MB; used as page 1 of every portfolio built for that curriculum. It is a full-page **background**: the portfolio service prints the course code (48 pt), course name (36 pt) and semester in black, centred about 6.5 cm above the page centre — keep that area light and empty. It always wins over a course's own `cover_image` in `00-meta/instructor_course_identity_template.csv` (D27). Without a curriculum cover the built-in default is used. |
| Submission history | every upload attempt is a row (`invalid`, `queued`, `generated`, `failed`). The downloadable PDF is the latest `generated` one; older generated rows keep their row but lose their file |

## Mapping CSV import

Endpoint pair: `POST /offerings/import/preview` (no writes) and `/apply` (one DB transaction). The Courses page supplies the **scope** with its filters:
**curriculum + academic year** (D2). The CSV has **no year column**. The file is processed once and **never stored** (D3).

### File format

Header (case-insensitive; spaces, underscores and hyphens are interchangeable; aliases `course code`, `email`, `role`, `upload`, `download` also work):

```
course id, course name, semester, user email, user role, upload permission, download permission
```

| Column | Rule |
|---|---|
| course id | required, trimmed |
| course name | required |
| semester | `1`, `2` or `3` |
| user email | optional (empty = course-only row); must be in `ALLOWED_EMAIL_DOMAINS`; lower-cased |
| user role | `Instructor` or `Staff` (case-insensitive); required when an e-mail is present |
| upload / download permission | `Yes/No`, `True/False`, `Y/N`, `1/0`; required (non-empty) when an e-mail is present |

Parsing rules: UTF-8, BOM allowed, CRLF allowed, quoted fields allowed, maximum `MAX_IMPORT_MB` (default 5 MB). **Row numbers** in error messages are physical CSV lines (header = row 1, blank lines are skipped but still counted), so they match what a spreadsheet shows.
A file with no data rows is an error ("the file has no data rows") because otherwise it would plan to delete every offering without uploads. A role or permission without an e-mail is an error. If the file has any row error, **preview returns 200 with the error list and the plan of the valid rows; apply returns 422 and changes nothing**.
A missing `course id`, `course name` or `semester` column is an error at row 1; the four user columns may be absent entirely (then every row is a course-only row) and unknown extra columns are ignored.

### Algorithm (D4, D8, D9, D10)

Scope = (selected curriculum, selected academic year). Let *file offerings* = distinct (course id, semester) in the file.

1. **Offerings in the file**
   * not in the DB -> **create** (the course is created if the (curriculum, code) pair is new);
   * in the DB with the same name -> **keep**; with a different name -> **update** (the course name is renamed once; if the same course id appears with different names the **last row wins**).
2. **Offerings in scope but not in the file**
   * has **ever had an upload attempt** (any submission row, even `invalid` or `failed`, D8) -> **keep_has_uploads**: the offering stays and **all its instructor/staff access rows are removed** (D9);
   * never uploaded -> **delete** (with its access rows). Nothing is "fully synced": courses of other curricula or years are never touched (D3/D4).
3. **Access** for every offering in the file is **mirrored** from the CSV (replace the set): rows added / changed / removed, `source='csv'`. This also replaces manually created rows of those offerings. A row with both permissions `No` grants nothing (it is ignored; the course is still kept/created). Duplicate rows for the same (course, semester, e-mail): **last wins** (a later both-No row cancels an earlier grant).
4. **Users**
   * e-mail not in the system -> **create** with the CSV role, department = the curriculum's department, curriculum = the selected curriculum (D10). A user who appears only on both-No rows is **not** created (conservative reading);
   * existing `staff` and CSV role `Instructor` -> **raise_role** to instructor. Roles are **never lowered** and `admin`/`super_admin` are never changed (their access row is still written);
   * a user on several rows with different roles gets the highest CSV role.
5. Everything runs in one transaction; `apply` recomputes the plan itself and never trusts the preview. Objects of deleted offerings are removed from MinIO after the commit.

Authorization: super admin, or admin with the **curriculum in scope**; otherwise 403.

### Worked example

Scope: curriculum `CS-B`, academic year `2026`. Database before:

| Offering | Name in DB | Uploads? | Access rows |
|---|---|---|---|
| 2301101 S1 | Intro to Programming | no | instructor1 (U+D, manual), staff1 (D, manual) |
| 2301102 S1 | Data Structures | **yes** | instructor1 (U only) |
| 2301103 S1 | Operating Systems | no | instructor2 (U+D) |

CSV:

```csv
course id,course name,semester,user email,user role,upload permission,download permission
2301101,Introduction to Programming,1,instructor1@chula.ac.th,Instructor,Yes,Yes
2301101,Introduction to Programming,1,newstaff@chula.ac.th,Staff,No,Yes
2301105,Databases,1,instructor1@chula.ac.th,Instructor,Yes,No
2301105,Databases,1,staff1@chula.ac.th,Staff,No,No
```

Resulting plan:

| Item | Action | Why |
|---|---|---|
| 2301101 S1 | **update** (name -> "Introduction to Programming") | in file and DB, name differs |
| 2301101: instructor1 U+D | nothing | identical to the existing row |
| 2301101: staff1 | **access remove** | access is mirrored; staff1 is not on a row of this offering |
| 2301101: newstaff D | **access add** | new grant |
| `newstaff@chula.ac.th` | **user create** (staff, dept CS, curriculum CS-B) | unknown e-mail (D10) |
| 2301105 S1 "Databases" | **create** | not in DB |
| 2301105: instructor1 U only | **access add** | |
| 2301105: staff1 (No/No) | nothing | both-No grants nothing; staff1 still has the department default download as staff |
| 2301102 S1 | **keep_has_uploads**; instructor1 **access remove** | not in file but it has uploads (D8); access cleared (D9) |
| 2301103 S1 | **delete** | not in file, never uploaded (D4); its access rows disappear with it |

Summary object: `rows 4, courses_create 1, courses_update 1, courses_keep 0, courses_delete 1, courses_kept_with_uploads 1, access_add 2, access_update 0, access_remove 2, users_create 1, users_role_raised 0`.
Applying the same file a second time is idempotent (every offering is then `keep`, no changes).

## Upload validation

`POST /offerings/{id}/submissions` validates the zip **immediately** (before queueing) with a pure Go port of the worker's rules, using the SRS section 7 titles. Result: errors -> status `invalid` and **nothing is queued**; warnings only -> `queued`.

### What is checked

* **Zip root**: junk is ignored (`__MACOSX`, `.DS_Store`, `._*`); up to 5 levels of single-folder wrappers are collapsed (a zip containing `portfolio/course/...` works). One top folder = the course folder (sibling files at the root are ignored). Several top folders are accepted only when they are all among the 9 required folders (and `99-abet`); otherwise `ZIP_ROOT` "must contain exactly one top-level folder. Found N."
* **Required folders (9)**: `00-meta`, `01-syllabus`, `02-textbook`, `03-schedule`, `10-lectures`, `20-assignments`, `30-quizzes`, `40-exams`, `90-grades` (missing = error).
* **Required files (4 CSVs)**: `00-meta/instructor_course_identity_template.csv`, `01-syllabus/instructor_syllabus_abet_reference.csv`, `03-schedule/instructor_course_schedule_template.csv`, `90-grades/instructor_course_grading_template.csv` (underscores, as the real worker contract, A5 — not the spaces printed in the SRS).
* **Allowed names per folder** (anything else is `UNEXPECTED_FILE_NAME`): `00-meta`: `course-meta.tex`, identity CSV · `01-syllabus`: syllabus CSV, `syllabus-abet.tex`, `syllabus-instructor.pdf` · `02-textbook`: `textbook-partNN.pdf` · `03-schedule`: `course-schedule.tex`, schedule CSV, `lecture_schedule_mapping.csv` · `10-lectures`: `weekNN-partNN.pdf` · `20-assignments`: `asg-all-scores.pdf`, `asgNN-{instructions,high,average,low,scores}.pdf` · `30-quizzes`: `quiz-all-scores.pdf`, `quizNN-{…}.pdf` · `40-exams`: `{midterm,final}-{instructions,high,average,low,scores}.pdf` · `90-grades`: `course-grading.tex`, grading CSV, `overall-grades.pdf`. NN = two digits.
* **Placement**: no stray top-level files/folders, no sub-folders inside category folders, nothing nested deeper than 2 levels.
* **Content**: a `.pdf` must contain `%PDF-` within its first 1024 bytes; `.csv .txt .md .tex` must be valid UTF-8 (BOM ok) without NUL bytes.
* **CSVs** (encoding/csv, lazy quotes, trimmed, BOM stripped, the worker's column names): identity CSV exactly one populated row with the required columns and the `course_code` must equal the offering's course code (`IDENTITY_COURSE_MISMATCH`); syllabus: header + SO1..SO7 columns, not template-only; schedule: `week` positive integers, at least one entry; grading: `component` + `weight_percent` numeric >= 0, at least one component. Row numbers = the worker's (header is row 1).
* **Warnings** (never block): `asgNN`/`quizNN` missing any of instructions/high/average/low (e.g. "asg02 is missing asg02-low.pdf"), `midterm`/`final` partially present, `syllabus-instructor.pdf` or `overall-grades.pdf` missing, no lecture files.
* **Limits**: more than 2000 entries or more than 1 GB uncompressed -> `ZIP_TOO_LARGE`.

### Issue codes

Every issue has `severity` (`error`|`warning`), `stage`, `code`, `title` (exact SRS text with placeholders filled), `description` (SRS "Cause / Explanation"), `location` (path or `row N`) and `suggested_fix` (SRS "User Action Required"). All texts are in `main-service/internal/core/validation/messages.go`.

| Stage | Code | Title (shape) | Severity |
|---|---|---|---|
| missing | `MISSING_FOLDER` | Missing required folder: {subdir} | error |
| missing | `MISSING_FILE` | Missing required file: {relpath} | error |
| missing | `INCOMPLETE_DOCUMENT` | Incomplete document: {what} | warning |
| structure | `UNEXPECTED_TOP_LEVEL_FOLDER` | Unexpected top-level folder: {relpath} | error |
| structure | `UNEXPECTED_NESTED_FOLDER` | Unexpected nested folder: {relpath} | error |
| structure | `UNEXPECTED_TOP_LEVEL_FILE` | Unexpected top-level file: {relpath} | error |
| structure | `UNEXPECTED_FILE_OUTSIDE` | Unexpected file outside required folders: {relpath} | error |
| structure | `UNEXPECTED_NESTED_FILE` | Unexpected nested file path: {relpath} | error |
| structure | `ZIP_ROOT` | Submission zip must contain exactly one top-level folder. Found {n}. | error |
| structure | `ZIP_TOO_LARGE`, `ZIP_INVALID` | too many files / not a zip | error |
| naming | `UNEXPECTED_FILE_NAME` | Unexpected file name in {category}: {relpath} | error |
| naming | `INVALID_PDF` | Invalid PDF content for file with .pdf extension: {relpath} | error |
| naming | `INVALID_TEXT` | Invalid text content for file with {suffix} extension: {relpath} | error |
| csv | `IDENTITY_NO_HEADER`, `IDENTITY_ROW_COUNT`, `IDENTITY_CODE_EMPTY`, `IDENTITY_MISSING_COLUMNS`, `IDENTITY_COURSE_MISMATCH`, `ROW_MISSING_VALUES` | identity CSV problems ("Row {n} missing required values: {columns}") | error |
| csv | `SYLLABUS_NO_HEADER`, `SYLLABUS_MISSING_COLUMNS`, `SYLLABUS_TEMPLATE_ONLY` | syllabus CSV problems | error |
| csv | `SCHEDULE_NO_HEADER`, `SCHEDULE_MISSING_COLUMNS`, `SCHEDULE_WEEK_REQUIRED`, `SCHEDULE_WEEK_INVALID`, `SCHEDULE_WEEK_NONPOSITIVE`, `SCHEDULE_EMPTY` | schedule CSV problems ("Row {n}: invalid week '{raw}'. Use integers") | error |
| csv | `GRADING_NO_HEADER`, `GRADING_MISSING_COLUMNS`, `GRADING_BOTH_REQUIRED`, `GRADING_WEIGHT_NUMERIC`, `GRADING_WEIGHT_NEGATIVE`, `GRADING_EMPTY` | grading CSV problems | error |
| build | `BUILD_VALIDATION` | a bullet of the worker's own validation report | error |
| build | `BUILD_FAILED` | PDF generation failed (description = tail of the worker error) | error |
| build | `BUILD_NO_OUTPUT` | worker finished without a PDF | error |
| build | `BUILD_TIMEOUT` | the build did not finish within `BUILD_TIMEOUT_MINUTES` (set by the reconciler; removed again if the worker finishes late and the submission becomes `generated`) | error |
| build | `QUEUE_UNAVAILABLE` | Portfolio build service unavailable. Please try again later. | error |

The first four stages come from the immediate validator; `build` issues come from the worker result (`ParseWorkerFailure` reads the bullets under `Errors:` and maps known titles back to their codes).

### Validator versus worker

The goal is "anything we accept, the worker also accepts". The validator is a port of the worker's `runtime_pipeline.py` and CSV generator scripts, plus the SRS. Where they differ we are deliberately **stricter or SRS-driven** (T05 deviations):

| # | Difference | Reason |
|---|---|---|
| 1 | Missing "artifact-only" folders (`02-textbook`, `10-lectures`, `20-assignments`, `30-quizzes`, `40-exams`) are **errors**; unknown files are **errors**; an **empty schedule is an error** (`SCHEDULE_EMPTY`). The worker would auto-create folders, auto-repair unknown files and accept an empty schedule | SRS section 7 lists them as errors. Consequence: the unfilled instructor-package template (empty schedule topics) is rejected on purpose until it is filled in (CTO decision: `SCHEDULE_EMPTY` stays an error) |
| 2 | The PDF check is `%PDF-` **within the first 1024 bytes** (the worker's `is_valid_pdf`), not "first 5 bytes" | otherwise we would reject files the worker accepts |
| 3 | `Thumbs.db` / `desktop.ini` are **not** junk; only `__MACOSX`, `.DS_Store`, `._*` are ignored | the worker does not strip the others and would reject the package (`Unexpected top-level file`) |
| 4 | Root detection copies the worker: a single top folder is the course; several top folders only if all are among the 9 required (+`99-abet`). "An extra folder next to the 9 folders at the zip root" is therefore `ZIP_ROOT ... Found N`, not `UNEXPECTED_TOP_LEVEL_FOLDER`. Wrapper collapsing: top folder + up to 4 more single-folder levels | parity with `find_wrapped_course_layout` |
| 5 | Extra codes `IDENTITY_NO_HEADER`, `ZIP_INVALID`; an empty `course_code` is reported as `ROW_MISSING_VALUES ... course_code` (as the worker does); `IDENTITY_CODE_EMPTY` exists for mapping worker messages; "Missing required columns" titles carry the column list | clearer messages, same decisions |
| 6 | Row numbers equal the worker's (`enumerate(reader, start=2)`: header is row 1, blank lines not counted) | users see the same numbers in both places |

A package can pass the validator and still fail in the worker (e.g. a PDF that only contains `%PDF-1.4` text, LaTeX errors): that surfaces as a `failed` submission with `BUILD_FAILED`.

## Storage limit (D11)

SRS: 10 MB per week x 15 weeks = **150 MB** per portfolio. The system allows **150 MB x 2 margin = 300 MB** (`MAX_UPLOAD_MB=300`, gateway body limit 320 MB = the largest of the three limits + 20 MB, computed from `.env`; the client pre-check reads `GET /api/v1/config`). A **new upload replaces the old result**: when the new build succeeds the PDFs of older generated submissions of that offering are deleted (their history rows remain); the uploaded zip itself is deleted after the build, or immediately when the upload is invalid or fails.
