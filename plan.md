# Implementation plan: phase-1-app (PostgreSQL store and runnable project API)

Active phase: `phase-1-app`. Depends on `phase-1-core` (go.mod, go.sum, README.md, .gitignore, config, model, store.go, sqlite.go already delivered).
Phase verification: `cd taskboard && go test ./...`

## Integration contract

SPEC HTTP routes (SPEC §6, architecture HTTP table). All routes are listed here so the contract matches the SPEC route table exactly:

| Method | Path | Phase that implements the handler |
|---|---|---|
| GET | `/api/health` | phase-1-app (`taskboard/internal/web/api.go`) |
| GET | `/api/ready` | phase-1-app (`taskboard/internal/web/api.go`) |
| GET | `/api/projects` | phase-1-app (`taskboard/internal/web/api.go`) |
| POST | `/api/projects` | phase-1-app (`taskboard/internal/web/api.go`) |
| DELETE | `/api/projects/{project_id}` | phase-1-app (`taskboard/internal/web/api.go`) |
| GET | `/api/projects/{project_id}/tasks` | phase-2-kanban (`taskboard/internal/web/api.go` extended) |
| POST | `/api/projects/{project_id}/tasks` | phase-2-kanban (`taskboard/internal/web/api.go` extended) |
| PATCH | `/api/tasks/{task_id}` | phase-2-kanban (`taskboard/internal/web/api.go` extended) |
| DELETE | `/api/tasks/{task_id}` | phase-2-kanban (`taskboard/internal/web/api.go` extended) |

Also served in this phase: `GET /` (embedded `taskboard/web/templates/index.html`).

Wiring:
- Entrypoint `taskboard/cmd/taskboard/main.go` calls `config.LoadConfig()`, selects `store.NewSQLiteStore` or `store.NewPostgresStore` by `TASKBOARD_DB_DRIVER`, then calls `web.NewApp(dbStore, embedFS)` and serves `App.Handler()` on `TASKBOARD_HOST:TASKBOARD_PORT`.
- Dependency order: `taskboard/internal/store/store.go` -> `taskboard/internal/store/postgres.go` -> `taskboard/internal/web/app.go` -> `taskboard/internal/web/api.go` -> `taskboard/cmd/taskboard/main.go`. `taskboard/web/assets.go` precedes `taskboard/internal/web/app.go`.
- Exported symbols per file: `store.NewPostgresStore` (postgres.go); `web.App`, `web.NewApp`, `(*App).Handler` (app.go); `FS` (taskboard/web/assets.go); `main` (taskboard/cmd/taskboard/main.go).

## Bead map

### ta-11e: taskboard/internal/store/postgres.go
- Scope: PostgreSQL implementation of the Store interface using `pgx/v5/stdlib` and `database/sql`.
- Architecture: `taskboard/internal/store/postgres.go` implements PostgreSQL storage with transactional advisory locks for schema setup.
- Interfaces: `NewPostgresStore`; implements every method of `store.Store`.
- Depends on: `taskboard/internal/store/store.go`.
- Consumed by: `taskboard/internal/store/postgres_integration_test.go`, `taskboard/cmd/taskboard/main.go`.
- Acceptance: schema creation runs in a transaction holding a PostgreSQL advisory lock; repeated and concurrent starts are idempotent; identity primary keys; `$1` placeholders only; no user input interpolated into SQL; `ON DELETE CASCADE` on `tasks.project_id`; index on `tasks.project_id`; UTC RFC3339 timestamps; ascending-ID ordering.
- Verify: `cd taskboard && go test ./internal/store/...`

### ta-nw9: taskboard/internal/store/store_test.go
- Scope: shared store contract tests against the SQLite store using `t.TempDir()`.
- Architecture: `taskboard/internal/store/store_test.go` covers SQLite schema init, CRUD, cascade deletion, Unicode length checks, and filtering.
- Interfaces: test-only; uses `store.NewSQLiteStore`.
- Depends on: `taskboard/internal/store/sqlite.go`.
- Consumed by: none.
- Acceptance: fresh schema empty and repeated init idempotent; projects list in ascending ID; tasks default to status `todo` and priority `medium`; filters by status, priority, and both combined; patch preserves omitted fields; project delete cascades to tasks; FK enforcement and persistence across close/reopen; Unicode code-point boundaries (80/120/1000); parameterized SQL.
- Verify: `cd taskboard && go test ./internal/store/...`

### ta-5qb: taskboard/internal/store/postgres_integration_test.go
- Scope: PostgreSQL contract tests, run only when `TASKBOARD_TEST_POSTGRES_URL` is set.
- Architecture: `taskboard/internal/store/postgres_integration_test.go` exercises the store against a disposable PostgreSQL database.
- Interfaces: test-only; uses `store.NewPostgresStore`.
- Depends on: `taskboard/internal/store/postgres.go`.
- Consumed by: none.
- Acceptance: isolated schema or uniquely named objects; cleans up only its own rows; covers cascade deletion, ordering, filters, timestamp encoding, schema idempotency; never touches a production database; skips cleanly when the env var is empty.
- Verify: `cd taskboard && go test ./internal/store/...` (with and without the env var set)

### ta-83w: taskboard/internal/web/api.go
- Scope: project REST handlers and health/ready in this phase; task handlers arrive in phase-2-kanban.
- Architecture: `taskboard/internal/web/api.go` defines route handler methods on `App`.
- Interfaces: methods on `*App` for `GET /api/health`, `GET /api/ready`, `GET /api/projects`, `POST /api/projects`, `DELETE /api/projects/{project_id}`; JSON error helper emitting `{"error":"..."}`.
- Depends on: `taskboard/internal/web/app.go`, `taskboard/internal/store/store.go`.
- Consumed by: `taskboard/internal/web/api_test.go`, `taskboard/cmd/taskboard/main.go` (via `NewApp`).
- Acceptance: health returns `{"status":"ok"}` 200 regardless of DB; ready returns `{"status":"ready"}` 200 or 503 `{"error":"database unavailable"}`, with no SQL or credential leakage; project create returns 201 and rejects blank or overlong names with 400; list is ascending ID and `[]` when empty; delete returns 204 with empty body or 404 when missing; non-client DB errors return 500 with a generic message.
- Verify: `cd taskboard && go test ./internal/web/...`

### ta-12r: taskboard/internal/web/app.go
- Scope: App struct, constructor, ServeMux wiring, and index page handler.
- Architecture: `taskboard/internal/web/app.go` builds the HTTP router; no mutable global database handle.
- Interfaces: `type App struct`, `func NewApp(s store.Store, assets fs.FS) *App`, `func (a *App) Handler() http.Handler`.
- Depends on: `taskboard/internal/store/store.go`, `taskboard/web/assets.go`.
- Consumed by: `taskboard/internal/web/api.go`, `taskboard/internal/web/api_test.go`, `taskboard/cmd/taskboard/main.go`.
- Acceptance: uses `net/http` ServeMux with method-aware patterns; `GET /` serves embedded `templates/index.html` as `text/html`; no SQL statements in this file.
- Verify: `cd taskboard && go test ./internal/web/...`

### ta-qfs: taskboard/web/assets.go
- Scope: embed directive for templates only in this phase; static assets are added in phase-2-kanban.
- Architecture: `taskboard/web/assets.go` exports `FS` and embeds HTML templates.
- Interfaces: `var FS embed.FS` with `//go:embed templates/*`.
- Depends on: `taskboard/web/templates/index.html`.
- Consumed by: `taskboard/internal/web/app.go`.
- Acceptance: `FS` contains `templates/index.html`; no business logic.
- Verify: `cd taskboard && go build ./...`

### ta-dw6: taskboard/web/templates/index.html
- Scope: real server-rendered HTML shell with `TaskBoard` heading and project controls.
- Architecture: `taskboard/web/templates/index.html` is the HTML layout template with no client-side JavaScript.
- Interfaces: none exported; embedded by `taskboard/web/assets.go`.
- Depends on: none.
- Consumed by: `taskboard/web/assets.go`, `taskboard/internal/web/app.go`.
- Acceptance: contains `TaskBoard` heading, project name input, and `Add project` button; no inline script; valid HTML5 document.
- Verify: `cd taskboard && go test ./internal/web/...`

### ta-5kn: taskboard/cmd/taskboard/main.go
- Scope: process entrypoint with config loading, store selection, HTTP server, graceful shutdown.
- Architecture: `taskboard/cmd/taskboard/main.go` handles startup, configuration, database init, and SIGINT/SIGTERM shutdown.
- Interfaces: `func main()`.
- Depends on: `taskboard/internal/web/app.go`, `taskboard/internal/store/postgres.go`, `taskboard/internal/store/sqlite.go`, `taskboard/internal/config/config.go`.
- Consumed by: none (process entrypoint).
- Acceptance: invalid config exits nonzero with a useful message and no URL or password in output; listens on `127.0.0.1:8080` by default; honors `TASKBOARD_DB_DRIVER`; shuts down gracefully on SIGTERM with a timeout and closes the DB.
- Verify: `cd taskboard && go build ./cmd/taskboard && go test ./...`

### ta-8l7: taskboard/internal/web/api_test.go
- Scope: httptest tests for project routes and health/ready endpoints.
- Architecture: `taskboard/internal/web/api_test.go` covers status codes, error JSON, and project round trips.
- Interfaces: test-only; uses `web.NewApp` with a temporary SQLite store.
- Depends on: `taskboard/internal/web/api.go`, `taskboard/internal/web/app.go`.
- Consumed by: none.
- Acceptance: exact liveness and ready JSON against healthy SQLite; project create/list/delete round trip with state assertions; blank or overlong name rejected with 400 and `{"error":...}`; missing project delete returns 404; empty list is `[]`; 204 body empty; list order ascending ID; ready returns 503 when the store is closed.
- Verify: `cd taskboard && go test ./internal/web/...`

## Verification

- Phase gate: `cd taskboard && go test ./...` run from `tasktracker/mayor/rig`.
- Bead coverage: exactly the nine paths above, each mapped to one bead ID from `bd list --limit=0 `.

## Future considerations

- phase-2-kanban adds `taskboard/internal/web/app_test.go`, `taskboard/internal/web/ui_content_test.go`, `taskboard/web/static/app.js`, `taskboard/web/static/style.css`, and the task routes listed in the Integration contract. Not in this phase's bead map.
- Task REST coverage (PATCH and task CRUD assertions) belongs to phase-2-kanban.
