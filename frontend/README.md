# ABET frontend

Vite 8 + React 19 + TypeScript 6 (strict) + Tailwind v4 + React Router 7 + TanStack Query 5 + lucide-react.
Visual reference: `../screen-design` (read-only). API contract: `plan/contracts/api.md`.

## Run

```bash
npm ci
npm run dev        # http://localhost:5173
npm run build      # tsc -b && vite build  → dist/
npm run lint       # oxlint
npm test           # vitest run (jsdom)
npm run preview
```

- **Through the stack (docker compose):** the gateway serves the UI on `GATEWAY_HTTP_PORT` and routes `/public/*` (auth-service) and `/auth/*` (main-service). The dev container runs `Dockerfile.dev` (vite on :5173); HMR connects through the gateway using `VITE_HMR_CLIENT_PORT` (default `8000`).
- **Vite alone (outside docker):** `/public` and `/auth` are proxied to the gateway at `VITE_DEV_GATEWAY` (default `http://localhost:8000`), so cookies/login keep working against a running stack.
- **Prod image:** `Dockerfile` builds with node:24 and serves `dist/` from nginx:1.27 (`nginx.conf`: SPA fallback, gzip, long cache for `/assets`).

| env var | default | purpose |
|---|---|---|
| `VITE_HMR_CLIENT_PORT` | `8000` | websocket port the browser uses for HMR (gateway port) |
| `VITE_DEV_GATEWAY` | `http://localhost:8000` | proxy target when vite runs outside docker |

## Structure

```
src/
  main.tsx
  app/          App, router (route table + guards), providers (Toast + QueryClient + 401 handler), queryClient
  config/       env.ts — API bases (/public/api/v1, /auth/api/v1) and upload limits
  lib/          http (fetch wrapper, ApiError, upload w/ progress, blob download), download, zipFolder (folder → zip),
                format (dates, bytes, semester labels), cn, useSort
  components/
    ui/         Button, IconButton, Modal (focus trap, Esc), ConfirmDialog, Badge/RoleBadge/StatusBadge, Table (sortable th),
                ColumnFilterPopover, SearchInput, SearchableSelect, FilterToolbar/FilterPill, FileDropzone, Field/NativeSelect,
                Tabs, Tooltip, Toast (+useToast), Alert, ProgressBar, EmptyState, Spinner, Checkbox
    layout/     AppShell, Sidebar (collapsible, role-based nav), PageHeader
  features/
    auth/ portfolios/ courses/ covers/ users/ departments/ curricula/   api.ts (query hooks) · types.ts · pages · components/
    shared/     useOrgFilters (dept/curriculum/year/semester/search in URL params), useOrgOptions (scope-limited options),
                DeptCurriculumFilters, YearSelect, scope, orgPills
  styles/index.css
  test/         vitest setup + mockApi/renderApp helpers
```

## Conventions

- Types mirror `api.md` exactly (snake_case kept). Server enforces authorization; the UI only hides/disables controls.
- Query keys: `['me']`, `['departments']`, `['curricula', deptId|null]`, `['users', params]`, `['offerings', params]`,
  `['offering-access', id]`, `['years']`, `['portfolios', params]`, `['submission', id]`. Mutations invalidate the affected keys.
- Every mutation error shows the API `error` text as a toast, except modals that render it inline (`meta: { inlineError: true }`).
- `401` from any API call → `me` is invalidated and the app navigates to `/login`.
- Semesters are stored as 1/2/3; label 3 is "Summer". Dates are shown as `DD/MM/YYYY HH:mm` local time.
- Portfolio upload: a dropped folder or `webkitdirectory` pick is zipped in the browser (`lib/zipFolder.ts`, keeps the top-level folder
  name, skips `.DS_Store`/`__MACOSX`/`Thumbs.db`); a `.zip` can also be chosen directly. Client limit 300 MB.

## E2E smoke suite (Playwright, chromium)

`frontend/e2e/` drives the real stack through the gateway. Prerequisites: `make dev` is running (docker compose) and `npx playwright install chromium` was done once.

```bash
make e2e                       # from the repo root;  or  cd frontend && npm run e2e
BASE_URL=http://localhost:8000 npx playwright test portfolios   # one file
```
`e2e/global-setup.ts` loads `deploy/dev/seed.sql` (idempotent) and mints one session cookie per seeded role with `go run ./cmd/devsession` inside the
auth-service container (no Google involved). Specs: `auth` (redirect, Google 302 params, sign out), `sidebar` (items per role), `org` (departments/curriculums/users),
`courses` (CRUD, access, CSV import preview+apply), `covers`, `portfolios` (invalid zip → issues; folder → Processing → Generated → cover is page 1 → download),
`roles` (every page loads without console/API errors). Tests create their own throw-away departments/curricula (unique codes) and delete them afterwards.
The valid upload takes ~20 s (real TeX build); the test allows 5 minutes. Reports: `playwright-report/`, traces in `test-results/` (both git-ignored).

### Hardening specs (`e2e/hardening.spec.ts`)
- H4 gateway allowlist (`/public/*`, `/auth/internal|metrics|healthz` → 404 JSON), H1 instructor filter options, H3 limits shown from `GET /config` run in the normal suite.
- H6 build-timeout scenario is opt-in: set `BUILD_TIMEOUT_MINUTES=1` and `RECONCILE_INTERVAL_SECONDS=10` in `.env`, `make dev`, then `E2E_BUILD_TIMEOUT=1 npx playwright test e2e/hardening.spec.ts -g H6` (the test stops/starts `portfolio-service` itself). Restore 60/60 afterwards.
- The Compose project (`<COMPOSE_PROJECT_NAME>-dev`) is derived from `.env` by `e2e/env.ts`.
