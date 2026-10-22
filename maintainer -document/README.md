# ABET Course Portfolio Management System — Maintainer Documentation

This folder is the handover documentation for whoever deploys, operates or extends the system.
Everything here was written from the code and configuration in this repository (not from the plan), and the
non-destructive commands were checked (`make help`, `make config`).

## Documentation index

| Document | Read it when you want to… |
|---|---|
| [structure/service diagram/service-diagram.md](structure/service%20diagram/service-diagram.md) | see how the containers talk to each other; login, upload and CSV-import sequences |
| [structure/folder structure/folder-structure.md](structure/folder%20structure/folder-structure.md) | find where code lives; learn the Go layering; add a module or a frontend feature |
| [deployment.md](deployment.md) | install on a server: prerequisites, `.env` reference (incl. MinIO), Google OAuth, first run, HTTPS, backups (`pg_dump` + `mc mirror`), upgrades, Makefile |
| [development.md](development.md) | run the dev stack, run tests, regenerate sqlc/mocks, add migrations, add an endpoint end-to-end |
| [operations.md](operations.md) | check health, read logs, fix common failures, understand queues, MinIO object layout and console, post-deploy login checklist |
| [security-and-permissions.md](security-and-permissions.md) | authorization matrix, sessions, gateway header stripping, limits, what is exposed |
| [monitoring.md](monitoring.md) | turn on Grafana (metrics) and Kibana (logs + end-to-end traces), read dashboards, find the trace of one submission, retention and `make mon-prune` |
| [business-rules.md](business-rules.md) | mapping-CSV import rules, upload validation rules and issue codes, storage limit, conventions |
| [api-reference.md](api-reference.md) | list every HTTP endpoint with role requirements |

## What the system does

The Faculty of Science needs one place where course instructors submit a *course portfolio* (a structured folder of
syllabus, lectures, assignments, quizzes, exams, grades) and get back a single ABET-style PDF. The system (SRS, October 2026):

* **Login** with an official `@chula.ac.th` Google account. Only users that were **pre-provisioned** in the system can enter;
  everybody else sees "Access Denied".
* **Upload** a portfolio (a folder zipped in the browser, or a `.zip`) for a course offering (course + academic year + semester).
  The server validates it **immediately** against the SRS §7 rules and shows errors/warnings. A valid package is queued and a
  separate worker builds the PDF (LaTeX). The user then downloads the PDF (single, or many as one zip).
* **Manage** (admins): users, course offerings, who may upload/download each offering (one by one or by a *mapping CSV*),
  and one cover page PDF per curriculum that becomes page 1 of every portfolio of that curriculum.
* **Organisation** (super admin): departments and curricula.

### Roles

| Role | Can do |
|---|---|
| `staff` | Download portfolios of their own department (default) and of explicitly assigned offerings; upload only if granted. |
| `instructor` | Upload and/or download only for offerings they were explicitly assigned (flags per assignment). |
| `admin` | Everything an instructor/staff can do inside their **scope** (departments and/or curricula) plus manage instructor/staff users, offerings, access, mapping CSV and covers inside that scope. |
| `super_admin` | Everything, including departments, curricula, admins and admin scopes. |

A user has exactly one global role (the highest applies). Details: [security-and-permissions.md](security-and-permissions.md).

### Components

| Component | Tech | Responsibility |
|---|---|---|
| `api-gateway` | Apache APISIX 3.13 (standalone, YAML routes) | single public entry point; forward-auth; strips client identity headers |
| `frontend` | React 19, Vite, TypeScript, Tailwind v4; nginx in prod | the web UI |
| `auth-service` | Go (Gin, hexagonal + CQRS) | Google OIDC login, session cookie/JWT, `/verify` for the gateway; DB `auth_db` |
| `main-service` | Go (Gin, hexagonal + CQRS) | all domain logic and authorization, upload validation, queue producer/consumer, file storage; DB `main_db` |
| `portfolio-service` | Python worker (LaTeX) | builds the PDF from a zip; consumes `course_build_queue`, publishes `course_status_queue`. Not exposed. Only change vs. the original: the cover patch ([operations.md](operations.md#portfolio-service-cover-patch-d1)) |
| `postgres` | PostgreSQL 17 | two databases: `auth_db`, `main_db` |
| `rabbitmq` | RabbitMQ 4 | the two queues between main-service and the worker |
| `minio` (+ one-shot `minio-init`) | MinIO (S3 API) | object storage for **every** file: uploads, covers, worker outputs, final PDFs (one bucket, `MINIO_BUCKET`). The S3 port is internal; the web console is published on `MINIO_CONSOLE_HOST_PORT` ([operations.md](operations.md#minio-console)) |
| `grafana`, `prometheus`, `elasticsearch`, `kibana`, `apm-server`, `filebeat`, exporters (profile `monitoring`, optional) | Grafana, Prometheus, Elastic 8.19 | metrics, logs and OpenTelemetry traces of all containers, reachable only through the gateway under `/grafana` and `/kibana`. Off by default: [monitoring.md](monitoring.md) |

Source of design decisions (D1–D11, A1–A8) lives in the CTO plan (`plan/decisions.md`, outside this repo); the relevant
ones are restated where they matter, e.g. in [business-rules.md](business-rules.md).

## 60-second start

```bash
make env            # creates .env from .env.template
# edit .env: replace every "change-me", set GOOGLE_CLIENT_ID/SECRET, SUPER_ADMIN_EMAILS, PUBLIC_BASE_URL, GOOGLE_REDIRECT_URL
make prod           # production build + start; open PUBLIC_BASE_URL
```
Full instructions and pitfalls: [deployment.md](deployment.md). For development use `make dev` ([development.md](development.md)).

## Repository map (short)

```
api-gateway/  auth-service/  main-service/  portfolio-service/  frontend/   the five code components
deploy/       postgres init script, portfolio-service entrypoint shim, dev seed data
screen-design/  UI prototype (reference only, not built)
compose.yaml + compose.dev.yaml + compose.prod.yaml + Makefile + .env.template
maintainer -document/   this documentation
```
