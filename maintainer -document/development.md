# Development guide

Back to the [documentation index](README.md). Commands run from the repository root unless a directory is given.

## Start the dev stack

```bash
make env            # once; then fill GOOGLE_*, SUPER_ADMIN_EMAILS if you want real Google login
make dev            # docker compose -p <name>-dev -f compose.yaml -f compose.dev.yaml up -d --build
make ps             # all services healthy?
make logs s=main-service
```

Every ops target accepts `ENV=dev|prod` (default `dev`): `make logs ENV=prod s=main-service`, `make worker-restart`. The dev stack is the compose project `<COMPOSE_PROJECT_NAME>-dev` (`abet-dev`), separate from the production project `abet`, so both can exist on one machine without sharing containers or volumes. `make seed`, `make dev-session` and `make e2e` are dev only and refuse `ENV=prod`. Details: [deployment.md](deployment.md#makefile-targets).

Open `http://localhost:8000` (`GATEWAY_HTTP_PORT`). Dev differs from prod as follows ([`compose.dev.yaml`](../compose.dev.yaml)):

| Service | Dev behaviour |
|---|---|
| `auth-service`, `main-service` | `Dockerfile.dev` runs **air**: sources are bind-mounted (`./<service>:/app`), a change to a `.go/.yaml/.html` file rebuilds and restarts the service. `ENABLE_SWAGGER=true` (`/swagger` inside the network). `auth-service` also gets `APP_ENV=development` |
| `frontend` | `Dockerfile.dev` runs the **Vite dev server** (:5173) with HMR over the gateway websocket (`VITE_HMR_CLIENT_PORT`); `VITE_USE_POLLING=true` because file events do not cross a Windows bind mount; `node_modules` is an anonymous volume |
| `postgres` | published on `POSTGRES_HOST_PORT` (default 5432): `psql -h localhost -U abet main_db` or `make db-shell` |
| `rabbitmq` | management UI published on `RABBITMQ_MGMT_HOST_PORT`: `make rabbit-ui` prints the URL (login = `RABBITMQ_USER`/`RABBITMQ_PASS`) |
| `api-gateway` | `apisix.dev.yaml` (frontend upstream is `frontend:5173`, websocket enabled); route files are re-read automatically on change |
| `portfolio-service` | same image as prod, `PORTFOLIO_WORKER_REPLICAS` (default 2) containers (no hot reload; `make restart s=portfolio-service` or `make worker-restart` after editing Python). To test the build timeout locally set `BUILD_TIMEOUT_MINUTES=1` in `.env`, run `docker compose -p abet-dev stop portfolio-service` and upload |

Stop with `make down` (keeps data) — `make clean` asks for confirmation (it names the stack) and wipes the volumes of that stack only.

Database settings of the Go services are `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE` (there is no `DATABASE_URL`); when running a service outside docker (`make run`) export these.

### Dev session, seed data and end-to-end tests (no Google needed)

| Command | Purpose |
|---|---|
| `make seed` | loads [`deploy/dev/seed.sql`](../deploy/dev/seed.sql) (idempotent, safe to repeat) into `main_db`: departments `CS`, `MATH`; curricula `CS-B`, `CS-M`, `MATH-B`; users `super1`, `admin1` (scope: dept CS), `admin2` (scope: curriculum MATH-B), `instructor1/2`, `staff1/2` (all `@chula.ac.th`); four courses with 2026 S1 (and S2) offerings and some access rows |
| `make dev-session email=admin1@chula.ac.th` | prints `abet_session=<jwt>` for an **existing** user. Runs `go run ./cmd/devsession` inside the auth-service container; refuses unless `APP_ENV=development` (set only in `compose.dev.yaml`; the prod image does not even contain the tool) |
| `make e2e` | Playwright (chromium) suite, 43 tests, about 1.5 min, against the running dev stack. First time: `cd frontend && npm ci && npx playwright install chromium`. It seeds the data itself, mints one cookie per role and cleans up what it creates |

Using a dev session with curl or the browser:

```bash
curl -H "Cookie: $(make -s dev-session email=admin1@chula.ac.th)" http://localhost:8000/auth/api/v1/me
# browser: DevTools -> Application -> Cookies -> add abet_session (Path /) with the value after "="
```

The e2e suite is meant to be run on a dev stack where `SUPER_ADMIN_EMAILS` does not matter; it relies on the seeded users. Never run `make seed` or dev sessions against production data.

### Acting as another user (dev only)

On the dev stack (`compose.dev.yaml` sets `DEV_ACT_AS_ENABLED=true` and `APP_ENV=development` for main-service) a logged-in **super admin** can test the app as any user without Google logins: use the "Switch role" switcher in the app (it only appears when `GET /me` returns `act_as.available: true`), or call the API:

```bash
S="$(make -s dev-session email=<super admin e-mail>)"        # real super_admin session
curl -b "$S" "http://localhost:8000/auth/api/v1/dev/act-as/users?role=instructor"
curl -b "$S" -c jar -H 'Content-Type: application/json' -d '{"user_id":"<uuid>"}' http://localhost:8000/auth/api/v1/dev/act-as   # 204
curl -b "$S" -b jar http://localhost:8000/auth/api/v1/me      # user/capabilities of the target, act_as.real_user = you
curl -b "$S" -X DELETE http://localhost:8000/auth/api/v1/dev/act-as                                                          # back
```

Things to know: writes made while acting are real and recorded as the acting user (for example `uploaded_by`); the list contains only existing users (create them in the Users page or `make seed` first); after changing the env vars recreate main-service (`make dev` recreates only changed containers). Production never enables it (two env guards, see [security-and-permissions.md](security-and-permissions.md#dev-only-act-as-user-role-switcher)).

## Running tests

| Scope | Command | Notes |
|---|---|---|
| everything | `make test` | Go tests of both services + frontend vitest |
| lint | `make lint` | `go vet` for both services + `oxlint` |
| one Go service | `cd main-service && go test ./... -count=1` (or `make test`) | Ginkgo v2 + Gomega + gomock; plain `go test` is enough, the ginkgo CLI is optional |
| one package | `go test ./internal/core/validation/... -count=1` | |
| frontend unit | `cd frontend && npm test` | vitest + Testing Library (57 tests) |
| frontend types/build | `cd frontend && npx tsc -b && npm run build` | |
| e2e | `make e2e` | needs the dev stack |

There are no DB-backed integration tests (`tests/integration/` is an empty placeholder): repositories are covered by unit tests with mocks plus the e2e run.
Go on a local Windows/386 install has no race detector; use `go test -race` inside a Linux container if you need it.

## Code generation

Run these inside the service directory (`main-service/` or `auth-service/`); they use that directory's `Makefile`.

| Task | Command | Details |
|---|---|---|
| sqlc (typed queries) | `make sqlc` | `docker run --rm -v "$PWD:/src" -w /src sqlc/sqlc:1.29.0 generate`. Input: `db/migrations` (schema) + `db/queries/*.sql`; output: `internal/adapter/outbound/postgres/sqlc/` (committed; do not edit by hand) |
| gomock mocks | `make mock` | for every `internal/core/port/*.go` writes `internal/core/port/mocks/mock_<name>.go` using `go run go.uber.org/mock/mockgen@v0.6.0` (needs network the first time) |
| new migration | `make migrate-new name=add_foo` | creates the next numbered pair `db/migrations/00000N_add_foo.{up,down}.sql` (empty files) |
| tidy deps | `make tidy` | `go mod tidy && go mod download` |

### Migrations workflow

1. `make migrate-new name=add_report_table` in the service directory; fill the `.up.sql` (and a working `.down.sql`).
2. Migrations are embedded (`db/migrations.go`, `go:embed`) and applied **automatically at service start** (golang-migrate, forward only, in one `schema_migrations` table per database). With air running in dev, saving the file restarts the service and applies it.
3. Add/adjust queries in `db/queries/<module>.sql` and run `make sqlc` (sqlc reads the migrations as the schema, so column types come from them).
4. Never edit an already released migration: add a new one. A failed migration marks the database "dirty"; fix the SQL, then repair `schema_migrations` manually (`UPDATE schema_migrations SET dirty=false, version=<previous>`) and restart.
5. Both services currently have exactly one migration (`000001_init`); all of `main_db` lives in it.

## Adding an endpoint end-to-end

Example: a read endpoint `GET /api/v1/offerings/{id}/summary` (use the existing modules as templates: `offering`, `cover`, `submission`).

1. **Contract first** — add the path to [`main-service/api/openapi/server/spec.yaml`](../main-service/api/openapi/server/spec.yaml) (paths include the `/api/v1` prefix; reuse the `Error` responses). Document the role requirement.
2. **SQL** — add the query to `db/queries/offerings.sql`, run `make sqlc`.
3. **Domain** — types and validation in `internal/core/domain/offering.go`. If a new permission rule is needed, add an `Actor` method in `policy.go` **and a row in `policy_test.go`**.
4. **Port** — in `internal/core/port/offering.go` add the repository method and the use-case interface/input struct; run `make mock`.
5. **Repository** — implement the method in `internal/adapter/outbound/postgres/offering_repository.go` (single statement: `r.store.q`; several statements: `r.store.WithTx`; wrap errors with `mapErr` so `pgx.ErrNoRows` -> `ErrNotFound`).
6. **Use case** — `internal/application/queries/offering.go` (read) or `commands/` (write): load the offering, call `actor.OfferingPermissions(...)`, return `domain.ErrForbidden` / `ErrNotFound` sentinels. Add a gomock test.
7. **HTTP** — response DTO in `dto/offering.go`, handler method in `handler/offering.go` (`actor := middleware.ActorFrom(c)`, parse params with `parseUUIDParam`, send errors with `writeError`), register the route in the handler's `Register`. Handler tests go next to `offering_http_test.go`.
8. **Wire** — only needed for a *new* handler/use case: construct it in `cmd/api/main.go` (numbered wiring steps) and append to `routes.API`.
9. **Frontend** — add the type to `frontend/src/features/<feature>/types.ts` (snake_case as in the JSON), a query/mutation hook in the feature's `api.ts` (use `apiGet/apiPost/...` from `lib/http.ts`, base `MAIN_API = /auth/api/v1`), use it in the page/component, invalidate the right query keys on mutations. Hide buttons using `/me` `capabilities` or the row's flags, but remember the server is the enforcement point.
10. **Docs/tests** — update [api-reference.md](api-reference.md) (and [security-and-permissions.md](security-and-permissions.md) for new rules), add a vitest and, for user-visible flows, an e2e spec.
11. `make test && make lint && (cd frontend && npm run build)`.

Conventions that the handlers and tests rely on: JSON snake_case, UUID strings, RFC3339 UTC timestamps, lists as `{"items":[...]}` (never `null`), errors as `{"error":"..."}` with 400/401/403/404/409/413/422; unexpected errors become `500 {"error":"internal error"}` (the real error is only logged).
Forbidden means 403 (not 404) everywhere, except an unknown offering id which is 404 before the permission check.

## Working on other components

* **API gateway**: edit `api-gateway/apisix.dev.yaml` / `apisix.prod.yaml` (both must end with `#END`; keep them in sync apart from the frontend upstream port and websocket flag). See [`api-gateway/README.md`](../api-gateway/README.md).
* **portfolio-service**: Python; change `portfolio-service/*` then `make restart s=portfolio-service`. Keep the cover patch ([operations.md](operations.md#portfolio-service-cover-patch-d1)) when updating from upstream.
* **Validator**: [`main-service/internal/core/validation`](../main-service/internal/core/validation) is a pure port of the worker's validation; its fixture `testdata/package` is a real package that the worker builds into a 128-page PDF. If you change either side, run both and keep [business-rules.md](business-rules.md) in sync.
