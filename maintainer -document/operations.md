# Operations guide

Back to the [documentation index](README.md).

In the commands below `DC` stands for the compose file pair of the stack you run:

```bash
DC="docker compose -p abet -f compose.yaml -f compose.prod.yaml"   # production (project = COMPOSE_PROJECT_NAME)
# dev: DC="docker compose -p abet-dev -f compose.yaml -f compose.dev.yaml"
```

The Makefile does the same for you: every ops target takes `ENV=dev|prod` (default `dev`), e.g. `make ps ENV=prod`, `make logs ENV=prod s=main-service`, `make worker-restart ENV=prod`.
Dev and prod are separate compose projects (`<name>-dev` and `<name>`) with separate containers and volumes; see [deployment.md](deployment.md#makefile-targets).

## Health and status

| What | How | Healthy answer |
|---|---|---|
| all containers | `$DC ps` (`make ps ENV=prod`; dev: `make ps`) | every service `healthy` (frontend/portfolio-service have no or simple checks); `minio-init` has exited 0 (`minio` healthy) |
| gateway from outside | `curl -sI http://HOST:PORT/` | `200` (SPA). The auth-service probes are **not** routed: `/public/healthz`, `/public/readyz`, `/public/metrics`, `/public/health` answer `404 {"error":"not found"}` |
| auth-service (inside the network only) | `$DC exec auth-service wget -qO- http://localhost:8080/healthz` and `.../readyz` | `200` (`readyz` pings `auth_db`); `/metrics` = Prometheus |
| main-service (inside the network only) | `$DC exec main-service wget -qO- http://localhost:8080/readyz` | 200 (pings `main_db` and the MinIO bucket: `"storage":"up"`; `503` + `"storage":"down"` when MinIO is unreachable) with `"build_queue":{"consumers":2,"messages":0,...}` = workers connected / jobs waiting; `/healthz` = liveness; `/metrics` = Prometheus |
| forward-auth | `curl -si http://HOST:PORT/auth/api/v1/me` (no cookie) | `401 {"error":"unauthenticated"}` |
| SPA | `curl -sI http://HOST:PORT/` | 200 |
| RabbitMQ | dev: management UI (`make rabbit-ui`); `$DC exec rabbitmq rabbitmq-diagnostics -q ping` | `Ping succeeded` |
| Postgres | `$DC exec postgres pg_isready -U abet` | `accepting connections` |
| worker | `$DC logs --tail=50 portfolio-service` | `Received message payload` / `Job processed` lines when jobs arrive and no repeated connection errors |

The main-service and auth-service container health checks call `/healthz` with `wget` every 10 s; the gateway only starts after both are healthy.
The old auth-service HTML page `/health` was removed; ops endpoints are reachable only inside the docker network — see [security-and-permissions.md](security-and-permissions.md#what-is-exposed-in-production).

## Logs

| Stack | Command |
|---|---|
| one service | `make logs s=main-service` (dev) or `make logs ENV=prod s=main-service` (last 200 lines, follows) |
| all | `make dev-logs` / `make logs ENV=prod` |
| without make | `$DC logs -f --tail=200 main-service` (omit the service for all) |

Go services log JSON lines (`log/slog`: `time`, `level`, `msg`, request fields). Set `LOG_LEVEL=debug` in `.env` and recreate the services for more detail.
Useful messages: main-service `session`/consumer reconnect lines with backoff 1 s..30 s (RabbitMQ unreachable), auth-service `session cleanup finished`, the gateway `forward-auth` warnings (`uri with no TLS` at startup is expected),
worker `[job_id] Starting stateless job`, `Pipeline completed in Ns`, `Using curriculum cover page`.

## Monitoring and tracing

Optional Grafana / Kibana stack, see [monitoring.md](monitoring.md). Quick reference: `make mon-urls` prints the URLs; Kibana **Observability -> APM** shows one trace per request across gateway,
auth-service, main-service, RabbitMQ and the worker; in Discover (`ABET logs`) search `trace.id : "<id>"` for the log lines of one trace. The Go services print `trace_id` / `span_id` in their JSON logs, the worker appends ` trace_id=<hex>`.
Elasticsearch never deletes anything by itself; at 95 % disk usage (flood-stage watermark) its indices become read-only and logs/traces are rejected until space is freed (`make mon-prune days=N`).
With monitoring off the services make no connection to the trace endpoint and log nothing about it.

## RabbitMQ management UI (dev)

`make rabbit-ui` prints `http://localhost:15672` (port from `RABBITMQ_MGMT_HOST_PORT`); log in with `RABBITMQ_USER` / `RABBITMQ_PASS`. In production the UI port is **not published**.
To use it there, temporarily add `ports: ["127.0.0.1:15672:15672"]` to the `rabbitmq` service in a local override file, or run `$DC exec rabbitmq rabbitmqctl list_queues name messages consumers`.

## Queues and message formats

Both queues are durable, on the default exchange, routing key = queue name, declared by producer and consumer with identical arguments (`durable=true`, nothing else — a mismatch makes RabbitMQ close the channel).

**`course_build_queue`** (`RABBITMQ_QUEUE`): main-service -> worker. Persistent, `application/json`:

```json
{
  "job_id": "7a0b0d8c-6c3e-4f6c-9d0e-0d6a4f1f3b11",
  "program_name": "cs-b-2026",
  "storage": "s3",
  "zip_key": "uploads/<submission_id>.zip",
  "cover_key": "covers/<curriculum_id>.pdf",
  "output_prefix": "outputs/<submission_id>/",
  "delete_zip_on_completion": true
}
```

* `job_id` = `submissions.job_id` (not the submission id). `program_name` = curriculum code lower-cased, characters outside `[a-z0-9-]` replaced by `-`, then `-<academic_year>`.
* Payload v2: object **keys** in the bucket `MINIO_BUCKET`, no filesystem paths. `cover_key` is present only if that object exists when the job is published. There is no `callback_url`. The worker still understands the old path-based payload (`zip_path`, `output_dir`, `cover_pdf_path`) for backward compatibility, but main-service no longer sends it.

**`course_status_queue`** (`RABBITMQ_STATUS_QUEUE`): worker -> main-service. Persistent JSON, one event per job:

```json
{"status":"success","job_id":"...","program_name":"cs-b-2026","elapsed_seconds":16.2,
 "exported_pdfs":[],"exported_reports":[],"exported_pdf_keys":["outputs/<sid>/2301101-main.pdf"],"exported_report_keys":["outputs/<sid>/2301101-report.md"],"timestamp":"2026-10-09T10:00:00Z"}
{"status":"failed","job_id":"...","error":"<message text>","traceback":"...","timestamp":"..."}
```

main-service reads only `exported_pdf_keys` (the old `exported_pdfs` list is ignored; only keys below `outputs/<submission_id>/` are accepted) and consumes with manual ack after the database commit (prefetch 10). Unknown `job_id` -> acked and a warning logged. The worker never reports "processing", so a submission is `queued` from upload until the result arrives (or until the [build timeout](#stuck-builds-never-processing-forever) fails it). A `success` event that arrives after the timeout still wins; a `failed` event for a submission that is already finished is ignored.
The worker `nack`s a failed job **without requeue** (after publishing the `failed` event) so there are no infinite retries.

## Storage layout and cleanup

All files are objects in ONE MinIO bucket (`MINIO_BUCKET`, default `abet`, volume `<COMPOSE_PROJECT_NAME>_miniodata`). There is no shared filesystem volume any more: main-service and the worker talk to MinIO over S3 with the service account `MINIO_ACCESS_KEY` (policy limited to this bucket); browsers never reach MinIO, downloads stream through main-service after its authorization check.

```
<bucket>/
├── uploads/<submission_id>.zip          written by main-service AFTER validation passed; deleted by the worker on success (delete_zip_on_completion) or by main-service on failure / publish error / offering delete
├── outputs/<submission_id>/<files>      written by the WORKER (PDFs + reports); main-service copies the PDF to portfolios/ then deletes the whole prefix
├── portfolios/<submission_id>.pdf       the final PDF (server-side copy); ONLY the newest generated submission of an offering keeps its object
└── covers/<curriculum_id>.pdf           the curriculum cover (replaced on upload, deleted with the curriculum)
```

An invalid or non-zip upload never creates an object: the zip is validated from a temp file (`UPLOAD_TMP_DIR`, the `uploadtmp` volume, always cleaned) before it is put into the bucket.

Automatic cleanup rules:

| Event | Objects removed |
|---|---|
| validation finds errors (`invalid`) or the file is not a zip | nothing was stored |
| publish failure, worker `failed` | `uploads/<id>.zip` (and `outputs/<id>/` for worker results) |
| worker `success` | `uploads/<id>.zip` (worker, `delete_zip_on_completion`), the `outputs/<id>/` prefix, `portfolios/<id>.pdf` of **older** generated submissions of the same offering (their rows stay, `pdf_size_bytes` becomes NULL) |
| offering deleted (single, bulk, or CSV import delete) | `uploads/`, `portfolios/`, `outputs/` objects of all its submissions, after the DB commit, best effort |
| curriculum deleted | `covers/<id>.pdf` |

A **janitor** inside main-service runs at startup and then hourly (same tick as the reconciler) and removes leftovers of crashes or lost events. It lists `uploads/`, `outputs/` and `portfolios/` (`List` with `LastModified`) and deletes `uploads/<id>.zip` and the `outputs/<id>/` group (as old as its newest object) whose submission is finished (`generated`/`failed`/`invalid`) or unknown, and `portfolios/<id>.pdf` without a `generated` row, in each case only when older than 1 hour. It never touches a `queued` submission and never `covers/`; object names that are not `<uuid>.zip` / `<uuid>/` / `<uuid>.pdf` are left alone. It logs `storage janitor finished` with the counts. Manual cleanup is rarely needed; never delete `portfolios/` or `covers/` objects that the database still references.

### MinIO console

`make minio-console` prints the URL (`http://localhost:<MINIO_CONSOLE_HOST_PORT>`, default 9001; same in dev and prod). Log in with `MINIO_ROOT_USER` / `MINIO_ROOT_PASSWORD` from `.env`. *Object Browser* -> bucket `abet` shows the layout above; after a successful build `uploads/` is empty, `outputs/` is empty and `portfolios/<id>.pdf` exists. The same view from a shell (the `mc` image used by `minio-init`):

```bash
$DC run --rm --no-deps --entrypoint sh minio-init -c 'export MC_CONFIG_DIR=/tmp/mc; mc alias set local http://minio:9000 "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD" >/dev/null && mc ls -r "local/$MINIO_BUCKET" && mc du "local/$MINIO_BUCKET"'
```

Security: see [security-and-permissions.md](security-and-permissions.md#what-is-exposed-in-production) (the console is plain HTTP with root login; restrict it).

## Common failures and fixes

| Symptom | Likely cause | Fix |
|---|---|---|
| An upload shows **Processing** for a long time | worker down/restarting, a long or hung LaTeX build, a backlog, or a lost message | It cannot stay forever: after `BUILD_TIMEOUT_MINUTES` the row becomes **Failed** with issue `BUILD_TIMEOUT` and the user can upload again. See [Stuck builds](#stuck-builds-never-processing-forever) |
| Upload becomes **Failed** with issue **BUILD_TIMEOUT** ("PDF generation timed out") | the build did not finish within `BUILD_TIMEOUT_MINUTES` (workers down, overloaded, or a hung build) | check `/readyz` `build_queue.consumers` and `$DC logs portfolio-service`; `make worker-restart ENV=prod`; the user uploads again. If the worker finishes late the row flips to Generated by itself |
| Upload answers 201 with status `failed` and issue **QUEUE_UNAVAILABLE** | RabbitMQ was down when publishing (files already cleaned) | start RabbitMQ; ask the user to upload again. main-service reconnects on its own |
| Worker restarts in a loop, "Failed to ... RabbitMQ" | wrong `RABBITMQ_*` in `.env`, or password changed after the volume existed | fix `.env` / the broker user ([deployment.md](deployment.md#secrets-to-generaterotate)); the shim logs `entrypoint: settings.json rewritten from environment` at each start |
| Build `failed` with **PDF generation failed** (BUILD_FAILED) | the package passed our validator but LaTeX/PDF tooling failed (e.g. a lecture PDF that is not really a PDF, corrupt file) | read the issue description (last 2000 chars of the worker error) and `$DC logs portfolio-service`; user fixes the file and re-uploads |
| Build `failed` with several `BUILD_VALIDATION` issues | the worker's own validation found problems the immediate validator did not | same as above; if it happens for packages that look correct, compare validator and worker rules ([business-rules.md](business-rules.md#validator-versus-worker)) |
| Download returns 404 although status shows Generated | `portfolios/<id>.pdf` object missing (bucket restored without matching files, manual deletion in the console) | restore the bucket (`mc mirror`) from the same backup moment or ask for a new upload |
| `/readyz` answers 503 `"storage":"down"`, uploads and downloads fail with 500 | MinIO is down, `MINIO_ENDPOINT` / service-account credentials are wrong, or the bucket was deleted | `$DC ps minio minio-init`, `$DC logs minio minio-init main-service`; re-run `$DC up -d minio-init` to recreate the bucket and the service account; fix `.env` and recreate main-service |
| Build `failed` with text starting `Storage error:` | the worker could not read the zip or write its outputs in MinIO | same checks as above, on `portfolio-service` logs |
| Upload fails with 413 `file exceeds N MB limit` | zip larger than `MAX_UPLOAD_MB` (or an HTML 413 page: the TLS terminator in front of the gateway has a smaller body limit; the gateway limit itself follows `.env` automatically) | reduce the package or raise `MAX_UPLOAD_MB` in `.env` and recreate ([deployment.md](deployment.md#env-reference)) |
| Disk full | `miniodata` growth (PDFs, leftovers in `uploads/` / `outputs/`), Docker images and build cache, the TeX image | `docker system df`; `docker builder prune`; check the bucket size in the MinIO console and clean orphans in `uploads/` and `outputs/` as above; extend the disk. When the disk is full uploads fail with 500 and the worker fails jobs: afterwards the timeout fails any stuck `queued` rows by itself |
| Everybody gets 401 / is logged out | `SESSION_SECRET` changed, `auth_db` reset, session expired (`SESSION_TTL_HOURS`) | sign in again |
| Login button shows `google oauth not configured` (503) | `GOOGLE_CLIENT_ID` empty | fill `.env`, recreate auth-service |
| Google says `redirect_uri_mismatch` | `GOOGLE_REDIRECT_URL` differs from the authorized URI in Google Cloud (scheme, host, port, path) | make them identical |
| Login succeeds but you land on `/login` again | the session cookie is not stored/sent: `COOKIE_SECURE=true` while the site is opened over plain http, or `PUBLIC_BASE_URL` points to a different host/scheme than the one in the address bar | align `PUBLIC_BASE_URL`, `GOOGLE_REDIRECT_URL`, `COOKIE_SECURE` and the real scheme/host |
| main-service refuses to start | DB not reachable/migration error, or an unparseable value | `$DC logs main-service`; auth-service additionally aborts if `SESSION_SECRET` < 32 chars |

### Login error meanings (`/login?error=...`)

| Code | Shown as | Causes |
|---|---|---|
| `access_denied` | "Access Denied" modal | the Google account's e-mail is not verified; its domain is not in `ALLOWED_EMAIL_DOMAINS` (exact match: `student.chula.ac.th` != `chula.ac.th`); or main-service answered 403 `not provisioned` (no row in `users` for that e-mail). Fix: add the user in the Users page / `SUPER_ADMIN_EMAILS`, or the domain to `ALLOWED_EMAIL_DOMAINS` |
| `login_failed` | generic "could not sign in" | the user cancelled at Google or Google returned an error; the `abet_oauth` state cookie is missing/expired (> 10 min) or does not match (blocked cookies, two tabs, wrong `PUBLIC_BASE_URL` host); code exchange or ID-token verification failed (wrong client id/secret, clock skew); main-service unavailable or its login-sync call failed (check `INTERNAL_API_TOKEN` is equal in both services); session could not be stored. Look at the auth-service log |

### Requeue a job / repair stuck submissions

Normally nothing has to be repaired by hand: the reconciler fails stuck rows. Use this only to re-publish a job on purpose or to fail rows earlier than the timeout. Find the stuck rows (older than, say, one hour):

```bash
$DC exec -T postgres sh -c 'psql -U "$POSTGRES_USER" -d "$MAIN_DB_NAME"' <<'SQL'
SELECT s.id, s.job_id, s.created_at, s.original_filename, c.code AS course, o.academic_year, o.semester
FROM submissions s
JOIN course_offerings o ON o.id = s.offering_id JOIN courses c ON c.id = o.course_id
WHERE s.status = 'queued' AND s.created_at < now() - interval '1 hour' ORDER BY s.created_at;
SQL
```

* **Re-publish** (the zip must still exist: object `uploads/<id>.zip` in the bucket, see the MinIO console): with the worker running, in the RabbitMQ UI go to *Queues -> course_build_queue -> Publish message*, set property `content_type=application/json`, `delivery_mode=2` and send the job JSON shown above with `job_id` = `submissions.job_id`, `zip_key` / `output_prefix` (and `cover_key`) for that submission id, and the `program_name` of its curriculum (see formula above). The result event will update the submission normally.
* **Give up** (zip gone or the user should simply re-upload): mark it failed so the UI stops showing "Processing":

  ```sql
  UPDATE submissions SET status='failed', completed_at=now(), updated_at=now() WHERE id='<submission id>' AND status='queued';
  ```
  (the UI then shows a generic "unexpected error" because no issue row exists; the user uploads again. A later upload becomes the latest submission anyway.)
* Remove leftovers of that submission: objects `uploads/<id>.zip` and `outputs/<id>/` (MinIO console).

## Stuck builds (never "Processing" forever)

Failure modes: the worker container is down or restarting, a worker crashes mid-job, a status event is lost, main-service is down while the event arrives, LaTeX hangs and blocks a worker, or the queue has a backlog.
The system defends in layers:

1. **Durable queues and persistent messages** with publisher confirms; main-service consumes the status queue with manual ack, so results survive a main-service restart.
2. **Redelivery**: the worker acks a job only when it is finished; if the worker dies, RabbitMQ redelivers the job to another or the restarted worker.
3. **Parallel workers**: `PORTFOLIO_WORKER_REPLICAS` (default 2) worker containers, prefetch 1 each, `restart: unless-stopped`. One hung build blocks only one worker. Scale: set `PORTFOLIO_WORKER_REPLICAS=4` in `.env` and `make dev` / `make prod` (or `docker compose ... up -d --scale portfolio-service=4`). Each worker needs RAM for a TeX build.
4. **Build timeout reconciler** (main-service): at startup and every `RECONCILE_INTERVAL_SECONDS` (default 60) one atomic `UPDATE ... WHERE status='queued' AND created_at < now() - BUILD_TIMEOUT_MINUTES` fails the stuck submissions with a `BUILD_TIMEOUT` issue ("PDF generation timed out. Please upload again."). It logs a warning, counts `abet_submissions_timed_out_total`, **keeps** both the upload zip and the `outputs/<id>/` objects so a job still waiting in the queue (or being built) can finish late — a late success flips the row to Generated; the hourly janitor removes the leftovers of terminal submissions older than 1 h. The UI shows the row as Failed with a Re-upload button; while queued it shows "Processing · N min".
5. **Late success**: if the worker finishes after the timeout, its `success` event turns the `failed`/`BUILD_TIMEOUT` submission into `generated` (timeout issue removed, normal PDF handling). Any other event for a finished submission is acked and ignored.
6. **Worker health signal**: every tick main-service does a passive declare of the build queue. `/readyz` shows `build_queue.consumers` and `.messages`; Prometheus gauges `abet_build_queue_consumers` and `abet_build_queue_messages`. If there are 0 consumers for more than 5 minutes it logs one `ERROR` ("no portfolio worker has been consuming the build queue ...") per incident. Uploads are still accepted (jobs wait in the queue; the timeout is the backstop).
7. **Janitor**: hourly cleanup of leftovers, see [Storage layout and cleanup](#storage-layout-and-cleanup).

Operating it:

| Task | How |
|---|---|
| Change the timeout | set `BUILD_TIMEOUT_MINUTES` in `.env` (default 60; a 300 MB package normally builds in well under 10 minutes), then `make dev` / `make prod` to recreate main-service. The UI reads the value from `GET /api/v1/config` |
| Change the check interval | `RECONCILE_INTERVAL_SECONDS` (the timeout fires at most one interval late) |
| Restart hung workers | `make worker-restart` (dev) / `make worker-restart ENV=prod`; unfinished jobs are redelivered |
| Check the workers | `curl`-free: `$DC exec main-service wget -qO- http://localhost:8080/readyz` -> `build_queue`; or `rabbitmqctl list_queues name messages consumers` (`consumers` should equal the replica count) |
| See what timed out | `$DC logs main-service | grep "timed out"`; metric `abet_submissions_timed_out_total`; SQL: `SELECT id, created_at FROM submissions s JOIN submission_issues i ON i.submission_id=s.id WHERE i.code='BUILD_TIMEOUT'` |
| A user reports a stuck upload | tell them to wait for the timeout (it then shows Failed) and upload again; if it keeps happening check the worker logs and the queue first |

## portfolio-service cover patch (D1)

portfolio-service is "final" third-party-style code. The only functional change is in [`portfolio-service/worker.py`](../portfolio-service/worker.py) (+11 lines, marked `1b.`) inside `process_job`, right after the temporary program workspace is scaffolded:

```python
# 1b. Optional per-curriculum cover: copy into the program workspace as config/assets/cover.pdf
cover_pdf_path = payload.get("cover_pdf_path")
if cover_pdf_path:
    cover_src = Path(cover_pdf_path)
    if cover_src.is_file():  # copy2 to <program_dir>/config/assets/cover.pdf, log "Using curriculum cover page"
    else:                    # log warning, default cover is used
```

Why: the SRS requires one cover page template per curriculum that becomes page 1 of every portfolio. The build pipeline already picks up `config/assets/cover.pdf` of the program workspace; the job payload had no way to supply it. main-service stores the PDF at `covers/<curriculum_id>.pdf` and passes the path in `cover_pdf_path` (`PUT /curricula/{id}/cover`). Without the key (or if the file is missing) the worker behaves exactly as before. `portfolio-service/README.md` got one matching row in its payload table.
Verified end-to-end: page 1 of a generated PDF was the uploaded cover. When updating portfolio-service from upstream, re-apply this block.

A second, non-code adaptation is the **deploy shim** [`deploy/portfolio-service/entrypoint.sh`](../deploy/portfolio-service/entrypoint.sh): `config.py` lets `settings.json` override environment variables, so at container start the shim rewrites `RABBITMQ_*`, `DEFAULT_OUTPUT_DIR`, `DELETE_ZIP_DEFAULT` and `LOG_LEVEL` inside the container's `/app/settings.json` from the environment (other keys such as `LAYOUT_PRESETS` are preserved; the repo file is untouched), then runs `python3 -u worker.py`.

## Manual Google-login checklist (acceptance test after a deploy)

Prereq: in `.env` set `GOOGLE_CLIENT_ID/SECRET`, `SUPER_ADMIN_EMAILS=<your chula address>`, `PUBLIC_BASE_URL=http://localhost:8000`, `GOOGLE_REDIRECT_URL=http://localhost:8000/public/api/v1/auth/google/callback`, `ALLOWED_EMAIL_DOMAINS` containing your domain; in Google Cloud the OAuth client has exactly that redirect URI (and the consent screen is Internal or your account is a test user). Then `make restart s=auth-service` (and main-service so `SUPER_ADMIN_EMAILS` is upserted) — or just `make dev`.
(For a production deploy use your real https `PUBLIC_BASE_URL`/`GOOGLE_REDIRECT_URL`, `COOKIE_SECURE=true`, and `make prod` instead of the dev commands.)

1. Open `http://localhost:8000/` in a private window → redirected to `/login` (not a blank page).
2. Click **Sign in with Google**, pick the allowed account → lands on `/portfolios`; the sidebar shows your real name, picture and role chip **SUPER ADMIN** with all six menu items.
3. DevTools → Application → Cookies: `abet_session` is HttpOnly, SameSite=Lax, Path=/; there is no cookie readable from JS.
4. Open `/users`: your row has the lock icon (protected) and your name/picture are now filled (they were empty before the first login).
5. Negative: sign out, sign in with an account on an allowed domain that is NOT in Users → back on `/login` with the **Access Denied** modal. With a non-allowed domain (gmail) → Google offers only `hd` accounts, or you land on Access Denied.
6. Provision a second real account: Users → Add (role Instructor, dept/curriculum) → sign in with that account in another browser profile → only "Course Portfolios", only offerings you assigned.
7. Deep link: while signed out open `/courses` → after login you land on `/courses` (redirect parameter), not `/portfolios`.
8. Sign Out → `/login`; pressing Back and reloading does not show data; `curl -H "Cookie: abet_session=<old>" …/auth/api/v1/me` → 401 (session revoked).
9. Wait/inspect: `SESSION_TTL_HOURS` (12) expiry → next API call returns 401 and the UI goes to `/login`.

Note: the authenticated smoke of the **production** stack was not automated during development (the dev stack passed 43 e2e tests three times, and prod passed the unauthenticated checks); this checklist is that acceptance test.
