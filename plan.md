# Implementation plan

Active phase: `phase-1-app` (PostgreSQL store and runnable project API). Depends on `phase-1-core`, whose files (`go.mod`, `config.go`, `model.go`, `store.go`, `sqlite.go`) already exist and are out of scope for new beads.

Phase gate (whole-project, behavioral): `cd taskboard && go test ./...`
Manual smoke for this phase: `GET /api/health`, `GET /`, create/list/delete project via `/api/projects`.

## Integration contract

**Entrypoint:** `taskboard/cmd/taskboard/main.go` (bead ta-5kn)

**Dependency wiring order:**
1. `config.LoadConfig()` reads TASKBOARD_* environment variables (phase-1-core).
2. Driver selection: `NewSQLiteStore` (sqlite.go) or `NewPostgresStore` (postgres.go, bead ta-71y) based on TASKBOARD_DB_DRIVER.
3. `web.NewApp(store, assets)` in `taskboard/internal/web/app.go` (bead ta-12r). No global database handle.
4. Static/template serving from the embed variable in `taskboard/web/assets.go` (bead ta-qfs), which embeds `templates/*` only in this phase.
5. HTTP server on TASKBOARD_HOST:TASKBOARD_PORT; graceful SIGINT/SIGTERM shutdown closes the store.

**SPEC HTTP route table (exact, registered in app.go / api.go):**

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

Task routes are registered by the route table but task behavior is phase-2; task handler behavior is deferred to phase-2, and no phase-1-app test may assert task CRUD.

**Per-file exports:**
- `taskboard/internal/store/postgres.go`: `PostgresStore` struct, `NewPostgresStore` function (implements the `Store` interface from store.go).
- `taskboard/internal/store/store_test.go`: test functions only (no exports).
- `taskboard/internal/store/postgres_integration_test.go`: test functions only, gated on `TASKBOARD_TEST_POSTGRES_URL`.
- `taskboard/internal/web/app.go`: `App` struct, `NewApp(store, assets)` function.
- `taskboard/internal/web/api.go`: route handler methods on `*App` for health, ready, and project endpoints.
- `taskboard/web/assets.go`: `FS` embed variable (`//go:embed templates/*` only in this phase).
- `taskboard/web/templates/index.html`: no Go exports; the embedded HTML shell.
- `taskboard/internal/web/api_test.go`: test functions only.
- `taskboard/cmd/taskboard/main.go`: `main` function.

## Bead map

### ta-71y: taskboard/internal/store/postgres.go
- Scope: PostgreSQL implementation of the shared `Store` contract using `github.com/jackc/pgx/v5/stdlib` and `database/sql`, with `$1` placeholders.
- Architecture: architecture.md, Store interface and phase-1-app PostgreSQL store. Dialect SQL stays in taskboard/internal/store.
- Interfaces: `PostgresStore`, `NewPostgresStore(ctx, url string) (*PostgresStore, error)`, and all `Store` methods.
- Depends on: taskboard/internal/store/store.go (phase-1-core, not a bead in this phase), taskboard/internal/model/model.go.
- Consumed by: taskboard/cmd/taskboard/main.go (ta-5kn), taskboard/internal/store/postgres_integration_test.go (ta-9ik).
- Acceptance:
  - Schema is created inside a transaction guarded by `pg_advisory_xact_lock`; running init twice or concurrently is safe and idempotent.
  - Projects and tasks list in ascending ID order; tasks.project_id is indexed.
  - Task `ON DELETE CASCADE` removes tasks when their project is deleted; foreign key violations are reported for tasks with missing projects.
  - Empty list results serialize as an empty slice (never nil) at the store boundary.
  - Connection establishment uses a bounded context; URL and password never appear in returned errors.
- Verify: `cd taskboard && go test ./...` (compiles; PostgreSQL tests skip when TASKBOARD_TEST_POSTGRES_URL is unset).

### ta-5rq: taskboard/internal/store/store_test.go
- Scope: SQLite-backed contract tests using `t.TempDir()`, never the runtime taskboard.db.
- Architecture: store_test.go covers DDL, project/task CRUD, filters, cascade, FK enforcement.
- Interfaces: none (test-only file).
- Depends on: taskboard/internal/store/sqlite.go (phase-1-core), taskboard/internal/store/store.go.
- Consumed by: none.
- Acceptance (each name maps to SPEC §8 store tests):
  - `TestSchemaIdempotent`: opening the same database file twice succeeds and leaves tables intact.
  - `TestProjectsAscendingIDs`: created projects list in ascending ID order.
  - `TestTaskDefaults`: new task has status `todo`, priority `medium`, valid UTC RFC3339 timestamps.
  - `TestTaskFilters`: status-only, priority-only, and combined filters return the AND-matched subset.
  - `TestPatchPreservesOmittedFields`: patching title, description, priority, or status changes only that field.
  - `TestDeleteProjectCascadesTasks`: deleting a project removes its tasks.
  - `TestForeignKeyEnforced` and `TestPersistenceAcrossReopen`: a task with a missing project is rejected; data survives close and reopen.
  - `TestUnicodeBoundaries`: name 80 vs 81 runes, title 120 vs 121 runes, description 1000 vs 1001 runes use rune counts, and SQL-injection-looking input is stored as literal text.
- Verify: `cd taskboard && go test ./...`

### ta-9ik: taskboard/internal/store/postgres_integration_test.go
- Scope: Run the same contract against PostgreSQL when `TASKBOARD_TEST_POSTGRES_URL` is set; use a dedicated disposable database, unique test object names, and clean up only own data.
- Architecture: postgres_integration_test.go gated on TASKBOARD_TEST_POSTGRES_URL.
- Interfaces: none (test-only file).
- Depends on: taskboard/internal/store/postgres.go (ta-71y).
- Consumed by: none.
- Acceptance:
  - Skips cleanly with `t.Skip` when the environment variable is empty; the default gate passes without PostgreSQL.
  - When configured: idempotent schema init, ascending ID ordering, cascade deletion, filter AND semantics, and persistence across reopen match store_test.go.
  - Never drops or resets shared schemas.
- Verify: `cd taskboard && go test ./...`

### ta-83w: taskboard/internal/web/api.go
- Scope: Project API handlers and health/ready endpoints returning the SPEC JSON shapes.
- Architecture: API table in architecture.md; error bodies are exactly `{"error":"..."}`.
- Interfaces: methods on `*App`: `handleHealth`, `handleReady`, `handleListProjects`, `handleCreateProject`, `handleDeleteProject`.
- Depends on: taskboard/internal/web/app.go (ta-12r), taskboard/internal/store/store.go, taskboard/internal/model/model.go.
- Consumed by: taskboard/internal/web/api_test.go (ta-or0), taskboard/cmd/taskboard/main.go (ta-5kn).
- Acceptance:
  - `GET /api/health` returns 200 `{"status":"ok"}`; `GET /api/ready` returns 200 `{"status":"ready"}` when the store pings, otherwise 503 `{"error":"database unavailable"}` with no SQL detail.
  - `GET /api/projects` returns `[]` when empty, otherwise ascending ID order.
  - `POST /api/projects` with blank or overlong name returns 400; success returns 201 with project JSON.
  - `DELETE /api/projects/{id}` returns 204 with empty body; missing project returns 404.
  - Non-integer or non-positive IDs return 400; storage failures return a generic 500 JSON error.
- Verify: `cd taskboard && go test ./internal/web/... -run 'TestAPI'`

### ta-or0: taskboard/internal/web/api_test.go
- Scope: httptest coverage of the phase-1-app project API against healthy SQLite.
- Architecture: api_test.go covers the REST route handlers for documented status codes and bodies.
- Interfaces: none (test-only file).
- Depends on: taskboard/internal/web/api.go (ta-83w), taskboard/internal/web/app.go (ta-12r).
- Consumed by: none.
- Acceptance (SPEC §8 API scenarios for projects):
  - Exact liveness JSON and ready JSON from a healthy SQLite store.
  - Project create/list/delete round trip asserts the list state changes.
  - Blank project name (empty and whitespace-only) returns 400; overlong name (81 runes) returns 400.
  - Empty lists serialize as `[]`; list order is ascending ID; 204 bodies are empty.
  - Missing project delete returns 404; malformed ID returns 400.
  - Readiness returns 503 with `{"error":"database unavailable"}` when the store fails a ping.
- Verify: `cd taskboard && go test ./internal/web/...`

### ta-12r: taskboard/internal/web/app.go
- Scope: `App` struct and `NewApp(store store.Store, assets fs.FS) *App`; router built on net/http ServeMux with method-aware patterns; serves `/` from html/template and `/static/` from assets.
- Architecture: App struct, NewApp, no global database handle.
- Interfaces: `App`, `NewApp`, `(*App).Routes() http.Handler`.
- Depends on: taskboard/internal/store/store.go, taskboard/web/assets.go (ta-qfs).
- Consumed by: taskboard/internal/web/api.go (ta-83w), taskboard/cmd/taskboard/main.go (ta-5kn).
- Acceptance:
  - All nine SPEC routes are registered with method-aware patterns; unknown routes return 404. Task route behavior is deferred to phase-2; no phase-1-app test asserts task CRUD.
  - `GET /` renders the embedded `index.html` via html/template and returns 200 HTML containing `TaskBoard`.
  - Does not interpolate user input into HTML.
- Verify: `cd taskboard && go test ./internal/web/... -run 'TestAPIIndexServesHTML'`

### ta-qfs: taskboard/web/assets.go
- Scope: `FS` embed variable with `//go:embed templates/*` only; static/* is added in phase-2-kanban.
- Architecture: FS embed variable; no business logic.
- Interfaces: `FS embed.FS`.
- Depends on: taskboard/web/templates/index.html (ta-dw6).
- Consumed by: taskboard/internal/web/app.go (ta-12r), taskboard/cmd/taskboard/main.go (ta-5kn).
- Acceptance:
  - Embedded `templates/index.html` is readable through `FS` and is nonempty.
  - Embed pattern is valid with no `..` and no static/* pattern yet.
- Verify: `cd taskboard && go test ./...`

### ta-dw6: taskboard/web/templates/index.html
- Scope: Real server-rendered HTML shell: `TaskBoard` heading, project name input with `Add project` control, and project list/selector region. Task controls and Kanban columns are phase-2 and are not required here.
- Architecture: index.html is the embedded template for GET `/`.
- Interfaces: template only, no Go exports.
- Depends on: none in this phase.
- Consumed by: taskboard/web/assets.go (ta-qfs).
- Acceptance:
  - Contains the heading `TaskBoard` and a page title.
  - Contains the project form markup for the phase-1 project API (name input and `Add project` button) so the page is a real shell, not a placeholder.
  - Does not require task controls, filters, Kanban columns, or client-side JavaScript handlers in this phase.
  - No inline user data is interpolated unsafely; template uses html/template syntax only where data is passed.
- Verify: `cd taskboard && go test ./...`

### ta-5kn: taskboard/cmd/taskboard/main.go
- Scope: `main` function: load config, select driver, construct store, construct app with assets, serve, handle SIGINT/SIGTERM with a shutdown timeout, then close the store.
- Architecture: entrypoint steps listed in the Integration contract.
- Interfaces: `main`.
- Depends on: taskboard/internal/config/config.go, taskboard/internal/store/sqlite.go, taskboard/internal/store/postgres.go (ta-71y), taskboard/internal/web/app.go (ta-12r), taskboard/web/assets.go (ta-qfs).
- Consumed by: none.
- Acceptance:
  - Invalid configuration exits nonzero with a useful message that never prints the database URL.
  - Default run serves `127.0.0.1:8080` with SQLite and no extra env vars.
  - `TASKBOARD_DB_DRIVER=postgres` without a URL fails startup; no silent SQLite fallback.
  - SIGTERM triggers HTTP shutdown and store Close.
- Verify: `cd taskboard && go build ./cmd/taskboard && go test ./...`

## YAGNI

Implement only what SPEC and architecture require for phase-1-app. Future work (task CRUD UI, filters, Playwright, Docker) belongs to later phases. Future considerations go in a separate section, not in acceptance bullets.

## Future considerations

- Phase-2 adds task handlers, static assets, and the Kanban UI.
- Phase-3 adds Playwright E2E.
