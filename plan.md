# Implementation plan

Active phase: `phase-1-app` (PostgreSQL store and runnable project API).
Phase verification: `cd taskboard && go test ./...`
Every Verify command runs tests only. Dependency resolution and package mutation are out of scope.

## Integration contract

**Entrypoint:** `taskboard/cmd/taskboard/main.go`

**Startup / dependency order:**

1. `config.LoadConfig` reads TASKBOARD_DB_DRIVER, TASKBOARD_DB_PATH, TASKBOARD_DATABASE_URL, TASKBOARD_HOST, TASKBOARD_PORT.
2. Store selection: `NewSQLiteStore` or `NewPostgresStore`, each running schema init. A PostgreSQL failure aborts startup; there is no silent SQLite fallback.
3. `web.NewApp(store, assets)` builds the App with the embedded `FS` from `taskboard/web/assets.go`.
4. Route registration on the stdlib ServeMux for the 9 API routes plus `/` and `/static/*`.
5. `http.ListenAndServe` on TASKBOARD_HOST:TASKBOARD_PORT.
6. SIGINT/SIGTERM triggers graceful shutdown with a context timeout, then the store is closed.

**HTTP routes (SPEC table):**

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

- `taskboard/internal/store/store.go`: `Store` interface, `ErrNotFound`, DDL constants.
- `taskboard/internal/store/sqlite.go`: `SQLiteStore`, `NewSQLiteStore`.
- `taskboard/internal/store/postgres.go`: `PostgresStore`, `NewPostgresStore`.
- `taskboard/internal/config/config.go`: `Config`, `LoadConfig`.
- `taskboard/web/assets.go`: embedded `FS`.
- `taskboard/internal/web/app.go`: `App`, `NewApp`.
- `taskboard/internal/web/api.go`: API route handler methods on `App`.
- `taskboard/cmd/taskboard/main.go`: `main` only.

## Bead map

### ta-11e: taskboard/internal/store/postgres.go
- Scope: PostgreSQL store implementation.
- Architecture: architecture.md, Module ownership and phase-1-app requirements.
- Interfaces: `PostgresStore`, `NewPostgresStore`; implements `store.Store`.
- Depends on: `taskboard/internal/store/store.go` (phase-1-core).
- Consumed by: `taskboard/cmd/taskboard/main.go`, `taskboard/internal/store/postgres_integration_test.go`.
- Acceptance:
  - Schema creation is idempotent and runs inside a transaction holding `pg_advisory_xact_lock`.
  - All SQL uses `$1` placeholders; no string concatenation of user input.
  - Primary keys are identity columns.
  - No silent SQLite fallback: connection or schema failure returns an error.
  - Foreign keys cascade so deleting a project removes its tasks.
- Verify: `cd taskboard && go test -timeout 30s -count=1 ./internal/store/...`

### ta-82r: taskboard/internal/store/store_test.go
- Scope: SQLite store contract tests.
- Architecture: architecture.md, Unit tests.
- Depends on: `taskboard/internal/store/sqlite.go`.
- Consumed by: none.
- Acceptance:
  - Tests use `t.TempDir()` for isolated SQLite files.
  - Covers schema init, project and task CRUD, cascade delete, Unicode code-point length limits, and combined status/priority filters.
  - Each test asserts a specific SPEC requirement; no always-pass tests.
- Verify: `cd taskboard && go test -timeout 30s -count=1 ./internal/store/...`

### ta-5qb: taskboard/internal/store/postgres_integration_test.go
- Scope: PostgreSQL integration tests, gated on `TASKBOARD_TEST_POSTGRES_URL`.
- Architecture: architecture.md, PostgreSQL Integration Testing.
- Depends on: `taskboard/internal/store/postgres.go`.
- Consumed by: none.
- Acceptance:
  - Skips cleanly when `TASKBOARD_TEST_POSTGRES_URL` is unset.
  - When set, exercises idempotent schema init under advisory lock, identity PKs, UTC timestamp precision, `$1` placeholder behavior, and FK cascade delete.
- Verify: `cd taskboard && go test -timeout 30s -count=1 ./internal/store/...`

### ta-5kn: taskboard/cmd/taskboard/main.go
- Scope: entrypoint wiring only.
- Architecture: architecture.md, Server Wiring Pattern.
- Interfaces: `main()`.
- Depends on: `taskboard/internal/config/config.go`, `taskboard/internal/store/postgres.go`, `taskboard/internal/store/sqlite.go`, `taskboard/internal/web/app.go`, `taskboard/web/assets.go`.
- Consumed by: Dockerfile (phase-4).
- Acceptance:
  - Calls `config.LoadConfig`, then the store matching TASKBOARD_DB_DRIVER, then `web.NewApp(store, assets)`.
  - Listens on TASKBOARD_HOST:TASKBOARD_PORT.
  - Handles SIGTERM: graceful shutdown with context timeout, then closes the DB.
  - PostgreSQL store failure exits non-zero; no SQLite fallback.
- Verify: `cd taskboard && go test -timeout 30s -count=1 ./cmd/taskboard/...`

### ta-83w: taskboard/internal/web/api.go
- Scope: REST handler methods on `App`.
- Architecture: architecture.md, Standardized HTTP API Table.
- Interfaces: handler methods on `App` for the 9 API routes.
- Depends on: `taskboard/internal/web/app.go`, `taskboard/internal/store/store.go`.
- Consumed by: `taskboard/internal/web/api_test.go`.
- Acceptance:
  - `GET /api/projects` returns 200 ordered by ascending ID; empty list serializes as `[]`.
  - `POST /api/projects` returns 201; 400 on blank or overlong name.
  - `DELETE /api/projects/{project_id}` returns 204; 404 if missing.
  - Task list supports combined `status` and `priority` AND filters; 400 on invalid enum; 404 if project missing.
  - `POST` task returns 201; `PATCH` task returns 200 and rejects empty or explicit-null fields with 400; `DELETE` task returns 204 or 404.
  - Errors use `{"error":"..."}`.
- Verify: `cd taskboard && go test -timeout 30s -count=1 ./internal/web/...`

### ta-8l7: taskboard/internal/web/api_test.go
- Scope: API tests via net/http/httptest.
- Architecture: architecture.md, Web API unit tests.
- Depends on: `taskboard/internal/web/api.go`.
- Consumed by: none.
- Acceptance:
  - Asserts status codes 200, 201, 204, 400, 404, 500, 503 for the endpoints that return them.
  - Asserts JSON error body shape and empty-list `[]` serialization.
  - Asserts server-generated fields (`created_at`, `updated_at`, default `status`) cannot be overridden on create.
- Verify: `cd taskboard && go test -timeout 30s -count=1 ./internal/web/...`

### ta-12r: taskboard/internal/web/app.go
- Scope: `App` struct, `NewApp`, router registration.
- Architecture: architecture.md, Module ownership.
- Interfaces: `App`, `NewApp(store, assets)`.
- Depends on: `taskboard/internal/store/store.go`, `taskboard/web/assets.go`.
- Consumed by: `taskboard/internal/web/api.go`, `taskboard/cmd/taskboard/main.go`.
- Acceptance:
  - `NewApp(store, assets)` stores the dependencies and builds one stdlib ServeMux.
  - Registers `/api/health` (200 `{"status":"ok"}`), `/api/ready` (200 `{"status":"ready"}` or 503 `{"error":"database unavailable"}`), and the 9 API routes from the SPEC table.
  - Registers `/` (HTML shell) and `/static/*` from the embedded FS.
  - Contains no SQL statements.
- Verify: `cd taskboard && go test -timeout 30s -count=1 ./internal/web/...`

### ta-qfs: taskboard/web/assets.go
- Scope: embedded assets FS.
- Architecture: architecture.md, Module ownership.
- Interfaces: embedded `FS`.
- Depends on: `taskboard/web/templates/index.html`.
- Consumed by: `taskboard/internal/web/app.go`, `taskboard/cmd/taskboard/main.go`.
- Acceptance:
  - `//go:embed templates/*` only in phase-1-app.
  - `static/*` embedding is deferred to phase-2-kanban.
  - Exports only the `FS` variable; no business logic.
- Verify: `cd taskboard && go test -timeout 30s -count=1 ./web/...`

### ta-dw6: taskboard/web/templates/index.html
- Scope: server-rendered HTML shell only.
- Architecture: architecture.md, Module ownership (client-side JS belongs to phase-2 `app.js`).
- Depends on: none.
- Consumed by: `taskboard/web/assets.go`.
- Acceptance:
  - Renders a "TaskBoard" heading and the To Do / Doing / Done Kanban column headers.
  - Includes a project creation form markup (input and submit) for the phase-2 script to wire up.
  - Contains no inline event handlers and no fetch or client JavaScript; those are phase-2 `app.js`.
- Verify: `cd taskboard && go test -timeout 30s -count=1 ./web/...`

## Future considerations
- Phase-2 wires client-side behavior and static assets; not in scope here.

## Phase context

Active delivery phase `phase-1-app` only. The orchestrator advances to the next phase automatically after QA `all_passed`.
