# Implementation plan — phase-1-app (PostgreSQL store and runnable project API)

Active phase: `phase-1-app`. Depends on `phase-1-core` (go.mod, config, model, store interface, SQLite store already delivered and closed).
Scope: exactly the 9 `required_files` below. Later phases get their own beads.

## Bead map

### ta-5el: taskboard/internal/store/postgres.go
- Scope: PostgreSQL implementation of the shared store.Store interface using pgx/v5 stdlib and database/sql.
- Architecture: OpenPostgres(ctx, url) (Store, error); no SQLite DDL, no modernc imports.
- Interfaces: OpenPostgres(ctx context.Context, url string) (Store, error).
- Depends on: taskboard/internal/store/store.go.
- Consumed by: taskboard/cmd/taskboard/main.go, taskboard/internal/store/store_test.go, taskboard/internal/store/postgres_integration_test.go.
- Acceptance:
  - Schema creation is idempotent and runs under a transaction-scoped advisory lock.
  - Identity primary keys; `$1` placeholders only; no interpolation of user input.
  - Projects and tasks list in ascending id; timestamps are UTC RFC3339.
  - Deleting a project cascades to its tasks; tasks.project_id is indexed.
- Verify: `cd taskboard && go test ./internal/store`.

### ta-en7: taskboard/internal/store/store_test.go
- Scope: shared store contract tests against SQLite (t.TempDir()) and PostgreSQL when configured.
- Architecture: SPEC section 8 store tests 1–8.
- Depends on: taskboard/internal/store/store.go, taskboard/internal/store/sqlite.go, taskboard/internal/store/postgres.go.
- Acceptance:
  - Fresh schema is empty; repeated initialization is idempotent.
  - Projects list ascending; tasks default to todo and medium with UTC timestamps.
  - Filters by status, priority, and both (AND).
  - Patch preserves omitted fields; deleting a project removes its tasks.
- Verify: `cd taskboard && go test ./internal/store`.

### ta-8pf: taskboard/internal/store/postgres_integration_test.go
- Scope: PostgreSQL-only tests, skipped unless TASKBOARD_TEST_POSTGRES_URL is nonempty.
- Architecture: dedicated disposable database; uniquely named objects; clean up only own data.
- Depends on: taskboard/internal/store/postgres.go.
- Acceptance:
  - Runs the same contract as SQLite: cascade, ordering, filters, timestamps, schema idempotency.
  - Never touches a shared or production schema.
- Verify: `cd taskboard && go test ./internal/store`.

### ta-xlo: taskboard/cmd/taskboard/main.go
- Scope: process entrypoint: parse config, open store, build app, serve, shut down gracefully.
- Architecture: wiring order in the Integration contract below.
- Depends on: taskboard/internal/config/config.go, taskboard/internal/store/sqlite.go, taskboard/internal/store/postgres.go, taskboard/internal/web/app.go, taskboard/web/assets.go.
- Acceptance:
  - Invalid configuration exits nonzero without printing the PostgreSQL URL or password.
  - Selects SQLite or PostgreSQL from config; never falls back silently.
  - On SIGTERM calls Shutdown with a bounded timeout, then closes the store.
- Verify: `cd taskboard && go build -o /tmp/taskboard-gate ./cmd/taskboard`.

### ta-uu7: taskboard/internal/web/app.go
- Scope: App type, NewApp, route registration, root page rendering, static serving.
- Architecture: App built only via NewApp(store.Store, fs.FS); no global database handle.
- Depends on: taskboard/internal/store/store.go, taskboard/web/assets.go, taskboard/web/templates/index.html.
- Acceptance:
  - Uses net/http ServeMux method-aware patterns.
  - GET / renders index.html through html/template.
  - Contains no SQL text and no direct driver use.
- Verify: `cd taskboard && go test ./internal/web`.

### ta-rod: taskboard/internal/web/api.go
- Scope: JSON handlers for health, ready, and project routes; error encoder.
- Architecture: SPEC section 6 health, ready, and project routes.
- Depends on: taskboard/internal/web/app.go, taskboard/internal/store/store.go.
- Acceptance:
  - GET /api/health returns 200 {"status":"ok"}, independent of the database.
  - GET /api/ready returns 200 {"status":"ready"} or 503 {"error":"database unavailable"}.
  - GET /api/projects returns ascending ids, `[]` when empty; 500 generic on storage failure.
  - POST /api/projects returns 201; blank or overlong name returns 400.
  - DELETE /api/projects/{project_id} returns 204 with empty body; missing returns 404.
- Verify: `cd taskboard && go test ./internal/web`.

### ta-is1: taskboard/internal/web/api_test.go
- Scope: httptest contract tests for health, ready, and project round trips.
- Architecture: SPEC section 8 API tests 1, 2, 4 (project subset), 5, 6, 9.
- Depends on: taskboard/internal/web/app.go, taskboard/internal/web/api.go, taskboard/internal/store/sqlite.go.
- Acceptance:
  - Exact liveness and ready JSON against a healthy SQLite store.
  - Project create, list, delete asserts stored state, not status codes alone.
  - Blank name rejected with 400; empty list serializes as `[]`; 204 bodies are empty.
  - Database failure yields generic 500 and readiness 503 without internal error text.
- Verify: `cd taskboard && go test ./internal/web`.

### ta-1gk: taskboard/web/assets.go
- Scope: the only go:embed location for templates and static assets; exports the filesystem.
- Architecture: go:embed cannot traverse parent directories.
- Interfaces: `var FS fs.FS`; phase-1-app embeds `templates/*` only. Static `static/*` is added in phase-2-kanban.
- Depends on: taskboard/web/templates/index.html.
- Acceptance: FS opens templates/index.html with nonempty content.
- Verify: `cd taskboard && go build ./web/...`.

### ta-1vd: taskboard/web/templates/index.html
- Scope: real server-rendered HTML shell parsed by html/template.
- Architecture: heading TaskBoard, project input and Add project button, project selector fed from /api/projects.
- Depends on: none.
- Acceptance:
  - Contains the text TaskBoard and the project input with Add project button.
  - No innerHTML interpolation of user data.
- Verify: `cd taskboard && go test ./internal/web`.

## Integration contract

Required because the active phase includes taskboard/cmd/taskboard/main.go.

HTTP routes per SPEC section 6 (exact paths):
- GET /api/health — phase-1-app
- GET /api/ready — phase-1-app
- GET /api/projects — phase-1-app
- POST /api/projects — phase-1-app
- DELETE /api/projects/{project_id} — phase-1-app
- GET /api/projects/{project_id}/tasks — phase-2-kanban (not in this phase)
- POST /api/projects/{project_id}/tasks — phase-2-kanban (not in this phase)
- PATCH /api/tasks/{task_id} — phase-2-kanban (not in this phase)
- DELETE /api/tasks/{task_id} — phase-2-kanban (not in this phase)
- GET / — phase-1-app (renders index.html)
- GET /static/app.js — phase-2-kanban (static assets not yet embedded)

Wire order: config.Load → OpenSQLite or OpenPostgres → web.NewApp(store, web.FS) → http.Server on TASKBOARD_HOST:TASKBOARD_PORT → on SIGTERM Shutdown then store.Close().

Exported symbols per file:
- taskboard/internal/store/postgres.go exports OpenPostgres.
- taskboard/internal/store/sqlite.go (phase-1-core) exports OpenSQLite.
- taskboard/internal/store/store.go (phase-1-core) exports Store, NewTask, ErrNotFound, ErrForeignKey.
- taskboard/internal/web/app.go exports App and NewApp.
- taskboard/web/assets.go exports FS.
- taskboard/internal/web/api.go exports no symbols; handlers are App methods.

Dependency order: store.go, sqlite.go → postgres.go → assets.go and index.html → app.go → api.go → main.go. Tests follow their subject.

## Future considerations

- Task CRUD routes, filters, and the Kanban UI belong to phase-2-kanban.
- Static asset embedding (static/*) is introduced in phase-2-kanban.
