# Implementation plan — phase-1-app (PostgreSQL store and runnable project API)

Scope: only the nine phase-1-app required_files. Dependencies come from phase-1-core (already delivered). Later phases (tasks, Kanban UI, Playwright, Docker, k3s) are out of scope here.

## Integration contract

The entrypoint `taskboard/cmd/taskboard/main.go` loads config via `config.LoadConfig()`, opens the store with `store.NewSQLiteStore` or `store.NewPostgresStore` based on `TASKBOARD_DB_DRIVER`, builds the app with `web.NewApp(dbStore, assets.FS)`, and serves on `TASKBOARD_HOST:TASKBOARD_PORT` with SIGINT/SIGTERM graceful shutdown.

Routes registered in phase-1-app (exact SPEC §6 table, projects and health only):
- GET `/api/health` returns 200 `{"status":"ok"}`
- GET `/api/ready` returns 200 `{"status":"ready"}` or 503 `{"error":"database unavailable"}`
- GET `/api/projects` returns 200 JSON array ordered by ID, 500 on storage failure
- POST `/api/projects` returns 201 project JSON, 400 on invalid name
- DELETE `/api/projects/{project_id}` returns 204 with empty body, 404 if missing
- GET `/` serves the embedded `templates/index.html`

Task routes (`/api/projects/{project_id}/tasks`, `/api/tasks/{task_id}`) and `/static/*` are NOT part of phase-1-app.

Exported symbols per file (from architecture ownership):
- `taskboard/internal/store/postgres.go`: `NewPostgresStore`
- `taskboard/internal/store/store.go`: `Store` interface, `ErrNotFound`, DDL constants
- `taskboard/web/assets.go`: `FS` (embeds `templates/*` only in this phase)
- `taskboard/internal/web/app.go`: `App`, `NewApp`
- `taskboard/internal/web/api.go`: API route methods on `*App`

## Bead map

### ta-11e: taskboard/internal/store/postgres.go
- Scope: PostgreSQL implementation of the Store contract, schema setup under a transactional advisory lock.
- Architecture: `architecture.md` Components and Module ownership table.
- Interfaces: `NewPostgresStore(ctx, url) (Store, error)`.
- Depends on: `taskboard/internal/store/store.go`.
- Consumed by: `taskboard/cmd/taskboard/main.go`, `taskboard/internal/store/postgres_integration_test.go`.
- Acceptance: identity primary keys, `$1` placeholders, UTC RFC3339 timestamps, `ON DELETE CASCADE` on tasks, idempotent schema on repeated and concurrent start.
- Verify: `cd taskboard && go test ./internal/store`

### ta-nw9: taskboard/internal/store/store_test.go
- Scope: shared contract tests against the SQLite store, isolated via `t.TempDir()`.
- Acceptance: missing rows return `ErrNotFound`; list functions return non-nil empty slices; project list ascending by ID; schema idempotent; persistence across close and reopen; foreign keys enforced.
- Depends on: `taskboard/internal/store/store.go`, `taskboard/internal/store/sqlite.go`.
- Verify: `cd taskboard && go test ./internal/store`

### ta-5qb: taskboard/internal/store/postgres_integration_test.go
- Scope: same contract as SQLite, run only when `TASKBOARD_TEST_POSTGRES_URL` is set, using a dedicated disposable database and uniquely named test objects.
- Acceptance: schema idempotency, project ordering, cascade delete, cleanup of only its own rows.
- Depends on: `taskboard/internal/store/postgres.go`.
- Verify: `cd taskboard && go test ./internal/store` (skips when env var is unset)

### ta-83w: taskboard/internal/web/api.go
- Scope: project REST handlers and JSON encoding for the routes listed in the Integration contract.
- Interfaces: method handlers on `*App`, registered from `NewApp`.
- Depends on: `taskboard/internal/web/app.go`, `taskboard/internal/store/store.go`.
- Consumed by: `taskboard/internal/web/api_test.go`.
- Acceptance: 400 for blank or overlong name (1–80 code points after trim); 404 for missing project on DELETE; list returns `[]` not `null`; DB errors return a generic 500 and never leak SQL or credentials.
- Verify: `cd taskboard && go test ./internal/web`

### ta-8l7: taskboard/internal/web/api_test.go
- Scope: httptest tests of the project API against a temporary SQLite store.
- Acceptance: liveness exact JSON; ready 200 on healthy DB and 503 `{"error":"database unavailable"}` on failure; project create, list, delete round trip with state assertions; 204 bodies empty; 400 on invalid name; 404 on missing project; list ascending by ID.
- Depends on: `taskboard/internal/web/api.go`.
- Verify: `cd taskboard && go test ./internal/web`

### ta-12r: taskboard/internal/web/app.go
- Scope: `App` struct, `NewApp(store, assets)`, `Handler()`, and the GET `/` handler serving the embedded template.
- Interfaces: `NewApp(s store.Store, fsys fs.FS) *App`.
- Depends on: `taskboard/internal/store/store.go`, `taskboard/web/assets.go`.
- Consumed by: `taskboard/internal/web/api.go`, `taskboard/cmd/taskboard/main.go`.
- Acceptance: no mutable global DB handle; route patterns use method-aware `net/http` ServeMux.
- Verify: `cd taskboard && go test ./internal/web`

### ta-qfs: taskboard/web/assets.go
- Scope: `embed` FS. Phase-1-app embeds `templates/*` only; `static/*` is added in phase-2-kanban.
- Interfaces: `var FS embed.FS`.
- Consumed by: `taskboard/internal/web/app.go`.
- Acceptance: `templates/index.html` is readable through `FS`.
- Verify: `cd taskboard && go test ./internal/web`

### ta-dw6: taskboard/web/templates/index.html
- Scope: real server-rendered HTML shell with `<h1>TaskBoard</h1>`, the project form, and the project list region. Task controls and Kanban columns are phase-2 and are not built here.
- Consumed by: `taskboard/web/assets.go`, served by `taskboard/internal/web/app.go`.
- Acceptance: page title and heading present; contains the project name input and add button.
- Verify: `cd taskboard && go test ./internal/web`

### ta-5kn: taskboard/cmd/taskboard/main.go
- Scope: entrypoint. Loads config, opens the selected store, runs schema setup, builds the app, serves, and shuts down on SIGINT/SIGTERM with an HTTP shutdown timeout and DB close.
- Depends on: `taskboard/internal/config/config.go`, `taskboard/internal/store/postgres.go`, `taskboard/internal/store/sqlite.go`, `taskboard/internal/web/app.go`.
- Acceptance: nonzero exit with a clear message on invalid config; no full DSN or password printed; starts on 127.0.0.1:8080 with no required environment variables.
- Verify: `cd taskboard && go test ./... && go run ./cmd/taskboard` (smoke: `curl -s http://127.0.0.1:8080/api/health`)

## Verification
Phase gate: `cd taskboard && go test ./...`. Manual smoke: GET `/api/health`, GET `/`, create/list/delete a project.
