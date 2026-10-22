# main-service

Domain API of the ABET Course Portfolio Management System (Go, Gin, hexagonal + CQRS, PostgreSQL via pgx + sqlc).
Reached by the browser through the gateway under `/auth/api/*` (prefix stripped); auth-service calls `/internal/v1/*` directly.

Implemented so far (T02): `me`, `departments`, `curricula` (CRUD), `users` (+ admin scopes), internal `login-sync`,
health/metrics, super-admin bootstrap. Contracts: `plan/contracts/{api,authz,db}.md`. OpenAPI: `api/openapi/server/spec.yaml`.

## Configuration (environment only)

| key | default | meaning |
|---|---|---|
| `PORT` | `8080` | listen port |
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_NAME` | `localhost` / `5432` / `abet` / `main_db` | Postgres connection (separate values, no URL) |
| `DB_PASSWORD` | empty | used verbatim: any character works, it is never put in a URL |
| `DB_SSLMODE` | `disable` | `disable`, `allow`, `prefer`, `require`, `verify-ca`, `verify-full` |
| `LOG_LEVEL` | `info` | slog level (case-insensitive) |
| `INTERNAL_API_TOKEN` | empty | `X-Internal-Token` for `/internal/v1/*`; **empty rejects every internal call** |
| `SUPER_ADMIN_EMAILS` | empty | comma separated; upserted as `super_admin` at startup |
| `ALLOWED_EMAIL_DOMAINS` | `chula.ac.th` | comma separated, exact domain match |
| `MINIO_ENDPOINT` | `minio:9000` | S3 endpoint (host:port, no scheme) of the MinIO that holds every file |
| `MINIO_ACCESS_KEY` / `MINIO_SECRET_KEY` | empty | service account of the bucket; used verbatim (any character works) |
| `MINIO_BUCKET` / `MINIO_USE_SSL` | `abet` / `false` | bucket (created at startup when missing) / TLS to MinIO |
| `UPLOAD_TMP_DIR` | `os.TempDir()` | local scratch dir where an upload is spooled before validation (always cleaned up) |
| `MAX_UPLOAD_MB` / `MAX_COVER_MB` / `MAX_IMPORT_MB` | `300` / `20` / `5` | upload limit (zip) / cover limit / mapping CSV limit; also served by `GET /api/v1/config` |
| `BUILD_TIMEOUT_MINUTES` | `60` | a submission still queued after this long becomes `failed` / `BUILD_TIMEOUT` |
| `RECONCILE_INTERVAL_SECONDS` | `60` | tick of the reconciler, queue health check and (hourly) storage janitor |
| `RABBITMQ_HOST/PORT/USER/PASS/VHOST/QUEUE/STATUS_QUEUE` | see `config/config.go` | job + status queues of the portfolio worker |
| `INSTRUCTOR_PACKAGE_DIR` | `/app/assets/instructor-package` | template zip source |
| `ENABLE_SWAGGER` / `ENABLE_PPROF` | `false` | `/swagger/`, `/debug/pprof/` |

## Run / test

```
make run                 # needs a reachable Postgres (DB_* variables); migrations run at startup
make test                # go test ./... -count=1  (Ginkgo v2 + Gomega + gomock)
make mock                # regenerate gomock mocks for every internal/core/port/*.go
make sqlc                # regenerate internal/adapter/outbound/postgres/sqlc (docker)
make migrate-new name=add_foo   # new db/migrations/NNNNNN_add_foo.{up,down}.sql
```

Dev container: `Dockerfile.dev` (air). Prod: `Dockerfile` (static binary, non-root `appuser`, HEALTHCHECK `/healthz`).

## Layout

```
cmd/api/main.go                        wiring (numbered steps), graceful shutdown
config/config.go                       env -> Config
db/migrations/*.sql, db/migrations.go  golang-migrate files, embedded, applied at startup (ALL main_db tables live in 000001)
db/queries/<module>.sql                sqlc queries, one file per module
internal/core/domain                   entities, errors (sentinels), policy.go (authorization), email.go
internal/core/port/<module>.go         use-case interfaces + repository interfaces of ONE module
internal/core/port/mocks               mockgen output (mock_<module>.go)
internal/application/commands|queries  use cases (Execute), one file per module
internal/adapter/inbound/http          router.go, handler/<module>.go, dto/<module>.go, middleware/
internal/adapter/outbound/postgres     store.go (pool, WithTx, Migrate, error mapping) + <module>_repository.go
internal/adapter/outbound/storage      minio.go (port.Storage on a MinIO bucket, minio-go v7)
internal/testsupport/memstore          in-memory port.Storage for tests
```

## Cross-cutting building blocks (reuse them)

* **Actor + policy** — `middleware.Authenticate` (applied to the `/api/v1` group only) reads `X-User-Id`, loads user + expanded
  admin scope from the DB on every request and stores a `*domain.Actor`; unknown/missing -> 401 `{"error":"unauthenticated"}`.
  In handlers: `actor := middleware.ActorFrom(c)`; pass it to the use case. All rules live in `internal/core/domain/policy.go`:
  `CanManageOrg`, `CanManageCurriculum(id)`, `CanSeeUser`, `CanManageUser`, `CanAssignRole`, `CanSetAdminScope`,
  `OfferingPermissions(OfferingScope, *AccessGrant) OfferingPerms` (view/upload/download/manage + `IsDefault`).
  `Actor.ScopeCurriculumIDs` already includes the curricula of scoped departments.
* **Errors** — return domain sentinels (`ErrInvalidInput`, `ErrForbidden`, `ErrNotFound`, `ErrConflict`, `ErrTooLarge`,
  `ErrUnprocessable`, ...). Attach a client-safe message with `fmt.Errorf("%w: department code already exists", domain.ErrConflict)`;
  `handler/errors.go` `writeError` maps sentinel -> status and sends the text after the sentinel. Anything else becomes
  `500 {"error":"internal error"}` and is logged with slog (never leaked). Repositories translate PG unique/FK violations to `ErrConflict`
  and `pgx.ErrNoRows` to `ErrNotFound` (`mapErr` in `postgres/store.go`).
* **Transactions** — `Store.WithTx(ctx, func(q *sqlc.Queries) error)` (commit on nil, rollback otherwise). Repository methods that
  must be atomic (e.g. user + admin scopes, CSV import) open the transaction themselves; use cases stay storage-agnostic.
* **Storage** — `port.Storage` (`Save` atomic temp+rename, `Open`, `Delete` (missing = nil), `DeleteDir`, `Path`, `Exists`), keys such as
  `covers/<id>.pdf`, `uploads/<submission_id>.zip`; keys containing `..` are rejected. `commands.CoverStorageKey(id)` builds the cover key.
* **Internal API** — `middleware.InternalToken(token)` guards `/internal/v1` (constant-time compare, empty config rejects all).
* **Email domains** — `domain.ValidateEmail(email, allowed)` / `domain.IsAllowedEmailDomain`.
* **Responses** — `dto.Items(list)` for `{"items":[...]}` (never `null`), `dto.FormatTime` for RFC3339 UTC, `bindJSON` +
  `parseUUIDParam` helpers in `handler/errors.go`.

## Modules

| module | files (queries / port / use cases / handler) | endpoints |
|---|---|---|
| me, departments, curricula, users, auth (T02) | `departments.sql`, `curricula.sql`, `users.sql`, `admin_scopes.sql`, `port/{department,curriculum,user,auth}.go` | `/me`, `/departments`, `/curricula`, `/users`, `/internal/v1/auth/login-sync` |
| offerings (T04) | `offerings.sql`, `port/offering.go`, `commands/offering.go`, `queries/offering.go`, `handler/offering.go` | `/offerings` CRUD, `/offerings/years`, `/offerings/bulk-delete` |
| access (T04) | `access.sql`, `port/access.go`, `commands/access.go`, `queries/offering.go` (list) | `/offerings/{id}/access[/{user_id}]` |
| mapping CSV import (T04) | `import.sql`, `port/import.go`, `application/importplan` (pure parser + planner), `commands/import.go`, `handler/import.go` | `/offerings/import/preview`, `/offerings/import/apply` |
| covers (T04) | `covers.sql`, `port/cover.go`, `commands/cover.go`, `handler/cover.go` | `/curricula/{id}/cover` (PUT, GET, DELETE) |
| submissions, portfolios (T05) | `submissions.sql`, `port/submission.go`, `commands/submission.go`, `queries/submission.go`, `handler/submission.go`, `core/validation`, `outbound/rabbitmq` | `/offerings/{id}/submissions` (POST, GET), `/submissions/{id}`, `/offerings/{id}/portfolio.pdf`, `/portfolios`, `/portfolios/download` |
| templates (T05) | `outbound/templates`, `handler/template.go` | `/templates/instructor-package.zip`, `/templates/course-instructor-mapping.csv` |

The mapping import is split into pure functions (`importplan.ParseCSV`, `importplan.BuildImportPlan`: no DB, no I/O) and
`ImportRepository.Apply`, which locks the scope offerings (`SELECT ... FOR UPDATE`), loads the state, runs the planner and writes the
plan in ONE transaction. Offering deletes remove the submission objects (`uploads/<sid>.zip`, `portfolios/<sid>.pdf`,
`outputs/<sid>/`) only after the commit, best effort.

## File storage (T09, MinIO)

Every file is an object in ONE bucket (`MINIO_BUCKET`); `port.Storage` (`Put`, `PutFile`, `Get`, `Stat`, `Copy`, `Delete`, `DeletePrefix`, `List`, `Ping`)
is implemented by `outbound/storage/minio.go`. Layout: `uploads/<sid>.zip` (written after validation; deleted by the worker on success or by
main-service), `covers/<curriculum_id>.pdf`, `outputs/<sid>/<files>` (written by the worker; the PDF is copied, then the prefix deleted),
`portfolios/<sid>.pdf` (final PDF). Downloads stream through main-service (`Get` + Content-Length; no Range support). `/readyz` pings the
bucket. The adapter specs (`go test ./internal/adapter/outbound/storage`) run against a real MinIO when `MINIO_TEST_ENDPOINT`,
`MINIO_TEST_ACCESS_KEY`, `MINIO_TEST_SECRET_KEY` are set and are skipped otherwise.

## Submission pipeline (T05)

Contract: `plan/contracts/submission-pipeline.md`. Flow of `POST /offerings/{id}/submissions`:

1. authorise upload (`OfferingPermissions`); the multipart body is read as a stream (`MultipartReader`, `http.MaxBytesReader`
   with `MAX_UPLOAD_MB` + 1 MB framing -> 413 `file exceeds N MB limit`) and spooled to a temp file in `UPLOAD_TMP_DIR` (removed in every case);
2. the temp file must open as a zip (else 400), then `validation.Validate` checks it against SRS section 7
   (pure, no DB/disk — see below) -> `[]domain.Issue`;
3. errors -> one transaction inserts the submission as `invalid` (nothing is stored in MinIO, nothing queued); otherwise the file is
   uploaded with `PutFile` to `uploads/<submission_id>.zip` and one transaction inserts the submission as `queued`;
4. the job goes to `RABBITMQ_QUEUE` (persistent, `application/json`, publisher confirm) as payload v2:
   `{"job_id","program_name","storage":"s3","zip_key":"uploads/<sid>.zip","cover_key":"covers/<cid>.pdf" (only when that object exists),
   "output_prefix":"outputs/<sid>/","delete_zip_on_completion":true}` (no filesystem paths). If publishing fails the uploaded object is
   deleted and the submission becomes `failed` with one `QUEUE_UNAVAILABLE` issue; the response is still 201.

The **status consumer** (`outbound/rabbitmq/consumer.go`, started in `main.go`, stopped on shutdown) reads `RABBITMQ_STATUS_QUEUE`
(prefetch 10, manual ack after the DB commit, reconnect with 1s..30s backoff, queues declared durable without arguments like
`worker.py`). `commands.ProcessBuildStatusHandler`: unknown job -> ack; `success` -> take the first `*-main.pdf` (else first `.pdf`) of the event's `exported_pdf_keys` (only keys below `outputs/<sid>/`),
`Copy` it server-side to `portfolios/<sid>.pdf`, `Stat` it for the size, mark `generated`, clear `pdf_size_bytes` of older generated
submissions of the offering and delete their PDFs, delete the `outputs/<sid>/` prefix; no PDF -> `failed` + `BUILD_NO_OUTPUT`; `failed` -> `validation.ParseWorkerFailure(error)` (bullets below
`Errors:` -> issues, known SRS messages mapped to their codes, else one `BUILD_FAILED`) and `uploads/<sid>.zip` and the `outputs/<sid>/` prefix are deleted.
`processing` is never set (the worker does not report it); the UI shows `queued` as "Processing".

`internal/core/validation` (`Validate(r io.ReaderAt, size, expectedCourseCode)`): a port of `runtime_pipeline.py`
(`copy_source_to_job_dir`, wrapper collapsing up to 5 levels, junk stripping, `validate_structure`, `validate_course_files`) and the four
CSV generator scripts, messages/descriptions/fixes in `messages.go`. Deliberate differences from the worker (all stricter or
SRS-driven): the 5 "artifact-only" folders are errors when missing (the worker would auto-create them), an empty schedule is an error
(SRS 7.4), junk is only `__MACOSX`, `.DS_Store`, `._*` (exactly the worker's list), unknown files are errors (the worker auto-repairs
them), `INCOMPLETE_DOCUMENT` findings are warnings only. Limits: more than 2000 entries or more than 1 GB uncompressed -> `ZIP_TOO_LARGE`.
Fixture: `internal/core/validation/testdata/package` (the portfolio-service instructor package with a filled schedule CSV; the worker
builds it into a 128-page PDF).

Bulk download streams `archive/zip` straight into the response (PDFs stored, not recompressed) after all checks ran; unavailable,
forbidden or unknown offerings are listed in `MISSING.txt`. `GET /portfolios` runs two queries (offerings + latest submission via one
correlated join, and the user's access rows) and applies `OfferingPermissions` in Go.

## Never stuck in "Processing" (T08)

Background jobs (`internal/adapter/inbound/scheduler`, started in `main.go`, one tick at startup and then every `RECONCILE_INTERVAL_SECONDS`):

- **Reconciler** (`commands.ReconcileBuildsHandler`): one atomic `UPDATE ... WHERE status='queued' AND created_at < now() - BUILD_TIMEOUT_MINUTES`
  sets `failed` + one `BUILD_TIMEOUT` issue; the upload zip and the `outputs/<sid>/` objects are kept (the worker may still
  finish; the janitor removes them later). Counter `abet_submissions_timed_out_total`.
- **Late success** (`ProcessBuildStatusHandler`): a `success` event for a `failed` submission that has a `BUILD_TIMEOUT` issue turns it into
  `generated` (the timeout issue is removed, `error_count` recalculated, normal PDF handling). A `failed` event for a terminal submission is acked and ignored.
- **Queue health** (`outbound/rabbitmq/inspector.go`, passive declare): consumers and waiting jobs in `/readyz` (`build_queue`) and the gauges
  `abet_build_queue_consumers`, `abet_build_queue_messages`; ERROR log once per incident when there were 0 consumers for more than 5 minutes
  (`application/queuehealth`). Uploads stay possible without workers.
- **Janitor** (hourly): deletes `uploads/<sid>.zip` and the `outputs/<sid>/` group of terminal or unknown submissions, and `portfolios/<sid>.pdf` without a
  `generated` row, judged by `List(prefix, recursive)` + `LastModified` (an `outputs/<sid>/` group is as old as its newest object), only when older than 1 hour; never anything of a `queued` submission.

`GET /api/v1/config` returns `max_upload_mb`, `max_cover_mb`, `max_import_mb`, `build_timeout_minutes` from the environment.
`GET /departments` and `/curricula` show instructor/staff only their own department/curriculum; super_admin and admin see everything.

## Extending: how to add a module (T04 / T05)

Modules are independent files; shared files are touched only additively.

1. **Queries**: add `db/queries/<module>.sql` (sqlc). Regenerate with `make sqlc` (new `<module>.sql.go`, plus `models.go`/`querier.go` update).
   No new migration: the whole schema is already in `000001_init` (ask the CTO before adding one).
2. **Port file**: `internal/core/port/<module>.go` with the repository interface (outbound) and the use-case interfaces + input structs
   (inbound). Run `make mock` (creates `mocks/mock_<module>.go`).
3. **Domain**: entities/constructors in `internal/core/domain/<module>.go`; put new authorization rules in `policy.go` as `Actor` methods
   (and add them to `policy_test.go`).
4. **Repository**: `internal/adapter/outbound/postgres/<module>_repository.go` — `type XRepository struct{ store *Store }`,
   `var _ port.XRepository = (*XRepository)(nil)`, use `r.store.q` for single statements, `r.store.WithTx` for atomic ones, wrap errors with `mapErr`.
5. **Use cases**: `internal/application/commands/<module>.go` / `queries/<module>.go` (`NewXHandler(deps...)`, `Execute(ctx, actor, in)`); always check
   the policy first and return `domain.ErrForbidden`. Test with gomock (`<module>_test.go`, external `_test` package).
6. **HTTP**: `dto/<module>.go` (requests with `Validate()`, responses with `XFromDomain`), `handler/<module>.go` with
   `func (h *XHandler) Register(rg *gin.RouterGroup)` mounting routes on the already authenticated `/api/v1` group.
7. **Wire**: in `cmd/api/main.go` construct repo + use cases + handler and append the handler to `routes.API`
   (or `routes.Internal` for `/internal/v1`). `router.go` needs no change.
8. **Spec**: add the paths to `api/openapi/server/spec.yaml` (paths include the `/api/v1` / `/internal/v1` prefix).

Route notes: `rg.Group("/curricula")` may be created again by a later handler (e.g. covers: `/curricula/:id/cover`);
Gin accepts distinct sub-routes. Keep static segments (`/offerings/import/preview`) next to `:id` routes — Gin prioritises static ones.

## Notes

* The sqlc package under `internal/adapter/outbound/postgres/sqlc` is generated by the real tool (`make sqlc`, sqlc 1.29.0 in Docker).
* `GIN_MODE` defaults to `release` unless set.
