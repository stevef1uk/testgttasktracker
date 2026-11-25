# Implementation plan (phase-1-app, rework amendment)

Active phase: `phase-1-app` (PostgreSQL store and runnable project API). Depends on `phase-1-core`. Phase gate: `cd taskboard && go test ./...`.

Scope: only the nine `required_files` of `phase-1-app` are in scope. Task routes, Kanban UI, `taskboard/web/static/*`, Playwright, Docker, and k3s work belong to later phases and are deferred.

Rework amendment: the QA failure was `taskboard/web/assets.go:7:24: pattern static/*: no matching files found`. `taskboard/web/static/` is empty because its owning beads belong to phase-2-kanban. For phase-1-app, `taskboard/web/assets.go` must embed `templates/*` only. Removing `static/*` from the directive is the required fix, recorded under the `taskboard/web/assets.go` bead below.

Validator alignment: the SPEC HTTP route table is the only source of HTTP paths in this plan. The full table, including the deferred task routes, appears in the Integration contract.

## Bead map

### ta-3o6: taskboard/web/templates/index.html
- Scope: real server-rendered HTML shell served by the root page handler. Full task controls are deferred to phase-2-kanban.
- Architecture: `html/template` compatible; contains the `TaskBoard` heading, the project input, the `Add project` button, and a project selector region. No inline script that interpolates user text.
- Interfaces: template name `index.html`; no Go symbols.
- Depends on: none
- Consumed by: taskboard/web/assets.go, taskboard/internal/web/app.go
- Acceptance: the file is nonempty and contains `TaskBoard`, a project name input, and an `Add project` control.
- Verify: `cd taskboard && go build ./web/...`

### ta-hzj: taskboard/web/assets.go
- Scope: `embed.FS` variable `FS` for phase-1-app.
- Architecture: the directive must be `//go:embed templates/*` only. `static/*` is added in phase-2-kanban, when `taskboard/web/static/` is created.
- Interfaces: `var FS embed.FS`.
- Depends on: taskboard/web/templates/index.html
- Consumed by: taskboard/internal/web/app.go, taskboard/internal/web/api_test.go
- Acceptance: `go build ./web/...` succeeds; `fs.ReadFile(FS, "templates/index.html")` returns the page; no `static/*` pattern appears in phase-1-app.
- Verify: `cd taskboard && go build ./web/...`

### ta-71y: taskboard/internal/store/postgres.go
- Status: closed bead; file present. Reopen only if the file is missing or stubbed.
- Scope: `PostgresStore`, `NewPostgresStore`, implementing the `Store` interface from `taskboard/internal/store/store.go`.
- Architecture: pgx/v5/stdlib through `database/sql`; `$1` placeholders; identity keys; schema creation in a transaction guarded by `pg_advisory_xact_lock`.
- Interfaces: `NewPostgresStore(ctx context.Context, url string) (*PostgresStore, error)`; `Ping`, `GetProject`, `CreateProject`, `ListProjects`, `DeleteProject`, `CreateTask`, `ListTasks`, `UpdateTask`, `DeleteTask`, `Close`.
- Depends on: taskboard/internal/store/store.go, taskboard/internal/model/model.go
- Consumed by: taskboard/internal/store/postgres_integration_test.go, taskboard/cmd/taskboard/main.go
- Acceptance: repeated and concurrent schema init is idempotent; ascending ID order; project delete cascades to tasks; missing rows return `store.ErrNotFound`; the connection URL never appears in errors.
- Verify: `cd taskboard && go test ./internal/store/...`

### ta-5rq: taskboard/internal/store/store_test.go
- Status: closed bead; file present. Reopen only if the file is missing or stubbed.
- Scope: SQLite contract tests for the shared store behavior, using `t.TempDir()` databases.
- Architecture: standard `testing` package; no dependence on test execution order.
- Acceptance: fresh schema is empty and repeated init is idempotent; ascending ID order; default status `todo` and priority `medium`; status, priority, and AND filters; PATCH preserves omitted fields; project delete cascades; foreign keys enforced; data persists across reopen; Unicode code-point boundaries.
- Verify: `cd taskboard && go test ./internal/store/...`

### ta-9ik: taskboard/internal/store/postgres_integration_test.go
- Status: closed bead; file present. Reopen only if the file is missing or stubbed.
- Scope: PostgreSQL integration tests for the same store contract.
- Architecture: runs only when `TASKBOARD_TEST_POSTGRES_URL` is nonempty; uses a disposable database and uniquely named objects; cleans up only its own rows.
- Acceptance: skips cleanly when the variable is empty; when configured, covers cascade, ordering, filters, timestamps, and schema idempotency; never drops shared schemas.
- Verify: `cd taskboard && go test ./internal/store/...`

### ta-5kn: taskboard/cmd/taskboard/main.go
- Scope: process entrypoint: load config, select the store, build the web app, serve HTTP, shut down gracefully.
- Architecture: `config.LoadConfig()`; `sqlite.NewSQLiteStore` or `postgres.NewPostgresStore`; `web.NewApp(s)`; `app.Handler()` passed to `http.Server`; no `http.DefaultServeMux`.
- Depends on: taskboard/internal/config/config.go, taskboard/internal/store/sqlite.go, taskboard/internal/store/postgres.go, taskboard/internal/web/app.go
- Acceptance: invalid config exits nonzero without printing the database URL; SIGINT and SIGTERM trigger shutdown with a timeout, then `Store.Close()`; default bind is `127.0.0.1:8080`.
- Verify: `cd taskboard && go build ./cmd/taskboard`

### ta-12r: taskboard/internal/web/app.go
- Status: in_progress.
- Scope: `App`, `NewApp`, `Handler`, the index page handler, and `writeJSON`.
- Architecture: `http.ServeMux` with method-aware patterns; the index page handler renders `templates/index.html` from `embedweb.FS`; static routes are registered only when the static sub-filesystem exists.
- Interfaces: `NewApp(s store.Store) *App`; `(*App).Handler() http.Handler`; `writeJSON(w http.ResponseWriter, status int, v any)`.
- Depends on: taskboard/internal/store/store.go, taskboard/web/assets.go
- Consumed by: taskboard/internal/web/api.go, taskboard/cmd/taskboard/main.go
- Acceptance: the index page returns 200 HTML containing TaskBoard; unknown paths return 404; no global database handle.
- Verify: `cd taskboard && go test ./internal/web/...`

### ta-83w: taskboard/internal/web/api.go
- Scope: project API handlers plus health and readiness routes, registered by `(*App).registerAPI()`.
- Architecture: phase-1-app registers `GET /api/health`, `GET /api/ready`, `GET /api/projects`, `POST /api/projects`, `DELETE /api/projects/{project_id}`. Task routes are deferred to phase-2-kanban.
- Depends on: taskboard/internal/web/app.go, taskboard/internal/store/store.go
- Consumed by: taskboard/internal/web/api_test.go, taskboard/cmd/taskboard/main.go
- Acceptance: health returns `{"status":"ok"}`; ready returns `{"status":"ready"}` or 503 `{"error":"database unavailable"}`; create returns 201; blank or overlong names return 400; delete returns 204 or 404; list returns `[]` when empty; storage failure returns a generic 500.
- Verify: `cd taskboard && go test ./internal/web/...`

### ta-or0: taskboard/internal/web/api_test.go
- Scope: `httptest` tests for the phase-1-app routes.
- Architecture: SQLite store in `t.TempDir()`; a failing store double covers the 503 and 500 paths.
- Acceptance: exact health and ready JSON; project round trip with state checked afterward; name validation; malformed IDs return 400; missing project returns 404; empty list is `[]`; ascending order; delete body is empty; database failure returns a safe error.
- Verify: `cd taskboard && go test ./internal/web/...`

## Integration contract

Full SPEC HTTP route table. Phase-1-app implements the first five rows in `taskboard/internal/web/api.go`. The four task rows are phase-2-kanban and are not registered in phase-1-app.

| Method | Path | Success | Owner |
|---|---|---|---|
| GET | `/api/health` | 200 `{"status":"ok"}` | ta-83w (phase-1-app) |
| GET | `/api/ready` | 200 `{"status":"ready"}`; 503 `{"error":"database unavailable"}` | ta-83w (phase-1-app) |
| GET | `/api/projects` | 200 JSON array ordered by ID | ta-83w (phase-1-app) |
| POST | `/api/projects` | 201 project JSON | ta-83w (phase-1-app) |
| DELETE | `/api/projects/{project_id}` | 204 empty body | ta-83w (phase-1-app) |
| GET | `/api/projects/{project_id}/tasks` | 200 JSON array ordered by ID | phase-2-kanban (deferred) |
| POST | `/api/projects/{project_id}/tasks` | 201 task JSON | phase-2-kanban (deferred) |
| PATCH | `/api/tasks/{task_id}` | 200 updated task JSON | phase-2-kanban (deferred) |
| DELETE | `/api/tasks/{task_id}` | 204 empty body | phase-2-kanban (deferred) |

The index page is served by the handler in `taskboard/internal/web/app.go` (owner ta-12r).

- `taskboard/cmd/taskboard/main.go` loads config, opens one store per `TASKBOARD_DB_DRIVER`, calls `web.NewApp(s)`, and serves `app.Handler()`.
- `registerAPI()` in `taskboard/internal/web/api.go` registers the five phase-1-app API routes; `NewApp` in `taskboard/internal/web/app.go` registers the index page handler.
- Exports: `taskboard/internal/store/postgres.go` exports `NewPostgresStore` and `PostgresStore`; `taskboard/internal/store/sqlite.go` exports `NewSQLiteStore` and `SQLiteStore`; `taskboard/internal/web/app.go` exports `NewApp` and `App`; `taskboard/web/assets.go` exports `FS`.
- Wire order: config and model, then `taskboard/internal/store/store.go`, then `sqlite.go` and `postgres.go`, then `taskboard/web/assets.go` and templates, then `taskboard/internal/web/app.go`, then `taskboard/internal/web/api.go`, then `taskboard/cmd/taskboard/main.go`.

## Future considerations

- Task routes, Kanban UI, and `taskboard/web/static/*` belong to phase-2-kanban.
- Playwright, Docker, and k3s belong to phases 3 through 5.
