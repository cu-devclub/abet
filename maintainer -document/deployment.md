# Deployment guide

Back to the [documentation index](README.md).

## Prerequisites

| Need | Notes |
|---|---|
| Docker Engine with the Compose plugin (v2) | tested with Docker 29.x / Compose v2 |
| GNU Make | on Windows use Git Bash (GNU Make 4.4); the Makefile uses POSIX shell syntax |
| A Google Cloud project with an OAuth client | see [Google OAuth client](#google-oauth-client) |
| A hostname + TLS terminator for production | the stack serves plain HTTP only, see [HTTPS](#https) |
| Disk | the `portfolio-service` image installs a full TeX distribution (several GB); plan space for uploads (up to 300 MB each) and PDFs |
| Internet during build | pulls base images, Go/npm/pip dependencies |

You do **not** need Go, Node or Python on the server; everything is built inside Docker.

## `.env` reference

All configuration is one root `.env` file (template: [`.env.template`](../.env.template)). Compose passes the values to the containers.
Keys marked **secret** must be unique random values in production (`openssl rand -hex 32` is a good generator).

| Key | Default (template) | Meaning | Used by | Secret / rotate |
|---|---|---|---|---|
| `COMPOSE_PROJECT_NAME` | `abet` | prefix of containers, network and volumes (`abet_pgdata`, `abet_miniodata`, ...) | compose | no; **never change after first run** (you would get new, empty volumes) |
| `GATEWAY_HTTP_PORT` | `8000` | host port the gateway publishes (the only published port in prod) | compose, frontend dev (HMR) | no |
| `PUBLIC_BASE_URL` | `http://localhost:8000` | browser-facing origin; post-login redirect target and base of the default redirect URL | auth-service | no |
| `POSTGRES_USER` | `abet` | owner of both databases | postgres, auth, main | no |
| `POSTGRES_PASSWORD` | `change-me` | Postgres password. The services receive it as `DB_PASSWORD` and never put it in a URL | postgres, auth, main | **secret**; any characters work (see [special characters](#passwords-with-special-characters)) |
| `POSTGRES_HOST` / `POSTGRES_PORT` | `postgres` / `5432` | where the Go services reach Postgres | auth, main | no |
| `AUTH_DB_NAME` / `MAIN_DB_NAME` | `auth_db` / `main_db` | database names (created by `deploy/postgres/init-databases.sh`) | postgres, auth, main | no |
| `POSTGRES_HOST_PORT` | `5432` | **dev only**: host port for Postgres | compose.dev | no |
| `RABBITMQ_HOST` / `RABBITMQ_PORT` | `rabbitmq` / `5672` | AMQP endpoint inside the network | main, portfolio | no |
| `RABBITMQ_USER` | `abet` | RabbitMQ user (also management UI login) | rabbitmq, main, portfolio | no |
| `RABBITMQ_PASS` | `change-me` | RabbitMQ password (url-escaped when main-service builds its AMQP URI; plain parameter for the worker) | rabbitmq, main, portfolio | **secret**; any characters work |
| `RABBITMQ_VHOST` | `/` | virtual host | rabbitmq, main, portfolio | no |
| `RABBITMQ_QUEUE` | `course_build_queue` | job queue (main -> worker) | main, portfolio | no |
| `RABBITMQ_STATUS_QUEUE` | `course_status_queue` | result queue (worker -> main) | main, portfolio | no |
| `RABBITMQ_MGMT_HOST_PORT` | `15672` | **dev only**: host port of the management UI | compose.dev | no |
| `GOOGLE_CLIENT_ID` | empty | OAuth client id; empty => `GET /public/api/v1/auth/google/login` answers 503 | auth | no |
| `GOOGLE_CLIENT_SECRET` | empty | OAuth client secret | auth | **secret** (rotate in Google Cloud Console) |
| `GOOGLE_REDIRECT_URL` | `http://localhost:8000/public/api/v1/auth/google/callback` | must equal an Authorized redirect URI of the OAuth client | auth | no |
| `ALLOWED_EMAIL_DOMAINS` | `chula.ac.th` | comma-separated, **exact** domain match (case-insensitive; `student.chula.ac.th` is *not* matched by `chula.ac.th`). The Google `hd` hint uses the first entry. Also validates e-mails of users you create | auth, main | no |
| `SESSION_SECRET` | `change-me-to-64-random-chars` | HS256 key of the session JWT and the OAuth state cookie; **min. 32 characters or auth-service refuses to start** | auth | **secret**; rotating logs everybody out |
| `SESSION_TTL_HOURS` | `12` | session lifetime | auth | no |
| `COOKIE_SECURE` | `false` | sets the `Secure` flag on the session cookie; **`true` whenever users reach the site via https** | auth | no |
| `INTERNAL_API_TOKEN` | `change-me` | shared secret for auth-service -> main-service `/internal/*` (empty token => every internal call rejected, nobody can log in) | auth, main | **secret** |
| `SUPER_ADMIN_EMAILS` | empty | comma-separated; upserted as `super_admin` every time main-service starts | main | no |
| `MINIO_ROOT_USER` / `MINIO_ROOT_PASSWORD` | `abetminio` / `change-me-minio-root` | MinIO root account = **console login**. The password needs at least 12 characters in production (MinIO refuses shorter ones); any characters work if double-quoted ([special characters](#passwords-with-special-characters)). Only `minio` and `minio-init` receive it | minio, minio-init | **secret**; see rotation note below |
| `MINIO_ENDPOINT` | `minio:9000` | S3 endpoint inside the compose network (host:port, no scheme); never published | main, portfolio | no |
| `MINIO_USE_SSL` | `false` | TLS between the services and MinIO (keep `false` on the compose network) | main, portfolio | no |
| `MINIO_BUCKET` | `abet` | the single bucket for all files; created by `minio-init` (main-service also creates it at startup if missing) | main, portfolio, minio-init | no; **do not change after first run** (existing files stay in the old bucket) |
| `MINIO_ACCESS_KEY` / `MINIO_SECRET_KEY` | `abet-app` / `change-me-minio-svc` | **service account** used by main-service and the worker. `minio-init` creates it with a policy that allows `s3:*` only on `MINIO_BUCKET` and its objects. Secret length 8-40 characters | main, portfolio, minio-init | **secret** |
| `MINIO_CONSOLE_HOST_PORT` | `9001` | host port of the MinIO web console (published in dev **and** prod, see [MinIO console](operations.md#minio-console)) | compose | no |
| `MINIO_API_HOST_PORT` | unset | **dev only**, optional: publishes the S3 API (9000) for debugging; the prod override never publishes it | compose (dev) | no |
| `MAX_UPLOAD_MB` | `300` | max portfolio zip (150 MB SRS limit x 2 margin, D11). main-service answers 413 above it | main, gateway | no |
| `MAX_COVER_MB` | `20` | max cover PDF | main, gateway | no |
| `MAX_IMPORT_MB` | `5` | max mapping CSV of the offerings import | main, gateway | no |
| `BUILD_TIMEOUT_MINUTES` | `60` | a submission still waiting for the worker after this long becomes `failed` / `BUILD_TIMEOUT` (see [Stuck builds](operations.md#stuck-builds-never-processing-forever)) | main | no |
| `RECONCILE_INTERVAL_SECONDS` | `60` | how often main-service looks for timed-out builds, checks the build queue and (hourly) cleans storage | main | no |
| `PORTFOLIO_WORKER_REPLICAS` | `2` | number of portfolio-service worker containers (each builds one PDF at a time) | compose | no |
| `DELETE_ZIP_DEFAULT` | `true` | worker default for deleting the input zip. main-service always sends `delete_zip_on_completion: true` in the job, so this has no practical effect | portfolio | no |
| `LOG_LEVEL` | `INFO` | log level of all services (Go lower-cases it) | all | no |

Variables that are set by compose itself and are **not** in `.env`: `PORT=8080`, `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` (mapped from `POSTGRES_*` and `AUTH_DB_NAME` / `MAIN_DB_NAME`) and `DB_SSLMODE=disable` for both Go services (there is no `DATABASE_URL` any more), `MAIN_SERVICE_URL=http://main-service:8080`,
`INSTRUCTOR_PACKAGE_DIR=/app/assets/instructor-package`, `UPLOAD_TMP_DIR=/tmp/abet-uploads` (main-service scratch dir, the `uploadtmp` volume; uploads are spooled there for validation and always removed), and dev-only `ENABLE_SWAGGER=true`, `APP_ENV=development`,
`VITE_HMR_CLIENT_PORT`, `VITE_USE_POLLING`. `ENABLE_PPROF` / `ENABLE_SWAGGER` default to `false` in both Go services (never enabled in prod).

All size limits come from `.env` alone; nothing is hard-coded any more:

* main-service reads `MAX_UPLOAD_MB`, `MAX_COVER_MB`, `MAX_IMPORT_MB` and serves them (with `BUILD_TIMEOUT_MINUTES`) at `GET /api/v1/config`; the frontend reads that endpoint (`useAppConfig()`) for its client-side checks and texts.
* The gateway body limit is computed when the container starts: `deploy/api-gateway/entrypoint.sh` sets `GATEWAY_MAX_BODY_MB = max(MAX_UPLOAD_MB, MAX_COVER_MB, MAX_IMPORT_MB) + 20`, which `api-gateway/config.yaml` uses as `client_max_body_size` (default `320m`). The entrypoint prints the values it used.

To change a limit edit `.env` and run `make dev` / `make prod` again (recreates main-service and the gateway). A TLS terminator in front of the gateway needs a limit of at least `GATEWAY_MAX_BODY_MB` as well.

### Secrets to generate/rotate

`POSTGRES_PASSWORD`, `RABBITMQ_PASS`, `MINIO_ROOT_PASSWORD`, `MINIO_SECRET_KEY`, `SESSION_SECRET`, `INTERNAL_API_TOKEN`, `GOOGLE_CLIENT_SECRET`.
Caveats when rotating on a running system:

* `POSTGRES_PASSWORD` and `RABBITMQ_PASS` are applied by the database/broker **only when their volume is first created**. Changing `.env`
  later does not change the stored password; change it inside the service (`ALTER USER ... PASSWORD`, `rabbitmqctl change_password`) and then update `.env`
  and restart the services.
* `MINIO_ROOT_PASSWORD` is applied by MinIO **only when `miniodata` is first created**; change it inside MinIO (console *Identity* or `mc admin user`) and then update `.env`. `MINIO_SECRET_KEY` is re-applied by every `minio-init` run (`mc admin user add` resets the secret): edit `.env`, then `docker compose ... up -d` (re-runs `minio-init`, recreates main-service and the workers).
* `SESSION_SECRET` / `INTERNAL_API_TOKEN`: edit `.env`, then `docker compose -f compose.yaml -f compose.prod.yaml up -d` (recreates the changed containers). Sessions become invalid.

### Passwords with special characters

`POSTGRES_PASSWORD`, `RABBITMQ_PASS`, `MINIO_ROOT_PASSWORD` and `MINIO_SECRET_KEY` may contain any character (`@ : / ? # % & = ' " \` and spaces). The Go services get the database settings as separate values
(`DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`) and build the connection programmatically, so the password is never interpolated into a URL;
main-service escapes user and password when it builds the AMQP URI, and the worker passes them as plain parameters. Only the dotenv syntax matters:
**put a value that contains `#`, a space, `$` or quotes in double quotes** in `.env`, for example `POSTGRES_PASSWORD="p@ss:w/rd?#%&='\"\\ x"` (escape `"` and `\` inside double quotes;
a `$` must be written `$$`). Use `make config` to check what compose hands to the containers.

Postgres and RabbitMQ apply their password only when the volume is first created (see above), so test a new password on a fresh volume (`make clean`, which deletes data).

## Google OAuth client

1. Google Cloud Console -> *APIs & Services* -> *OAuth consent screen*: choose **Internal** (Workspace) or External and add test users.
2. *Credentials* -> *Create credentials* -> *OAuth client ID* -> type **Web application**.
3. **Authorized redirect URI** = `<PUBLIC_BASE_URL>/public/api/v1/auth/google/callback`, character-for-character equal to `GOOGLE_REDIRECT_URL`
   (e.g. `https://abet.example.ac.th/public/api/v1/auth/google/callback`). No authorized JavaScript origin is needed.
4. Put the client id/secret into `.env`.
5. Make sure `ALLOWED_EMAIL_DOMAINS` contains the domain of every account that must log in (e.g. add `student.chula.ac.th` for student test accounts).
   Users must also be provisioned in the system, otherwise they get Access Denied.

## First run

```bash
make env                 # copies .env.template to .env (does nothing if .env exists)
$EDITOR .env             # replace every change-me; set GOOGLE_*, PUBLIC_BASE_URL, GOOGLE_REDIRECT_URL,
                         # SUPER_ADMIN_EMAILS, COOKIE_SECURE (true for https), ALLOWED_EMAIL_DOMAINS
make config              # optional: validates the merged compose files (dev and prod)
make prod                # docker compose -f compose.yaml -f compose.prod.yaml up -d --build
docker compose -f compose.yaml -f compose.prod.yaml ps    # all services should become healthy
```

The first build is slow (TeX installation, Go and npm dependencies). Startup order is handled by compose health checks: Postgres creates `auth_db` and `main_db`,
`minio` starts, the one-shot `minio-init` creates the bucket and the bucket-limited service account (idempotent; main-service and the worker wait for it), each Go service runs its embedded migrations before it starts serving.
Then open `PUBLIC_BASE_URL` and run the [manual Google-login checklist](operations.md#manual-google-login-checklist-acceptance-test-after-a-deploy).

### First super admin

Set `SUPER_ADMIN_EMAILS=first.admin@chula.ac.th,second.admin@chula.ac.th` **before** (or after, then restart `main-service`) the first start. On every main-service startup
each listed address is upserted as a `super_admin` (case-insensitive). The person can then sign in with Google and create departments, curricula and the other users.
Removing an address from the variable does **not** demote the user; change the role in the Users page (another super admin) or the database.

### HTTPS

The gateway speaks plain HTTP (APISIX, container port 9080 published as `GATEWAY_HTTP_PORT`). Put a TLS terminator (nginx, Caddy, a load balancer) in front, forward to that port, and:

* set `PUBLIC_BASE_URL=https://your.host`, `GOOGLE_REDIRECT_URL=https://your.host/public/api/v1/auth/google/callback` (and register it in Google);
* set `COOKIE_SECURE=true` (otherwise the session cookie is also sent over http; with `true` browsers will not send it over plain http at all);
* the terminator must pass `Host`, allow request bodies of at least `GATEWAY_MAX_BODY_MB` (320 MB with the default limits), and use read/send timeouts of at least 300 s (large uploads, bulk zip downloads);
* publish only the TLS port to the world; bind the gateway port to localhost if the terminator runs on the same host (edit the `ports:` mapping of `api-gateway` in `compose.prod.yaml`, e.g. `127.0.0.1:8000:9080`).

Then `make restart`-equivalent: `docker compose -f compose.yaml -f compose.prod.yaml up -d` to apply the changed environment.

## Backups

State lives in three places: the two Postgres databases, the MinIO data and `.env`. `make clean` and `down -v` delete it; the plain `down` targets keep it.

```bash
DC="docker compose -f compose.yaml -f compose.prod.yaml"
mkdir -p backup

# 1) databases (custom format, restorable with pg_restore)
$DC exec -T postgres sh -c 'pg_dump -U "$POSTGRES_USER" -Fc "$MAIN_DB_NAME"' > backup/main_db.dump
$DC exec -T postgres sh -c 'pg_dump -U "$POSTGRES_USER" -Fc "$AUTH_DB_NAME"' > backup/auth_db.dump

# 2) the MinIO bucket (generated PDFs, covers, pending uploads), copied with `mc mirror` (consistent while the stack runs;
#    do not tar the miniodata volume of a running MinIO). The mc image is the one `minio-init` uses; --no-deps keeps it from re-running the init.
mkdir -p backup/files
$DC run --rm --no-deps --user root -v "$PWD/backup/files":/backup --entrypoint sh minio-init -c \
  'export MC_CONFIG_DIR=/tmp/mc; mc alias set local http://minio:9000 "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD" && mc mirror --overwrite "local/$MINIO_BUCKET" /backup'
```

* `main_db` is the essential one (users, org structure, courses, access, submission history). `auth_db` only holds login sessions: restoring it is optional (users just sign in again), but it is cheap to include.
* The bucket holds `portfolios/<id>.pdf` (downloadable results) and `covers/<curriculum_id>.pdf` (plus transient `uploads/` and `outputs/`). Without it, rows in `main_db` say "generated" but downloads return 404 and covers are missing; the PDFs can be rebuilt only by uploading again.
* `.env` must be backed up separately (it is not in git).

Restore (stack stopped except postgres): `docker compose ... exec -T postgres pg_restore -U <user> -d <db> --clean --if-exists < backup/main_db.dump` and
the same `$DC run ... minio-init` command with the mirror reversed (`mc mirror --overwrite /backup "local/$MINIO_BUCKET"`; `minio` and `minio-init` must have run once so the bucket exists). Take both from the same moment to keep DB and files consistent. Backups of the `mc mirror` command are documented here but were not exercised in the acceptance run; test a restore on a scratch project before you rely on it.

## Upgrades

```bash
git pull                                   # or copy the new sources
# read the release notes; take the backups above
make prod                                  # rebuilds changed images and recreates changed containers
docker compose -f compose.yaml -f compose.prod.yaml ps
```

* Database migrations are embedded and run automatically at service start (forward only). A rollback of code that already migrated needs the database restored from backup.
* New `.env` keys: compare with `.env.template` (`diff <(grep -o '^[A-Z_]*' .env.template | sort) <(grep -o '^[A-Z_]*' .env | sort)`).
* Postgres/RabbitMQ images are pinned to major tags (`postgres:17-alpine`, `rabbitmq:4-management-alpine`); a Postgres major upgrade needs a dump/restore.
* Brief downtime: submissions in `queued` state survive a restart (durable queues), the status consumer reconnects automatically.

## Monitoring (optional)

`MONITORING_ENABLED=false` in `.env` is the default. Set it to `true` (or run `make dev MON=1` / `make prod MON=1`) to start the `monitoring` compose profile (Prometheus + Grafana, Elasticsearch + Kibana + APM Server + Filebeat, exporters)
and to switch tracing on in auth-service, main-service and the worker (`OTEL_ENABLED` follows `MONITORING_ENABLED`). Extra `.env` keys: `ELASTIC_PASSWORD`, `KIBANA_SYSTEM_PASSWORD`, `ES_INGEST_PASSWORD`, `ES_READER_PASSWORD`,
`KIBANA_ENCRYPTION_KEY` (>= 32 chars), `APM_SECRET_TOKEN`, `ES_HEAP_MB` (1024), `GRAFANA_ADMIN_USER`, `GRAFANA_ADMIN_PASSWORD`, `POSTGRES_MONITOR_PASSWORD`, `PROMETHEUS_RETENTION` (15d), `OTEL_SAMPLE_RATIO` (1.0),
`PROMETHEUS_HOST_PORT` (dev only). Grafana is then `<PUBLIC_BASE_URL>/grafana/`, Kibana `<PUBLIC_BASE_URL>/kibana/`. Plan about 4 GB extra RAM. Everything is described in [monitoring.md](monitoring.md).

## Makefile targets

Run `make help` for the list. **`ENV=dev|prod` (default `dev`) selects the stack for every ops target**, e.g. `make logs ENV=prod s=main-service`. It also selects the compose files and the
compose project name: dev uses project `<COMPOSE_PROJECT_NAME>-dev` (e.g. `abet-dev`), prod uses `<COMPOSE_PROJECT_NAME>` (e.g. `abet`). Dev and prod therefore have **separate containers, networks
and volumes** and can exist side by side (the dev volumes are `abet-dev_pgdata`, ...; the prod ones `abet_pgdata`, ...). Only `COMPOSE_PROJECT_NAME` is read from `.env` by the Makefile (a plain `sed`, so passwords with `#` or `$` are not mangled).
Targets marked *dev only* refuse to run with `ENV=prod`. Targets that take a service use `s=`.

| Target | What it does |
|---|---|
| `help` (default) | list targets and the ENV usage |
| `env` | create `.env` from `.env.template` if missing |
| `dev` | `ENV=dev` start: `docker compose -f compose.yaml -f compose.dev.yaml up -d --build` (hot reload, published ports) |
| `dev-logs` | shortcut for `make logs ENV=dev` |
| `prod` | `ENV=prod` start: `docker compose -f compose.yaml -f compose.prod.yaml up -d --build` |
| `prod-down` | shortcut for `make down ENV=prod` (keeps volumes) |
| `down` | stop the stack of `ENV`, keep volumes |
| `logs s=<service>` | follow the last 200 lines of one service of `ENV` (all services when `s` is empty) |
| `ps` | status of the `ENV` stack |
| `restart s=<service>` | restart one service of `ENV` (requires `s`) |
| `worker-restart` | restart all `portfolio-service` workers of `ENV` (e.g. after a hung build) |
| `build` | build all images with the `ENV` files |
| `test` | `go test ./... -count=1` in `auth-service` and `main-service`, `npm test` in `frontend` |
| `lint` | `go vet ./...` in both Go services, `npm run lint` in `frontend` |
| `db-shell` | `psql` into `main_db` of the `ENV` stack |
| `dev-session email=<e-mail>` | **dev only** print an `abet_session=<jwt>` cookie for an existing user (needs `APP_ENV=development`) |
| `seed` | **dev only** load `deploy/dev/seed.sql` into `main_db` |
| `e2e` | **dev only** run the Playwright suite against the running dev stack (`BASE_URL` default `http://localhost:8000`) |
| `mon-urls` | print the Grafana and Kibana URLs and the login user names (never passwords) |
| `mon-prune days=N` | ask, then delete logs and traces older than N days from Elasticsearch (never automatic); see [monitoring.md](monitoring.md#retention-forever-and-manual-cleanup) |
| `MON=1` (variable) | `make dev MON=1` / `make prod MON=1` start with the monitoring profile even when `MONITORING_ENABLED=false`; `down`, `clean`, `ps`, `logs` always include it |
| `minio-console` | print the MinIO console URL and the login hint (`MINIO_ROOT_USER` / `MINIO_ROOT_PASSWORD`); same port in dev and prod |
| `rabbit-ui` | print the RabbitMQ management URL and the login hint (dev stack) |
| `clean` | **DANGER** names the stack, asks for `yes`, then `down -v --remove-orphans`: deletes the Postgres, RabbitMQ and MinIO (`miniodata`) volumes of `ENV` only |
| `config` | validate the merged compose files of `ENV` (`dev config OK` / `prod config OK`) |

The volume names are `<project>_pgdata` / `<project>_miniodata`: `abet_miniodata` for prod, `abet-dev_miniodata` for dev.
