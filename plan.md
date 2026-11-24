# Implementation plan

Active phase: `phase-1-app` (PostgreSQL store and runnable project API). Phase verification: `cd taskboard && go test ./...`.

## Integration contract

**Entrypoint:** `taskboard/cmd/taskboard/main.go`

**Startup / dependency order (entrypoint):**

1. `main` calls `config.LoadConfig()` (from `taskboard/internal/config/config.go`) to read TASKBOARD_DB_DRIVER, TASKBOARD_DB_PATH, TASKBOARD_DATABASE_URL, TASKBOARD_HOST, TASKBOARD_PORT.
2. Based on TASKBOARD_DB_DRIVER, call `store.NewSQLiteStore` (`taskboard/internal/store/sqlite.go`) or `store.NewPostgresStore` (`taskboard/internal/store/postgres.go`). Both return the shared `Store` interface from `taskboard/internal/store/store.go`. Schema init is idempotent; PostgreSQL schema setup runs under a transactional advisory lock.
3. Build the HTTP app with `web.NewApp(store, assets)` (`taskboard/internal/web/app.go`), where `assets` is the embed FS variable from `taskboard/web/assets.go`.
4. Register routes on the standard library `net/http` mux: API handlers from `taskboard/internal/web/api.go`, `GET /` rendering `index.html` via `html/template`, and static files.
5. Listen on host:port from config.
6. On SIGTERM/SIGINT, perform graceful server shutdown and close the store.

**HTTP routes (SPEC verbatim):**

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

**Per-file exports (architecture ownership):**

- `taskboard/internal/config/config.go`: `Config` struct, `LoadConfig` function.
- `taskboard/internal/model/model.go`: `Project` struct, `Task` struct, validation functions.
- `taskboard/internal/store/store.go`: `Store` interface, repository error constants, DDL constants.
- `taskboard/internal/store/sqlite.go`: `NewSQLiteStore` function.
- `taskboard/internal/store/postgres.go`: `NewPostgresStore` function.
- `taskboard/web/assets.go`: embed FS variable (templates and static assets).
- `taskboard/internal/web/app.go`: `App` struct, `NewApp` function.
- `taskboard/internal/web/api.go`: API route handler methods on `App`.
- `taskboard/cmd/taskboard/main.go`: `main` entrypoint.
- `taskboard/web/templates/index.html`: HTML layout template.

## Bead map

### ta-11e: taskboard/internal/store/postgres.go
- Scope: PostgreSQL implementation of the `Store` interface via `NewPostgresStore`, using pgx stdlib and `$1` placeholders.
- Architecture: `taskboard/internal/store/postgres.go` implements PostgreSQL storage with transactional advisory locks for schema setup.
- Interfaces: `NewPostgresStore`, plus methods satisfying `Store`.
- Depends on: `taskboard/internal/store/store.go`.
- Consumed by: `taskboard/cmd/taskboard/main.go`, `taskboard/internal/store/postgres_integration_test.go`.
- Acceptance:
  - Schema init is idempotent under concurrent startup (advisory lock serializes DDL).
  - Project CRUD: create returns the stored project; list returns projects in ascending ID order; delete removes the project and cascades to its tasks.
  - Task filtering by status, priority, and both together works with parameterized queries.
  - Unicode code-point length limits are enforced before insert.
  - Verify: `cd taskboard && go test ./...`

### ta-nw9: taskboard/internal/store/store_test.go
- Scope: SQLite unit tests for the shared store contract (the SQLite store is exercised through the `Store` interface).
- Architecture: unit tests use t.TempDir() and cover CRUD, cascade deletion, Unicode length checks, and filtering.
- Depends on: `taskboard/internal/store/sqlite.go`, `taskboard/internal/store/store.go`.
- Consumed by: none.
- Acceptance:
  - Fresh schema is empty; repeated initialization is idempotent.
  - Create and list projects with ascending IDs.
  - Create tasks with default status and priority and valid UTC timestamps.
  - Filter by status, priority, and both together.
  - Patch all mutable fields and preserve omitted fields.
  - Delete project and verify cascading task deletion; foreign keys enforced; data persists across close and reopen.
  - Verify: `cd taskboard && go test ./...`

### ta-5qb: taskboard/internal/store/postgres_integration_test.go
- Scope: PostgreSQL integration tests, skipped unless TASKBOARD_TEST_POSTGRES_URL is set.
- Architecture: exercises advisory-lock schema setup, identity PKs, UTC timestamps, `$1` placeholders, and cascade deletion.
- Depends on: `taskboard/internal/store/postgres.go`.
- Consumed by: none.
- Acceptance:
  - Same functional scenarios as the SQLite contract run against a live database.
  - Concurrent schema initialization completes without error.
  - Test is skipped (not failed) when the URL env var is absent.
  - Verify: `cd taskboard && go test ./...`

### ta-83w: taskboard/internal/web/api.go
- Scope: REST handler methods on `App` for the SPEC route table.
- Architecture: handlers return JSON, errors as `{"error": "message"}`, and status codes 200, 201, 204, 400, 404, 500, 503.
- Interfaces: handler methods on `*App` for every route in the integration contract.
- Depends on: `taskboard/internal/web/app.go`.
- Consumed by: `taskboard/internal/web/api_test.go`, `taskboard/cmd/taskboard/main.go`.
- Acceptance:
  - `GET /api/health` returns 200 with `{"status":"ok"}`.
  - `GET /api/ready` returns 200 when the database pings, 503 otherwise.
  - `POST /api/projects` returns 201 with project JSON; `GET /api/projects` returns ascending IDs; `DELETE /api/projects/{id}` returns 204, and 404 for unknown IDs.
  - Server-generated fields (created_at, updated_at, status on create) cannot be overridden by clients.
  - Verify: `cd taskboard && go test ./...`

### ta-8l7: taskboard/internal/web/api_test.go
- Scope: httptest-based tests of the REST API.
- Architecture: covers status codes, JSON payloads, error structure, and filter combinations.
- Depends on: `taskboard/internal/web/api.go`, `taskboard/internal/web/app.go`.
- Consumed by: none.
- Acceptance:
  - Health and ready endpoints return the specified bodies and status codes.
  - Project create returns 201; list returns ascending IDs; delete returns 204, and a repeat delete returns 404.
  - Task list honors status and priority filters, alone and combined.
  - Invalid input returns 400 with `{"error": "..."}`.
  - Verify: `cd taskboard && go test ./...`

### ta-12r: taskboard/internal/web/app.go
- Scope: `App` struct and `NewApp` constructor that build the router.
- Architecture: builds the net/http router, wires the store and embedded assets, and serves `GET /` via `html/template`.
- Interfaces: `App`, `NewApp(store Store, assets fs.FS) *App`.
- Depends on: `taskboard/internal/store/store.go`, `taskboard/web/assets.go`.
- Consumed by: `taskboard/internal/web/api.go`, `taskboard/cmd/taskboard/main.go`.
- Acceptance:
  - `GET /` renders the real `index.html` shell through `html/template` with status 200 and an HTML content type.
  - `GET /api/health` and `GET /api/ready` are routed and return their specified responses.
  - Static files under `/static/` are served from the embedded FS.
  - Verify: `cd taskboard && go test ./...`

### ta-qfs: taskboard/web/assets.go
- Scope: embed directive exposing templates and static assets.
- Architecture: embeds `taskboard/web/templates/*` as the FS variable used by the app.
- Interfaces: embed FS variable exported from package `web`.
- Depends on: `taskboard/web/templates/index.html`.
- Consumed by: `taskboard/internal/web/app.go`, `taskboard/cmd/taskboard/main.go`.
- Acceptance:
  - The embedded FS contains `templates/index.html` and parses successfully via `html/template`.
  - Requesting the index template from the FS returns non-empty HTML containing the TaskBoard heading.
  - Verify: `cd taskboard && go test ./...`

### ta-dw6: taskboard/web/templates/index.html
- Scope: HTML layout shell rendered by `GET /`.
- Architecture: contains the TaskBoard heading, project creation form, and To Do / Doing / Done Kanban column headers.
- Depends on: none.
- Consumed by: `taskboard/web/assets.go`, `taskboard/internal/web/app.go`.
- Acceptance:
  - Rendered page contains the TaskBoard title and the To Do, Doing, Done column headers.
  - Project creation form with a name input and submit button is present.
  - Template parses with `html/template` without error.
  - Verify: `cd taskboard && go test ./...`

### ta-5kn: taskboard/cmd/taskboard/main.go
- Scope: process entrypoint following the startup order in the integration contract.
- Architecture: loads config, selects the store by driver, runs schema setup, builds the app, serves HTTP, and shuts down gracefully on SIGTERM/SIGINT.
- Interfaces: `main()`.
- Depends on: `taskboard/internal/config/config.go`, `taskboard/internal/store/sqlite.go`, `taskboard/internal/store/postgres.go`, `taskboard/internal/web/app.go`, `taskboard/web/assets.go`.
- Consumed by: `taskboard/Dockerfile` (later phase).
- Acceptance:
  - Driver selection uses TASKBOARD_DB_DRIVER: sqlite and postgres both start.
  - Server listens on TASKBOARD_HOST and TASKBOARD_PORT; `/api/health` responds 200 once started.
  - SIGTERM or SIGINT triggers graceful shutdown and closes the store.
  - Verify: `cd taskboard && go test ./...`

## Phase context

Active delivery phase `phase-1-app` only. Later phases start automatically after QA `all_passed`.
