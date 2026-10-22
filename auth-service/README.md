# auth-service

Go (Gin, hexagonal + CQRS, same layout as the CUGrader boilerplate). Handles Google OAuth 2.0 / OIDC login (PKCE S256 + state),
session cookies with server-side revocation, and the `verify` endpoint used by the API gateway (APISIX forward-auth).

Browser paths are prefixed with `/public` by the gateway (prefix stripped before it reaches this service). Paths below are service-internal.
The gateway exposes ONLY `login`, `callback` and `logout` (an allowlist); `/internal/*`, `/healthz`, `/readyz`, `/metrics` are reachable only from inside the docker network.

| method | path | description |
|---|---|---|
| GET | `/api/v1/auth/google/login?redirect=/x` | state + PKCE verifier → signed cookie `abet_oauth` (Path `/public/api/v1/auth`, 10 min) → 302 to Google. `503 {"error":"google oauth not configured"}` when `GOOGLE_CLIENT_ID` is empty |
| GET | `/api/v1/auth/google/callback` | validates state, exchanges code, verifies the ID token, checks domain, calls main-service login-sync, creates session, sets `abet_session`, 302 |
| POST | `/api/v1/auth/logout` | revokes the session, clears the cookie, 204 |
| GET | `/internal/v1/auth/verify` | **internal only** forward-auth: 200 + `X-User-Id`/`X-User-Email`, or 401 `{"error":"unauthenticated"}` |
| GET | `/healthz`, `/readyz`, `/metrics` | ops probes / Prometheus (docker network only, not routed by the gateway) |

Full contract: `api/openapi/server/spec.yaml`. Swagger UI at `/swagger` only when `ENABLE_SWAGGER=true`; pprof only when `ENABLE_PPROF=true`.

## Behaviour notes

* Allowed users: ID-token `email_verified=true` **and** the lower-cased email domain exactly equals one of `ALLOWED_EMAIL_DOMAINS` (no suffix/subdomain match),
  **and** main-service `login-sync` answers 200 (user pre-provisioned). A rejection (domain, unverified or 403 from main-service) → `PUBLIC_BASE_URL/login?error=access_denied`; any other failure → `/login?error=login_failed`.
* `redirect` must start with a single `/` and contain no `\` or control characters, otherwise `/` is used.
* Session JWT: HS256, claims `sub` (user id), `email`, `sid` (session row id), `iat`, `exp`. Cookie `abet_session`: HttpOnly, SameSite=Lax, Path `/`, `Secure` = `COOKIE_SECURE`.
* Verify = JWT check + one primary-key lookup in `sessions` (not revoked, not expired, same user).
* Expired sessions older than 7 days are deleted on startup and hourly.
* The service starts without Google credentials (the rest of the stack works; only login is unavailable). `SESSION_SECRET` shorter than 32 chars aborts startup.

## Configuration (environment variables)

`PORT` (8080) · `DB_HOST` · `DB_PORT` (5432) · `DB_USER` · `DB_PASSWORD` · `DB_NAME` · `DB_SSLMODE` (disable; set on the pgx config directly, never inside a URL, so any password character is allowed) · `GOOGLE_CLIENT_ID` · `GOOGLE_CLIENT_SECRET` · `GOOGLE_REDIRECT_URL` · `ALLOWED_EMAIL_DOMAINS` (comma separated) · `SESSION_SECRET` (>= 32 chars) ·
`SESSION_TTL_HOURS` (12) · `COOKIE_SECURE` (false) · `PUBLIC_BASE_URL` · `MAIN_SERVICE_URL` · `INTERNAL_API_TOKEN` · `LOG_LEVEL` (info) · `ENABLE_SWAGGER` (false) · `ENABLE_PPROF` (false).
In the compose stack they all come from the root `.env` (see `.env.template`).

## Create the Google OAuth client

1. Google Cloud Console → *APIs & Services* → *OAuth consent screen*: user type **Internal** if your Google Workspace allows it, otherwise External + add test users.
2. *Credentials* → *Create credentials* → *OAuth client ID* → application type **Web application**.
3. **Authorized redirect URI** = `${PUBLIC_BASE_URL}/public/api/v1/auth/google/callback`, e.g. `http://localhost:8000/public/api/v1/auth/google/callback` for dev
   (must be identical to `GOOGLE_REDIRECT_URL` in `.env`). For production use the https origin and set `COOKIE_SECURE=true`.
4. Copy the client id/secret into `.env` (`GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`) and restart: `make restart s=auth-service`.

## Develop

```bash
make test          # go test ./... -count=1
make mock          # regenerate gomock files (go run go.uber.org/mock/mockgen@v0.6.0)
make sqlc          # regenerate sqlc code via docker (sqlc/sqlc:1.29.0); schema = db/migrations
go vet ./... && go build ./...
```

Run with the whole stack from the repository root: `make env && make dev` (air hot reload, source bind-mounted).
Run alone: set the variables above (at least `DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `SESSION_SECRET`) and `make run`.

Migrations (`db/migrations/*.sql`, golang-migrate, embedded) run automatically at startup.

## Dev session without Google (`cmd/devsession`)

For tests and demos you cannot complete the Google consent screen. `cmd/devsession` is a **separate CLI** (the production image builds only `./cmd/api`):
given the e-mail of a user that already exists in main-service it calls `login-sync`, inserts a session row in `auth_db` and prints
`abet_session=<jwt>` (signed with `SESSION_SECRET`, same claims/TTL as a real login) on stdout. It **refuses to run unless `APP_ENV=development`**
(set only in `compose.dev.yaml`, never in prod).

```bash
make seed                                        # example users: super1/admin1/admin2/instructor1/instructor2/staff1/staff2@chula.ac.th
make dev-session email=admin1@chula.ac.th        # → abet_session=eyJ...
curl -H "Cookie: $(make -s dev-session email=admin1@chula.ac.th)" http://localhost:8000/auth/api/v1/me
```
In a browser: DevTools → Application → Cookies → add `abet_session` (path `/`, HttpOnly optional) with the value after the `=`.

## Layout

```
cmd/api/main.go                      wiring, graceful shutdown, hourly cleanup
cmd/devsession/main.go               DEV-ONLY CLI: mint a session cookie for an existing user (APP_ENV=development)
config/config.go                     env only
db/migrations/ + db/migrations.go    embedded SQL migrations ; db/queries/ sqlc queries
internal/core/domain                 Session, Identity, EmailDomainPolicy, SanitizeRedirect, OAuthState, errors
internal/core/port                   inbound use-case + outbound ports (auth.go) ; mocks/ (gomock)
internal/application/commands        start login, google callback, logout, cleanup
internal/application/queries         verify session
internal/adapter/inbound/http        router, handler (auth, health, errors, oauth cookie), middleware, view
internal/adapter/outbound/           postgres (+sqlc), google (OIDC), jwt, mainclient (gobreaker)
```
