# ABET Course Portfolio Management System

A web system for the Faculty of Science, Chulalongkorn University. Course instructors upload a *course portfolio*
(syllabus, lectures, assignments, quizzes, exams, grades) for a course offering. The server checks it right away,
and a background worker builds one ABET-style PDF from it.

- **Login** with an `@chula.ac.th` Google account. Only users added to the system beforehand can sign in.
- **Upload** a folder or a `.zip`. Errors and warnings show at once; a valid package is queued and built into a PDF (LaTeX).
- **Download** single PDFs, or many at once as one zip.
- **Manage** (admins): users, course offerings, upload/download access (one by one or with a mapping CSV), and a cover page per curriculum.

Roles: `staff`, `instructor`, `admin` (limited to departments and/or curricula), `super_admin`.

## Architecture

```
browser ──► api-gateway (APISIX) ──► frontend        React 19 + Vite + Tailwind (nginx in prod)
                 │  forward-auth ──► auth-service    Go: Google OIDC, sessions
                 └─────────────────► main-service    Go: domain logic, authorization, upload validation
                                          │
                     postgres ◄───────────┤          auth_db, main_db
                     minio    ◄───────────┤          every file (uploads, covers, PDFs)
                     rabbitmq ◄───────────┘──► portfolio-service   Python + LaTeX PDF worker
```

Optional monitoring (Grafana, Prometheus, Elasticsearch, Kibana, APM) runs behind the gateway at `/grafana` and `/kibana`.

## Quick start

Requirements: Docker with Compose v2, and `make`.

```bash
make env     # creates .env from .env.template
# edit .env: replace every "change-me", set GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET,
# SUPER_ADMIN_EMAILS, PUBLIC_BASE_URL and GOOGLE_REDIRECT_URL
make prod    # build and start the production stack, then open PUBLIC_BASE_URL
```

For development (hot reload, seed data, no Google login needed):

```bash
make dev                                   # http://localhost:8000
make seed                                  # load test users and offerings
make dev-session email=admin1@chula.ac.th  # print a session cookie for a seeded user
```

Add `MON=1` to `make dev` / `make prod` to start monitoring too. Run `make help` to see all targets
(`logs`, `ps`, `restart`, `test`, `lint`, `db-shell`, `down`, `clean`, ...).

## Repository layout

| Path | Contents |
|---|---|
| `api-gateway/` | APISIX routes and config (dev and prod) |
| `auth-service/` | Go login and session service |
| `main-service/` | Go core API |
| `portfolio-service/` | Python PDF build worker |
| `frontend/` | React web UI and Playwright end-to-end tests |
| `deploy/` | Postgres init script, entrypoint shims, monitoring config, dev seed data |
| `screen-design/` | UI prototype (reference only, not built) |
| `compose*.yaml`, `Makefile`, `.env.template` | stack definition and commands |
| `maintainer -document/` | full maintainer documentation |

## Documentation

Detailed guides are in [`maintainer -document/`](maintainer%20-document/README.md):

- [Deployment](maintainer%20-document/deployment.md): server setup, `.env` reference, Google OAuth, HTTPS, backups, upgrades
- [Development](maintainer%20-document/development.md): dev stack, tests, migrations, adding an endpoint
- [Operations](maintainer%20-document/operations.md): health checks, logs, common failures
- [Security and permissions](maintainer%20-document/security-and-permissions.md)
- [Monitoring](maintainer%20-document/monitoring.md)
- [Business rules](maintainer%20-document/business-rules.md): CSV import and upload validation rules
- [API reference](maintainer%20-document/api-reference.md)

Each service also has its own `README.md`.

## Security note

Never commit `.env`; it is git-ignored. Only `.env.template` (placeholders) belongs in the repository.
