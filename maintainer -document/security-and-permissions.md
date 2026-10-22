# Security and permissions

Back to the [documentation index](README.md). Authorization is **always enforced in main-service**; the UI only hides buttons.
The rules below were checked against [`main-service/internal/core/domain/policy.go`](../main-service/internal/core/domain/policy.go) and its table tests (`policy_test.go`).

## Roles and scope

* A user has **one global role**: `super_admin` > `admin` > `instructor` > `staff` (the highest applies when an account would hold several).
* The role and the admin scope are loaded from `main_db` **on every request**, so a role change or a removed access takes effect immediately (no re-login).
* **Admin scope** = `admin_scopes` rows, each either a department or a curriculum. `ScopeCurricula` = the explicit curricula plus every curriculum of a scoped department. Only a super admin can write scopes. An admin with an empty scope is valid and can manage nothing.
* A **user is "in scope"** of an admin when: the user's curriculum is in `ScopeCurricula`, or the user's department is a scoped department, or the user has no curriculum and their department owns a curriculum that is in `ScopeCurricula`.
* **Offering access** (`course_access`): one row per (offering, user) with `can_upload` / `can_download` (at least one true), `source` = `manual` or `csv`.

## Authorization matrix

Per course offering (course in curriculum *c*, department *d*):

| Role | View | Upload | Download | Manage (edit offering/access) |
|---|---|---|---|---|
| super_admin | yes | yes | yes | yes |
| admin, *c* in scope | yes | yes | yes | yes |
| admin, *c* not in scope | only through an explicit access row (acts like an instructor) | `can_upload` | `can_download` | no |
| instructor | only with an access row | `can_upload` | `can_download` | no |
| staff | with an access row **or** *d* = own department | `can_upload` | `can_download` **or** *d* = own department (the *default*, shown "(default)") | no |

View = upload or download (or admin/super in scope). `GET /portfolios` returns only viewable offerings, with per-row `can_upload` / `can_download`.
A staff user's default download needs no row; an explicit row (e.g. upload) can be added on top and the default still applies.

Management endpoints:

| Area | super_admin | admin | instructor / staff |
|---|---|---|---|
| Departments, curricula: read | all rows | all rows | **only their own**: departments = their department (empty if none); curricula = their curriculum if set, else the curricula of their department (empty if none) |
| Departments, curricula: create/edit/delete | yes | no (403) | no |
| Users: list | all | only users in scope (admins in the list are read-only) | 403 |
| Users: create/edit/delete | any role | only `instructor`/`staff` users who are in scope **before and after** the change; cannot assign `admin`/`super_admin`; cannot touch `admin_scopes` | 403 |
| Users: own account | cannot delete self (409), cannot change own role (403) | same | n/a |
| Offerings: create/edit/delete/bulk delete, access, CSV import | yes | curriculum in scope (create: target curriculum; move: old and new both in scope; bulk delete: one out-of-scope id = 403 for the whole request) | 403 |
| Offerings: years list | yes | yes | yes |
| Covers: upload/view/download/delete | yes | curriculum in scope | 403 |
| Submissions: upload | yes | in scope, or access row with upload | access row with `can_upload` |
| Submissions: read / history / issues | yes | in scope or access row | needs view |
| Portfolio PDF + bulk download | yes | in scope or access row | needs download (staff: department default) |
| Templates (`/templates/*`) | yes | yes | yes |

The Portfolios filters of an instructor/staff user are the union of those own lists and the department/curriculum present in the portfolio rows they may see (they can be assigned offerings outside their own department, and those rows must stay filterable).

Error mapping: unauthenticated -> 401 `{"error":"unauthenticated"}`; forbidden -> 403 `{"error":"forbidden"}` (403 is also used for "not visible"; 404 is only used for ids that do not exist at all, e.g. an unknown offering or user).
A CSV import may only **raise** a user from `staff` to `instructor`, never lower a role and never change `admin`/`super_admin` roles.
Per-endpoint roles: [api-reference.md](api-reference.md).

## Session model

1. Login with Google OIDC (authorization-code flow with **PKCE S256 + `state`**; `hd` hint = first allowed domain). Cookie `abet_oauth` (HttpOnly, SameSite=Lax, Path `/public/api/v1/auth`, 10 minutes, signed with `SESSION_SECRET`) carries state and PKCE verifier.
2. The ID token must be valid (issuer, audience, expiry), `email_verified=true`, and the lower-cased e-mail domain **exactly** equal to an entry of `ALLOWED_EMAIL_DOMAINS` (no suffix or sub-domain match, so `evil.chula.ac.th.attacker.com` and `x.student.chula.ac.th` do not pass for `chula.ac.th`).
3. auth-service calls main-service `login-sync`; the user must already exist (**pre-provisioned**), otherwise `403 not provisioned` -> Access Denied. main-service stores name, picture and `last_login_at`.
4. A row is inserted in `auth_db.sessions` and the browser receives cookie **`abet_session`**: HS256 JWT with claims `sub` (user id), `email`, `sid` (session row id), `iat`, `exp`; `HttpOnly`, `SameSite=Lax`, `Path=/`, `Secure` = `COOKIE_SECURE`, `Max-Age` = `SESSION_TTL_HOURS` x 3600 (default 12 h).
5. **Verification on every API call** (gateway forward-auth -> `GET /internal/v1/auth/verify`, an internal-only route that the gateway never exposes): signature + expiry of the JWT + the session row exists, is not revoked, is not expired, belongs to the same user. DB error -> 503 (fails closed).
6. **Logout** revokes the row (`revoked_at`) and clears the cookie, so a stolen copy is dead at once. Expired sessions older than 7 days are purged on startup and hourly.
7. The redirect target after login must be a relative path starting with a single `/` (no `\`, no control characters, no scheme/host), otherwise `/` is used (no open redirect).
8. CSRF: cookie-authenticated state-changing requests are protected by `SameSite=Lax` (cross-site POSTs do not carry the cookie); there is no separate CSRF token.

Secrets that protect this: `SESSION_SECRET` (>= 32 chars, enforced), `INTERNAL_API_TOKEN`, `GOOGLE_CLIENT_SECRET`.

## Gateway header stripping and the identity headers

main-service trusts `X-User-Id` and nothing else. To make that safe:

* Route `/auth/api/*` runs `serverless-pre-function` (rewrite phase, priority 10000, **before** forward-auth at 2002) that clears any client-supplied `X-User-Id` and `X-User-Email`.
* `forward-auth` then **sets** `X-User-Id` / `X-User-Email` on the upstream request from the verify response (overwrite semantics). Requests that fail verification never reach main-service.
* `proxy-rewrite.headers.remove` must **not** be used for this: it runs after forward-auth and would delete the verified headers (documented in [`api-gateway/README.md`](../api-gateway/README.md)).
* main-service is not reachable from outside except through this route: the container has no published port. If you ever publish it, anyone could send `X-User-Id` directly. Do not.
* Tested live: a request with a valid cookie and a spoofed `X-User-Id` reached the upstream with the real id; the e2e suite also checks that a spoofed header is ignored.

## Internal token

`/internal/v1/*` (only `POST /internal/v1/auth/login-sync`) requires header `X-Internal-Token` equal to `INTERNAL_API_TOKEN` (constant-time compare; an **empty configured token rejects everything**). Wrong/missing -> 401. The gateway has no route to `/internal/*` (`/auth/internal/...` does not match `/auth/api/*`); only auth-service calls it over the private docker network (with a circuit breaker and 5 s timeout).

## Upload and size limits

| Limit | Value | Where enforced |
|---|---|---|
| Portfolio zip | `MAX_UPLOAD_MB` = 300 MB (150 MB SRS x 2) | frontend pre-check (limits read from `GET /api/v1/config`); gateway `client_max_body_size` = `max(MAX_UPLOAD_MB, MAX_COVER_MB, MAX_IMPORT_MB) + 20` MB (320 MB by default, computed at container start); main-service early 413 on `Content-Length` > limit + 1 MB framing and `http.MaxBytesReader` while streaming; response `413 {"error":"file exceeds 300 MB limit"}` |
| Zip contents | <= 2000 entries and <= 1 GB uncompressed (zip-bomb guard) -> issue `ZIP_TOO_LARGE` | validator |
| Cover PDF | `MAX_COVER_MB` = 20 MB, must start with `%PDF-` | main-service (413 / 400), frontend |
| Mapping CSV | `MAX_IMPORT_MB` = 5 MB (+1 MB multipart framing) | main-service (413) |
| Timeouts | connect 60 s, send/read 300 s | gateway |

Uploads are streamed to a temp file in `UPLOAD_TMP_DIR` (never fully in memory; the temp file is always removed), must open as a zip (else 400), and are validated *before* anything is stored in MinIO or queued: an invalid upload never creates an object. File names inside the archive are only inspected, never extracted to disk by main-service; the worker extracts inside an ephemeral temp directory that is wiped after every job. Object keys that are empty, absolute or contain `.` / `..` / empty segments are rejected by the storage adapter; the status consumer only accepts exported PDF keys below the job's own `outputs/<submission_id>/` prefix.
Cover responses are served inline with `X-Frame-Options: SAMEORIGIN`, `X-Content-Type-Options: nosniff`, `Cache-Control: no-cache`.

## Dev-only "act as user" (role switcher)

A super admin logged in once can switch to any existing user (any role) and use the app exactly as that user, then switch back. It exists only to test role/scope behaviour on a development stack.

* **Guards**: the feature is ON only when `DEV_ACT_AS_ENABLED=true` **and** `APP_ENV=development` in main-service. Both are set only in `compose.dev.yaml`; `compose.yaml` and `compose.prod.yaml` never set them. If `DEV_ACT_AS_ENABLED=true` is found with another `APP_ENV`, main-service logs an ERROR and keeps the feature OFF; when ON it logs `dev act-as enabled` (WARN) at startup. When OFF the three endpoints are not registered (`404 {"error":"not found"}`), the cookie is never read and `GET /me` answers `act_as: {"available": false, "real_user": null}`.
* **Who**: only a real `super_admin` (the identity from the verified session, never the acted-as user) can list, start or stop acting; anyone else gets `403`. The cookie of a non-super-admin is ignored, as are a malformed, stale (unknown user) or own-id cookie (no 401, the real identity is used).
* **State**: cookie `abet_act_as=<target user uuid>`, `Path=/auth/api` (the public gateway path), `HttpOnly`, `SameSite=Lax`, `Secure` when `COOKIE_SECURE=true`, no `Max-Age` (ends with the browser session). It is set and cleared only by main-service. auth-service, gateway and portfolio-service are untouched; the gateway already forwards `Cookie` to main-service.
* **Effect**: the auth middleware replaces the request actor with the target loaded from the database (same role, scope, course access and capabilities as that user), so every authorization check runs unchanged. **Writes are real** and records that store "who did it" (for example `uploaded_by` of a submission) store the *acting* user. This is acceptable on a dev database and is the reason the feature must never run in production. Request logs get an extra `real_user_id` attribute while acting, and a debug line `acting_as=<id> real_user=<id>`.
* **API**: `GET /api/v1/dev/act-as/users?q=&role=`, `POST /api/v1/dev/act-as {"user_id"}` (204 + cookie), `DELETE /api/v1/dev/act-as` (204, clears cookie); `GET /me` adds `act_as`. See [api-reference.md](api-reference.md) and `plan/contracts/act-as-t11.md`.

## What is exposed in production
## What is exposed in production

* **Published ports**: the gateway (`GATEWAY_HTTP_PORT` -> container 9080) and the **MinIO console** (`MINIO_CONSOLE_HOST_PORT`, default 9001 -> container 9001; the only extra port in `compose.prod.yaml`). Postgres, RabbitMQ, the MinIO **S3 API (9000)**, both Go services, the frontend and the worker are reachable only inside the compose network. Dev additionally publishes Postgres, the RabbitMQ UI and (optional, `MINIO_API_HOST_PORT`) the S3 API.
* **Through the gateway, without login**: the SPA and static assets (`/*`), and an **allowlist of exactly three auth-service routes**: `GET /public/api/v1/auth/google/login`, `GET /public/api/v1/auth/google/callback`, `POST /public/api/v1/auth/logout`. Everything else under `/public/*` (also `healthz`, `readyz`, `metrics`, the old `health` page and any verify path) is answered by the gateway itself with `404 {"error":"not found"}` and never reaches auth-service. auth-service's `/internal/v1/auth/verify` is called only by the gateway forward-auth over the docker network; `/healthz`, `/readyz`, `/metrics` are docker-network only.
* **Through the gateway, with login**: `/auth/api/*` = main-service `/api/v1/*` only. main-service `/internal/*`, `/healthz`, `/readyz`, `/metrics`, `/swagger`, `/debug/pprof` are not routed (swagger and pprof are additionally disabled unless `ENABLE_SWAGGER` / `ENABLE_PPROF` are true; compose sets `ENABLE_SWAGGER` only in dev).
* **Monitoring (optional profile)**: Grafana and Kibana are reachable **only through the gateway** under `/grafana` and `/kibana` (no published port of their own, not in dev either except Prometheus `:9090` in dev). They have their **own logins** (`GRAFANA_ADMIN_*`, `elastic` / `ELASTIC_PASSWORD`); the ABET session does not grant access. On those two routes the gateway removes the `abet_session` and `abet_oauth` cookies and any client `X-User-*` headers, so the tools never see an ABET session. Prometheus, Elasticsearch, APM Server, exporters and the gateway's metrics port are internal only. Services send traces to `apm-server:8200` with the secret `APM_SECRET_TOKEN` (`Authorization: Bearer`); it is a secret like the others (random, in `.env` only, never logged). Traces and logs contain URLs, user ids and e-mail-free request metadata; SQL spans never contain parameter values. When monitoring is off the routes answer the normal `404` JSON and no trace data leaves the services. See [monitoring.md](monitoring.md).
* **Transport**: plain HTTP at the gateway; TLS must be provided in front ([deployment.md](deployment.md#https)). Set `COOKIE_SECURE=true` there.
* **Containers**: the Go services run as non-root `appuser` in minimal Alpine images; the worker runs as root (needed by the TeX toolchain). There is no shared filesystem any more: both talk to MinIO with the **service account** `MINIO_ACCESS_KEY` / `MINIO_SECRET_KEY`, whose policy (`minio-init`) allows `s3:*` only on the bucket `MINIO_BUCKET` and its objects, so a leaked service credential cannot read other buckets or administer MinIO. The MinIO root account is used only by `minio` and `minio-init` and for the console login.
* **MinIO console exposure**: the console is published in production on `MINIO_CONSOLE_HOST_PORT` and is **plain HTTP with the root login**; it is the only way to see and delete files outside the application, and it bypasses the application's authorization. Use a strong `MINIO_ROOT_PASSWORD` (>= 12 characters, unique), do not expose the port to the internet: block it in the firewall, or bind it to localhost (override `ports: ["127.0.0.1:9001:9001"]` for `minio` in a local compose override) and reach it over a VPN/SSH tunnel or a TLS reverse proxy. Browser downloads never use MinIO: they stream through main-service after the permission check, and the S3 API (9000) is never published in prod.
* **Secrets handling**: `.env` is git-ignored; `.env.template` has placeholders; the Go services log JSON without secrets; error responses never include internal error text (500 is always `{"error":"internal error"}`).

### Maintenance notes (observations, not changed)

* There is no rate limiting at the gateway.
* `/api/v1/auth/verify` (without the `/public` prefix) no longer exists in auth-service; at the gateway such a path simply falls through to the SPA (harmless HTML).
* With `sslmode` `require`, `prefer`, `allow` or `verify-ca` the Go services use TLS without verifying the server certificate (libpq semantics); only `verify-full` verifies the host. The stack itself runs with `DB_SSLMODE=disable` on the private docker network.
