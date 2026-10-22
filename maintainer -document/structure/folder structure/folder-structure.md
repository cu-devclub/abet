# Folder structure

Back to the [documentation index](../../README.md). Paths are relative to the repository root.

## Repository root

```
abet/
├── .env.template          documented template; `make env` copies it to .env (.env is git-ignored)
├── .gitignore  .gitattributes   (*.sh forced to LF so the shell scripts run in Linux containers)
├── Makefile               every day-to-day command (see deployment.md "Makefile targets")
├── compose.yaml           base: postgres, rabbitmq, minio, minio-init, auth-service, main-service, portfolio-service, frontend, api-gateway
├── compose.dev.yaml       dev overrides: *.dev Dockerfiles (air / vite), bind-mounted sources, published postgres/rabbit/minio/gateway ports, APP_ENV=development
├── compose.prod.yaml      prod overrides: restart policies, only the gateway port published, prod APISIX routes
├── api-gateway/           APISIX standalone config: config.yaml, apisix.dev.yaml, apisix.prod.yaml, README.md
├── auth-service/          Go service: Google login + sessions         (own go.mod, module abet/auth-service)
├── main-service/          Go service: domain API + pipeline           (own go.mod, module abet/main-service)
├── portfolio-service/     Python worker (LaTeX PDF builder). Treat as vendor code; only worker.py has the cover patch
├── frontend/              React SPA (+ Playwright e2e)
├── deploy/
│   ├── postgres/init-databases.sh        creates auth_db and main_db on the FIRST start of an empty pgdata volume
│   ├── portfolio-service/entrypoint.sh   rewrites /app/settings.json from env, then runs worker.py (see operations.md)
│   ├── dev/seed.sql                      DEV-ONLY example data for `make seed`
│   └── monitoring/                   (optional profile) prometheus/, apm-server/, filebeat/, grafana/{provisioning,dashboards}/, kibana/setup.sh
├── screen-design/         UI prototype (reference only, not part of any build)
└── maintainer -document/  this documentation
```

Generated or local-only directories you will see but should ignore: `*/tmp` (air build output), `frontend/dist`, `frontend/node_modules`,
`frontend/playwright-report`, `frontend/test-results` (all git-ignored).

## Go services: the layering (main-service and auth-service share it)

The layout follows the CUGrader boilerplate: **hexagonal architecture (ports & adapters) + CQRS**. Dependencies point inwards only:

```
 inbound adapter (HTTP)  ──►  application (use cases)  ──►  core/domain + core/port (interfaces)
                                                                     ▲
 outbound adapter (postgres, rabbitmq, storage, google, jwt) ────────┘ implements the ports
```

| Layer | Directory | What lives here | May import |
|---|---|---|---|
| **domain** | `internal/core/domain` | entities, validating constructors, error sentinels (`errors.go`), authorization rules (`policy.go`, main-service) | stdlib only |
| **port** | `internal/core/port` | *interfaces*: use-case interfaces (inbound) and repository/storage/publisher interfaces (outbound), one file per module; `mocks/` = gomock output | domain |
| **application** | `internal/application/commands` (writes) and `queries` (reads) | use cases: check the policy, orchestrate ports. Pure logic that needs no I/O is separate, e.g. `importplan/` | domain, port |
| **inbound adapter** | `internal/adapter/inbound/http` | `router.go`, `handler/<module>.go` (Gin, one `Register(rg)` per module), `dto/<module>.go` (request/response types, `Validate()`), `middleware/` (actor loading, internal token, logging, metrics) | application, port, domain |
| **outbound adapter** | `internal/adapter/outbound/<tech>` | `postgres/` (pgx pool, `store.go`, one `<module>_repository.go` each, generated `sqlc/`), `rabbitmq/`, `storage/` (MinIO adapter, `minio.go`), `templates/`; auth-service: `google/`, `jwt/`, `mainclient/` | port, domain |
| **composition root** | `cmd/api/main.go` | builds everything in numbered steps (config → migrate → pool → repos → use cases → handlers → router → server) and handles graceful shutdown | everything |

Other per-service files: `config/config.go` (environment only), `db/migrations/*.sql` (embedded with `go:embed`, applied at startup),
`db/queries/<module>.sql` (sqlc input), `sqlc.yaml`, `api/openapi/server/spec.yaml` (OpenAPI), `Makefile`, `Dockerfile` (prod, static
binary, non-root `appuser`), `Dockerfile.dev` + `.air.toml` (hot reload), `.pre-commit-config.yaml`, `tests/integration/` (empty placeholder).

### main-service in detail

```
main-service/
├── cmd/api/main.go
├── config/config.go                       env -> Config (defaults listed in deployment.md)
├── db/
│   ├── migrations/000001_init.{up,down}.sql   the WHOLE main_db schema (9 tables); migrations.go embeds them
│   └── queries/  departments curricula users admin_scopes offerings access import covers submissions  (.sql, sqlc)
├── api/openapi/server/spec.yaml
├── assets/instructor-package/             empty in git: compose bind-mounts portfolio-service/templates/instructor-package here (read-only)
├── testdata/mapping_sample.csv mapping_errors.csv    sample mapping files
└── internal/
    ├── core/
    │   ├── domain/       department curriculum user offering import submission email errors policy(.go)
    │   ├── port/         department curriculum user auth offering access import cover submission storage health (.go) + mocks/
    │   └── validation/   PURE upload validator: validate.go tree.go csv.go completeness.go messages.go workererr.go
    │                     + testdata/package/ (a complete valid portfolio used by tests)
    ├── application/
    │   ├── commands/     department curriculum user auth offering access import cover submission helpers (.go)
    │   ├── queries/      department curriculum user offering submission (.go)
    │   └── importplan/   PURE mapping-CSV parser (parse.go) and planner (plan.go)
    └── adapter/
        ├── inbound/http/ router.go; handler/{me,department,curriculum,user,offering,import,cover,submission,template,internal_auth,health,errors}.go;
        │                 dto/; middleware/{actor,internal_token,logging,metrics}.go
        └── outbound/     postgres/ (store.go, *_repository.go, sqlc/)  rabbitmq/ (config, publisher, consumer, inspector)
        inbound/scheduler/ reconciler + queue health + janitor ticker; application/queuehealth/ (no-consumer detection)
                          storage/minio.go  templates/ (instructor-package zip, mapping CSV sample)
    └── testsupport/memstore  in-memory port.Storage used by unit tests (adapter specs use a real MinIO when MINIO_TEST_ENDPOINT is set)
```

### auth-service in detail

```
auth-service/
├── cmd/api/main.go                server (also an hourly cleanup of sessions expired > 7 days)
├── cmd/devsession/main.go         DEV-ONLY CLI, refuses unless APP_ENV=development (not in the prod image)
├── db/migrations/000001_init      table sessions (+ idx on user_id, expires_at); db/queries/sessions.sql
└── internal/
    ├── core/domain      session, identity, email_policy (exact-domain match), oauth_state, redirect (sanitiser), errors
    ├── core/port        auth.go (+ mocks/)
    ├── application/     commands/{start_google_login,google_callback,logout,cleanup_sessions}.go   queries/ (verify session)
    └── adapter/
        ├── inbound/http  router, handler/{auth,health,oauth_cookie,errors}.go, middleware (the HTML /health page was removed)
        └── outbound/     postgres/ (+sqlc), google/ (OIDC), jwt/ (HS256), mainclient/ (login-sync, circuit breaker)
```

Both Go services also contain `internal/adapter/outbound/telemetry/` (OpenTelemetry setup + slog trace handler, see [monitoring.md](../../monitoring.md#tracing-in-the-code-for-developers));
main-service adds `internal/adapter/outbound/rabbitmq/trace.go` (W3C headers in AMQP messages). `portfolio-service/worker.py` contains the worker side of the tracing.

### Where to add a new module (main-service)

A "module" is one business area (e.g. `reports`). Add files, don't edit shared ones, except two additive lines in `cmd/api/main.go`.

| Step | File to create | Notes |
|---|---|---|
| 1 | `db/queries/reports.sql` | then `make sqlc` in `main-service/` |
| 2 | `internal/core/port/reports.go` | repository interface + use-case interfaces; then `make mock` |
| 3 | `internal/core/domain/reports.go` (+ rules in `policy.go`) | entities, validation, authorization methods on `Actor` |
| 4 | `internal/adapter/outbound/postgres/reports_repository.go` | uses `store.q` / `store.WithTx`, maps errors with `mapErr` |
| 5 | `internal/application/commands/reports.go` / `queries/reports.go` | check policy first, return `domain.ErrForbidden` |
| 6 | `internal/adapter/inbound/http/dto/reports.go`, `handler/reports.go` | `Register(rg *gin.RouterGroup)` |
| 7 | `cmd/api/main.go` | construct repo/use case/handler, append handler to `routes.API` (or `routes.Internal`); `router.go` stays untouched |
| 8 | `api/openapi/server/spec.yaml` | paths include the `/api/v1` prefix |
| 9 | tests | Ginkgo + gomock, external `_test` package per directory |

A new database table needs a **new migration** (`make migrate-new name=...`); see [development.md](../../development.md).
Full walk-through: [development.md](../../development.md#adding-an-endpoint-end-to-end).

## frontend

```
frontend/
├── index.html  vite.config.ts  tsconfig*.json  .oxlintrc.json  package.json(+lock)  nginx.conf  Dockerfile  Dockerfile.dev
├── playwright.config.ts
├── e2e/                      Playwright suite (43 tests): global-setup.ts seeds data + mints cookies; *.spec.ts per area
└── src/
    ├── main.tsx
    ├── app/                  App, router.tsx (route table + guards), providers.tsx (QueryClient, Toast, 401 handler), queryClient.ts
    ├── config/env.ts         API bases: AUTH_API=/public/api/v1, MAIN_API=/auth/api/v1; size limits are read from `GET /auth/api/v1/config` (`useAppConfig()`)
    ├── lib/                  http.ts (fetch wrapper, ApiError, upload progress), download.ts, zipFolder.ts (folder -> zip in the browser),
    │                         format.ts, cn.ts, useSort.ts
    ├── components/
    │   ├── ui/               design-system pieces: Button, Modal, Table, Toast, FileDropzone, ...
    │   └── layout/           AppShell, Sidebar (role-based nav), PageHeader
    ├── features/             one folder per page/area, each owns its API hooks and types
    │   ├── auth/             useMe, LoginPage, AccessDeniedModal, RequireAuth, RequireCapability
    │   ├── portfolios/       Course Portfolios page: PortfoliosPage, api.ts, components/{PortfolioTable,UploadPortfolioModal,IssuesModal,BulkDownloadBar}
    │   ├── courses/          Courses page: offerings, access sub-table, ImportMappingModal
    │   ├── covers/           Cover Page page
    │   ├── users/  departments/  curricula/
    │   └── shared/           useOrgFilters (URL-synced filters), useOrgOptions/scope (admin scope limiting), YearSelect
    ├── styles/index.css      Tailwind import + theme tokens
    └── test/                 vitest setup and helpers
```

| Route | Page | Visible when (`/me` capabilities) |
|---|---|---|
| `/login` | Login (+ Access Denied modal on `?error=access_denied`) | public |
| `/portfolios` (default) | Course Portfolios | any signed-in user |
| `/courses` | Courses | `manage_courses` |
| `/covers` | Cover Page | `manage_covers` |
| `/users` | Users | `manage_users` |
| `/departments`, `/curricula` | Departments, Curriculums | `manage_org` |

### Where to add a frontend feature

1. Create `src/features/<name>/` with `types.ts` (mirror the API JSON, snake_case), `api.ts` (TanStack Query hooks, query key `['<name>', params]`),
   `<Name>Page.tsx` and `components/`.
2. Register the route (with the right `RequireCapability`) in `src/app/router.tsx`; add the sidebar entry in `components/layout/Sidebar.tsx`.
3. Reuse `components/ui` and `lib/http.ts` (never call `fetch` directly; it adds credentials and 401 handling).
4. Add a vitest next to the code and, for user-visible flows, a spec in `e2e/`.

## portfolio-service and deploy/

`portfolio-service/` is the original Python project (`worker.py`, `config.py`, `settings.json`, `scripts/`, `templates/`, `programs/`, ...).
Do not restructure it. The deploy shim `deploy/portfolio-service/entrypoint.sh` is mounted at `/deploy` and used as the container entrypoint so
the worker's RabbitMQ settings come from `.env` without editing `settings.json` in the repo.
