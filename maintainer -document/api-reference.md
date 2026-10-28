# API reference

Back to the [documentation index](README.md).

Machine-readable specs (OpenAPI 3.0.3, paths include the `/api/v1` prefix and match the routers):

* main-service: [`main-service/api/openapi/server/spec.yaml`](../main-service/api/openapi/server/spec.yaml)
* auth-service: [`auth-service/api/openapi/server/spec.yaml`](../auth-service/api/openapi/server/spec.yaml)

Swagger UI exists only in dev (`ENABLE_SWAGGER=true`, container path `/swagger`, not routed through the gateway).

## Conventions

* The browser path is what the SPA and external clients call through the gateway: **`/public` + auth-service path**, **`/auth` + main-service path**. The gateway strips that prefix.
* JSON, snake_case, UUID strings, RFC3339 UTC timestamps, nullable fields are `null`. Lists are `{"items":[...]}` (no pagination; sorting is client-side).
* Errors: `{"error":"<message>"}` with 400 invalid input, 401 unauthenticated, 403 forbidden, 404 not found, 409 conflict, 413 too large, 422 unprocessable (e.g. CSV with row errors), 500 `{"error":"internal error"}`, 503 when a dependency is down.
* Authentication: cookie `abet_session` (set by the Google login). Every `/auth/api/*` call without a valid session returns 401 from the gateway.
* "Any" below = any signed-in user; "super" = `super_admin`; "admin(scope)" = `admin` with the curriculum/department in scope, else 403. Full matrix: [security-and-permissions.md](security-and-permissions.md).

## auth-service (browser prefix `/public`, no authentication, allowlist of three routes)

| Method | Browser path | Service path | Role | Notes |
|---|---|---|---|---|
| GET | `/public/api/v1/auth/google/login?redirect=/x` | `/api/v1/auth/google/login` | public | 302 to Google; 503 `google oauth not configured` if no client id |
| GET | `/public/api/v1/auth/google/callback` | `/api/v1/auth/google/callback` | public | sets `abet_session`, 302 to the app or `/login?error=access_denied|login_failed` |
| POST | `/public/api/v1/auth/logout` | `/api/v1/auth/logout` | public | revokes the session, 204 always |

Only these three routes are exposed by the gateway. Every other `/public/*` path (including `healthz`, `readyz`, `metrics` and any verify path) is answered by the gateway with `404 {"error":"not found"}`.

Not reachable from outside (docker network only):

| Method | Path | Purpose |
|---|---|---|
| GET | `/internal/v1/auth/verify` | called by the gateway forward-auth (moved from `/api/v1/auth/verify`); 200 + `X-User-Id`/`X-User-Email`, or 401 |
| GET | `/healthz`, `/readyz`, `/metrics` | liveness / DB readiness / Prometheus of auth-service (the old HTML `/health` page was removed) |

## main-service (browser prefix `/auth`, session required)

### Me and config

| Method | Browser path | Service path | Role |
|---|---|---|---|
| GET | `/auth/api/v1/me` | `/api/v1/me` | Any — returns `{user, capabilities:{manage_org, manage_users, manage_courses, manage_covers}, act_as:{available, real_user}}` (`act_as` = dev-only role switcher state, see below) |
| GET | `/auth/api/v1/dev/act-as/users` | `/api/v1/dev/act-as/users` | DEV ONLY (404 unless `DEV_ACT_AS_ENABLED=true` and `APP_ENV=development`); real super_admin; `?q=&role=`; users except yourself, max 500 |
| POST | `/auth/api/v1/dev/act-as` | `/api/v1/dev/act-as` | DEV ONLY; real super_admin; `{"user_id"}` -> 204 + `abet_act_as` cookie (Path `/auth/api`); 400 own id/bad body, 404 unknown user |
| DELETE | `/auth/api/v1/dev/act-as` | `/api/v1/dev/act-as` | DEV ONLY; real super_admin; 204, clears the cookie (idempotent) |
| GET | `/auth/api/v1/config` | `/api/v1/config` | Any — `{max_upload_mb, max_cover_mb, max_import_mb, build_timeout_minutes}` from `.env` (the frontend reads its limits here) |

### Departments and curricula

| Method | Browser path | Service path | Role |
|---|---|---|---|
| GET | `/auth/api/v1/departments` | `/api/v1/departments` | Any: super/admin get all rows, instructor/staff only their own department (empty list if none) |
| POST | `/auth/api/v1/departments` | `/api/v1/departments` | super (409 duplicate code) |
| PATCH | `/auth/api/v1/departments/{id}` | `/api/v1/departments/{id}` | super |
| DELETE | `/auth/api/v1/departments/{id}` | `/api/v1/departments/{id}` | super (409 if referenced) |
| GET | `/auth/api/v1/curricula?department_id=` | `/api/v1/curricula` | Any: super/admin get all rows, instructor/staff only their own curriculum, else their department's curricula (empty if neither) |
| POST | `/auth/api/v1/curricula` | `/api/v1/curricula` | super |
| PATCH | `/auth/api/v1/curricula/{id}` | `/api/v1/curricula/{id}` | super |
| DELETE | `/auth/api/v1/curricula/{id}` | `/api/v1/curricula/{id}` | super (409 if referenced; also deletes its cover file) |

### Covers

| Method | Browser path | Service path | Role |
|---|---|---|---|
| PUT | `/auth/api/v1/curricula/{id}/cover` | `/api/v1/curricula/{id}/cover` | super or admin(scope); multipart `file` = PDF, max `MAX_COVER_MB` |
| GET | `/auth/api/v1/curricula/{id}/cover[?download=1]` | `/api/v1/curricula/{id}/cover` | super or admin(scope); inline PDF, `?download=1` = attachment; 404 if none |
| DELETE | `/auth/api/v1/curricula/{id}/cover` | `/api/v1/curricula/{id}/cover` | super or admin(scope); 204 |

### Users

| Method | Browser path | Service path | Role |
|---|---|---|---|
| GET | `/auth/api/v1/users?search=&role=&department_id=` | `/api/v1/users` | super (all) or admin (users in scope); others 403 |
| POST | `/auth/api/v1/users` | `/api/v1/users` | super (any role) or admin (instructor/staff in scope); e-mail must match `ALLOWED_EMAIL_DOMAINS` |
| PATCH | `/auth/api/v1/users/{id}` | `/api/v1/users/{id}` | as POST; e-mail is immutable; `admin_scope` only by super |
| DELETE | `/auth/api/v1/users/{id}` | `/api/v1/users/{id}` | as POST; 409 on self |

### Offerings (Courses page), access and mapping CSV

| Method | Browser path | Service path | Role |
|---|---|---|---|
| GET | `/auth/api/v1/offerings?academic_year=&department_id=&curriculum_id=&semester=&search=` | `/api/v1/offerings` | super or admin (scope-filtered); `academic_year` required (400); instructor/staff 403 |
| GET | `/auth/api/v1/offerings/years` | `/api/v1/offerings/years` | Any — `{items:[...], current}` |
| POST | `/auth/api/v1/offerings` | `/api/v1/offerings` | super or admin(target curriculum in scope); 409 if the offering exists |
| PATCH | `/auth/api/v1/offerings/{id}` | `/api/v1/offerings/{id}` | super or admin (old and new curriculum in scope) |
| DELETE | `/auth/api/v1/offerings/{id}` | `/api/v1/offerings/{id}` | super or admin(scope); removes its submission files |
| POST | `/auth/api/v1/offerings/bulk-delete` | `/api/v1/offerings/bulk-delete` | super or admin(scope for every id); `{ids:[...]}` -> `{deleted:n}` |
| GET | `/auth/api/v1/offerings/{id}/access` | `/api/v1/offerings/{id}/access` | super or admin(scope); includes default staff entries |
| PUT | `/auth/api/v1/offerings/{id}/access/{user_id}` | `/api/v1/offerings/{id}/access/{user_id}` | super or admin(scope); `{can_upload, can_download}` (at least one true; target must be instructor/staff) |
| DELETE | `/auth/api/v1/offerings/{id}/access/{user_id}` | `/api/v1/offerings/{id}/access/{user_id}` | super or admin(scope); 204 |
| POST | `/auth/api/v1/offerings/import/preview` | `/api/v1/offerings/import/preview` | super or admin(curriculum in scope); multipart `file` (CSV <= `MAX_IMPORT_MB`, default 5 MB), `curriculum_id`, `academic_year`; 200 ImportPlan, no writes |
| POST | `/auth/api/v1/offerings/import/apply` | `/api/v1/offerings/import/apply` | same; 200 `applied:true`, or 422 + plan when rows have errors |

### Portfolios and submissions

| Method | Browser path | Service path | Role |
|---|---|---|---|
| GET | `/auth/api/v1/portfolios?academic_year=&department_id=&curriculum_id=&semester=&search=` | `/api/v1/portfolios` | Any — only viewable offerings with `can_upload`/`can_download` |
| POST | `/auth/api/v1/offerings/{id}/submissions` | `/api/v1/offerings/{id}/submissions` | upload permission on the offering; multipart `file` (.zip <= `MAX_UPLOAD_MB`); 201 Submission (`invalid` / `queued` / `failed`) with issues; 400 not a zip, 413 too big |
| GET | `/auth/api/v1/offerings/{id}/submissions` | `/api/v1/offerings/{id}/submissions` | view permission; history, newest first |
| GET | `/auth/api/v1/submissions/{id}` | `/api/v1/submissions/{id}` | view permission on its offering; includes issues |
| GET | `/auth/api/v1/offerings/{id}/portfolio.pdf` | `/api/v1/offerings/{id}/portfolio.pdf` | download permission; latest generated PDF as attachment `<code>_<year>_S<sem>_portfolio.pdf`; 404 if none |
| POST | `/auth/api/v1/portfolios/download` | `/api/v1/portfolios/download` | Any, per-offering download permission; `{offering_ids:[...]}` -> zip `ABET_Course_Portfolios_<year|selected>.zip` (+ `MISSING.txt`); 400 empty list, 404 none available |

### Templates

| Method | Browser path | Service path | Role |
|---|---|---|---|
| GET | `/auth/api/v1/templates/instructor-package.zip` | `/api/v1/templates/instructor-package.zip` | Any — the empty portfolio folder template (under `course-id-course-name/`), saved as `course-template.zip` |
| GET | `/auth/api/v1/templates/course-instructor-mapping.csv` | `/api/v1/templates/course-instructor-mapping.csv` | Any — header + 2 example rows |

### Not reachable from outside

| Method | Path | Purpose |
|---|---|---|
| POST | `/internal/v1/auth/login-sync` | auth-service -> main-service after Google login; header `X-Internal-Token`; 200 `{user_id,email,role}` / 403 `not provisioned` / 401 |
| GET | `/healthz`, `/readyz`, `/metrics` | probes and Prometheus of main-service (docker network only). `/readyz` also reports `build_queue` (`consumers`, `messages`); metrics include `abet_submissions_timed_out_total`, `abet_build_queue_consumers`, `abet_build_queue_messages` |

## Main objects (see the OpenAPI schemas for all fields)

* `Submission`: `status` is `invalid|queued|processing|generated|failed` (`processing` is never set today; the UI shows `queued` as "Processing · N min"; a build that exceeds `BUILD_TIMEOUT_MINUTES` becomes `failed` with issue `BUILD_TIMEOUT`, and a late worker success turns it into `generated`); `issues` only on `POST` and `GET /submissions/{id}`.
* `Issue`: `severity`, `stage` (`missing|structure|naming|csv|build`), `code`, `title`, `description`, `location`, `suggested_fix` — codes in [business-rules.md](business-rules.md#issue-codes).
* `ImportPlan`: `applied`, `summary{...}`, `courses[]` (`create|update|keep|delete|keep_has_uploads`), `access[]` (`add|update|remove`), `users[]` (`create|raise_role`), `errors[{row,message}]`.
