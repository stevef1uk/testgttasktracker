# Implementation plan

## Phase scope

Active phase: `phase-1-app` (PostgreSQL store and runnable project API).

Required implement beads (9 paths, all from `required_files`):

- `ta-71y` → `taskboard/internal/store/postgres.go`
- `ta-5rq` → `taskboard/internal/store/store_test.go`
- `ta-9ik` → `taskboard/internal/store/postgres_integration_test.go`
- `ta-5kn` → `taskboard/cmd/taskboard/main.go`
- `ta-12r` → `taskboard/internal/web/app.go`
- `ta-83w` → `taskboard/internal/web/api.go`
- `ta-62r` → `taskboard/internal/web/api_test.go`
- `ta-hzj` → `taskboard/web/assets.go`
- `ta-0ia` → `taskboard/web/templates/index.html`

Phase-1a files (`store.go`, `sqlite.go`, `model.go`, `config.go`) are completed dependencies. They have no beads in this phase and must not be reimplemented here.

`taskboard/internal/web/app_test.go` is not in this phase's `required_files`. Its finding is deferred to the phase that owns it. Routing and static-404 coverage for this phase lives in `api_test.go`.

## Integration contract

Entrypoint: `taskboard/cmd/taskboard/main.go`.

Startup order:

1. Load configuration.
2. Construct `PostgresStore` via `NewPostgresStore` in `taskboard/internal/store/postgres.go`. Schema setup runs under a transactional advisory lock.
3. Pass the store and `embedweb.FS` to `NewApp(s store.Store) *App` in `taskboard/internal/web/app.go`.
4. `NewApp` registers `GET /{$}`, `GET /static/`, and calls `registerAPI` from `taskboard/internal/web/api.go`.
5. Serve `(*App).Handler()` on the configured address. Shut down gracefully on SIGTERM.

HTTP routes (SPEC, verbatim):

| Method | Path |
|--------|------|
| GET | `/api/health` |
| GET | `/api/ready` |
| GET | `/api/projects` |
| POST | `/api/projects` |
| DELETE | `/api/projects/{project_id}` |
| GET | `/api/projects/{project_id}/tasks` |
| POST | `/api/projects/{project_id}/tasks` |
| PATCH | `/api/tasks/{task_id}` |
| DELETE | `/api/tasks/{task_id}` |

Exported symbols per phase file:

- `taskboard/internal/store/postgres.go`: `PostgresStore`, `NewPostgresStore`.
- `taskboard/internal/web/app.go`: `App`, `NewApp(s store.Store) *App`, `(*App).Handler() http.Handler`, `writeJSON`.
- `taskboard/internal/web/api.go`: `(*App).registerAPI` and the REST handlers for projects, tasks, health, and ready.
- `taskboard/web/assets.go`: `FS embed.FS`.
- `taskboard/cmd/taskboard/main.go`: `main`.

## Bead map

### ta-71y: taskboard/internal/store/postgres.go
- Scope: PostgreSQL implementation of `store.Store`.
- Architecture: advisory-lock schema setup; `NewPostgresStore` is the constructor.
- Interfaces: `PostgresStore`, `NewPostgresStore`.
- Depends on: `taskboard/internal/store/store.go` (completed, phase 1a).
- Consumed by: `taskboard/internal/store/store_test.go`, `taskboard/internal/store/postgres_integration_test.go`, `taskboard/cmd/taskboard/main.go`.
- Acceptance: schema init is idempotent; project and task CRUD return `store.ErrNotFound` for missing rows; no stub bodies.
- Verify: `cd taskboard && go mod tidy && go test -timeout 30s -count=1 ./internal/store/...`

### ta-5rq: taskboard/internal/store/store_test.go
- Scope: SQLite in-memory unit tests for the Store contract (phase 1a dependency).
- Depends on: `taskboard/internal/store/postgres.go`, `taskboard/internal/store/sqlite.go`.
- Consumed by: none.
- Acceptance: tests cover project and task CRUD, foreign-key cascade, and filter combinations. No always-pass tests.
- Verify: `cd taskboard && go mod tidy && go test -timeout 30s -count=1 ./internal/store/...`

### ta-9ik: taskboard/internal/store/postgres_integration_test.go
- Scope: Postgres integration tests, gated on `TASKBOARD_TEST_POSTGRES_URL`.
- Depends on: `taskboard/internal/store/postgres.go`.
- Consumed by: none.
- Acceptance: when the env var is unset the tests skip. When set, they exercise schema init, CRUD, and not-found errors.
- Verify: `cd taskboard && go mod tidy && go test -timeout 30s -count=1 ./internal/store/...`

### ta-12r: taskboard/internal/web/app.go
- Scope: router and App wiring.
- Interfaces: `App`, `NewApp(s store.Store) *App`, `(*App).Handler() http.Handler`, `writeJSON`.
- Depends on: `taskboard/internal/store/postgres.go` (store interface), `taskboard/web/assets.go` (`embedweb.FS`).
- Consumed by: `taskboard/internal/web/api.go`, `taskboard/cmd/taskboard/main.go`.
- Acceptance: `GET /` serves `templates/index.html`; `GET /static/` serves embedded assets; a missing static file returns 404; `registerAPI` is invoked from `NewApp`.
- Verify: `cd taskboard && go mod tidy && go build ./internal/web/...`

### ta-83w: taskboard/internal/web/api.go
- Scope: REST handlers for the nine SPEC routes.
- Interfaces: `(*App).registerAPI`, plus unexported handlers for projects, tasks, health, and ready.
- Depends on: `taskboard/internal/web/app.go` (`App`, `writeJSON`), `taskboard/internal/store/postgres.go`.
- Consumed by: `taskboard/internal/web/api_test.go`.
- Acceptance: `GET /api/health` returns exact JSON; `GET /api/ready` returns 503 when the ping fails; invalid IDs return 400; missing project or task returns 404; PATCH rejects `null` and empty bodies; DELETE project cascades and returns 204.
- Verify: `cd taskboard && go mod tidy && go test -timeout 30s -count=1 ./internal/web/... -run 'TestAPI'`

### ta-62r: taskboard/internal/web/api_test.go
- Scope: HTTP tests for the REST API and static routing.
- Depends on: `taskboard/internal/web/api.go`, `taskboard/internal/web/app.go`.
- Consumed by: none.
- Acceptance: covers task create defaults; PATCH rejects empty and null fields; filters combine with AND; 404 for a missing project or task; length limits enforced; a missing static file returns 404.
- Verify: `cd taskboard && go mod tidy && go test -timeout 30s -count=1 ./internal/web/...`

### ta-hzj: taskboard/web/assets.go
- Scope: `go:embed` of `templates/*` and `static/*`.
- Interfaces: `FS embed.FS`.
- Depends on: `taskboard/web/templates/index.html`, `taskboard/web/static/app.js`, `taskboard/web/static/style.css`.
- Consumed by: `taskboard/internal/web/app.go`.
- Acceptance: the embed directives match files present on disk. No build error from a missing pattern.
- Verify: `cd taskboard && go mod tidy && go build ./web/...`

### ta-0ia: taskboard/web/templates/index.html
- Scope: Kanban page template.
- Depends on: `taskboard/web/static/app.js`, `taskboard/web/static/style.css`.
- Consumed by: `taskboard/web/assets.go`, `taskboard/internal/web/app.go`.
- Acceptance: the page has real controls (project select, add-task form, status and priority filters) wired to the REST API through `app.js`. It is not static content only.
- Verify: `cd taskboard && go test ./...`

### ta-5kn: taskboard/cmd/taskboard/main.go
- Scope: process entrypoint.
- Interfaces: `main`.
- Depends on: `taskboard/internal/store/postgres.go` (`NewPostgresStore`), `taskboard/internal/web/app.go` (`NewApp`, `Handler`), `taskboard/web/assets.go`.
- Consumed by: none.
- Acceptance: loads configuration, builds the store and app, serves the handler, and shuts down gracefully on SIGTERM.
- Verify: `cd taskboard && go mod tidy && go test -timeout 30s -count=1 ./cmd/taskboard/...`

## Phase context

Active delivery phase `phase-1-app` only. Phase verification: `cd taskboard && go test ./...`.
