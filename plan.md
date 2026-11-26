# Implementation plan — phase-1-app (PostgreSQL store and runnable project API)

Active phase: `phase-1-app`. Depends on `phase-1-core`: go.mod, config, model, store interface, and SQLite store are already delivered.
Scope: exactly the 9 `required_files` below. Later phases own their own beads.

## Phase verification gate (applies to every bead in this phase)

Every bead's Verify line is the phase gate, run from the rig's `taskboard/` directory:

```
cd taskboard && go build -o /tmp/taskboard-gate ./cmd/taskboard && go test ./... && (TASKBOARD_HOST=127.0.0.1 TASKBOARD_PORT=8080 /tmp/taskboard-gate >/tmp/taskboard-gate.log 2>&1 & PORT_PID=$!; ok=1; for i in $(seq 1 30); do curl -fsS http://127.0.0.1:8080/api/health >/dev/null 2>&1 && ok=0 && break; sleep 1; done; kill $PORT_PID 2>/dev/null; wait $PORT_PID 2>/dev/null; exit $ok)
```

A bead is closed only when this full command exits 0. The `/api/health` probe must succeed, not just compile. If the polecat-scope gate rejects the full command, report it as a verification blocker; do not narrow the gate.

## Bead map

### ta-5el: taskboard/internal/store/postgres.go
- Scope: PostgreSQL implementation of `store.Store` using pgx/v5 stdlib and database/sql.
- Architecture: `OpenPostgres(ctx, url) (Store, error)`; no SQLite DDL and no modernc imports.
- Interfaces: `OpenPostgres(ctx context.Context, url string) (Store, error)`.
- Depends on: taskboard/internal/store/store.go.
- Consumed by: taskboard/cmd/taskboard/main.go, taskboard/internal/store/store_test.go, taskboard/internal/store/postgres_integration_test.go.
- Acceptance:
  - Schema creation is idempotent and runs under a transaction-scoped advisory lock.
  - Identity primary keys; `$1` placeholders only; no interpolation of user input.
  - Projects and tasks list in ascending id; timestamps are UTC RFC3339.
  - Deleting a project cascades to its tasks; `tasks.project_id` is indexed.
- Verify: full phase gate above.

### ta-en7: taskboard/internal/store/store_test.go
- Scope: shared store contract tests against SQLite (`t.TempDir()`) and PostgreSQL when configured.
- Architecture: SPEC section 8 store tests 1–8.
- Depends on: taskboard/internal/store/store.go, taskboard/internal/store/sqlite.go, taskboard/internal/store/postgres.go.
- Acceptance:
  - Fresh schema is empty; repeated initialization is idempotent.
  - Projects list ascending; tasks default to `todo` and `medium` with UTC timestamps.
  - Filters by status, priority, and both (AND).
  - Patch preserves omitted fields; deleting a project removes its tasks.
- Verify: full phase gate above.

### ta-8pf: taskboard/internal/store/postgres_integration_test.go
- Scope: PostgreSQL-only tests, skipped unless `TASKBOARD_TEST_POSTGRES_URL` is nonempty.
- Architecture: dedicated disposable database; uniquely named objects; cleans up only its own data.
- Depends on: taskboard/internal/store/postgres.go.
- Acceptance:
  - Runs the same contract as SQLite: cascade, ordering, filters, timestamps, schema idempotency.
  - Never touches a shared or production schema.
- Verify: full phase gate above.

### ta-xlo: taskboard/cmd/taskboard/main.go
- Scope: process entrypoint: parse config, open store, build app, serve, shut down gracefully.
- Architecture: wiring order in the Integration contract below.
- Depends on: taskboard/internal/config/config.go, taskboard/internal/store/sqlite.go, taskboard/internal/store/postgres.go, taskboard/internal/web/app.go, taskboard/web/assets.go.
- Acceptance:
  - Invalid configuration exits nonzero without printing the PostgreSQL URL or password.
  - Selects SQLite or PostgreSQL from config; never falls back silently.
  - Listens on `TASKBOARD_HOST:TASKBOARD_PORT`; `GET /api/health` returns 200 while the process runs.
  - On SIGTERM calls `Shutdown` with a bounded timeout, then closes the store.
- Verify: full phase gate above (the health probe is required, not optional).

### ta-uu7: taskboard/internal/web/app.go
- Scope: `App` type, `NewApp`, route registration, root page rendering, static serving.
- Architecture: `App` is built only via `NewApp(store.Store, fs.FS)`; no global database handle.
- Depends on: taskboard/internal/store/store.go, taskboard/web/assets.go, taskboard/web/templates/index.html.
- Acceptance:
  - Uses net/http ServeMux method-aware patterns.
  - `GET /` renders index.html through html/template.
  - Contains no SQL text and no direct driver use.
- Verify: full phase gate above.

### ta-rod: taskboard/internal/web/api.go
- Scope: JSON handlers for health, ready, and project routes; generic error encoder.
- Architecture: SPEC section 6 health, ready, and project routes.
- Depends on: taskboard/internal/web/app.go, taskboard/internal/store/store.go.
- Acceptance:
  - `GET /api/health` returns 200 `{"status":"ok"}`, independent of the database.
  - `GET /api/ready` returns 200 `{"status":"ready"}` or 503 `{"error":"database unavailable"}`.
  - `GET /api/projects` returns ascending ids, `[]` when empty; generic 500 on storage failure.
  - `POST /api/projects` returns 201; blank or overlong name returns 400.
  - `DELETE /api/projects/{project_id}` returns 204 with empty body; missing returns 404.
- Verify: full phase gate above.

### ta-is1: taskboard/internal/web/api_test.go
- Scope: httptest contract tests for health, ready, and project round trips.
- Architecture: SPEC section 8 API tests 1, 2, 4 (project subset), 5, 6, 9.
- Depends on: taskboard/internal/web/app.go, taskboard/internal/web/api.go, taskboard/internal/store/sqlite.go.
- Acceptance:
  - Exact liveness and ready JSON against a healthy SQLite store.
  - Project create, list, and delete assert stored state, not status codes alone.
  - Blank name rejected with 400; empty list serializes as `[]`; 204 bodies are empty.
  - Database failure yields generic 500 and readiness 503 without internal error text.
- Verify: full phase gate above.

### ta-1gk: taskboard/web/assets.go
- Scope: the only `go:embed` location for templates; exports the filesystem.
- Architecture: `go:embed` cannot traverse parent directories.
- Interfaces: `var FS fs.FS`; phase-1-app embeds `templates/*` only. `static/*` is added in phase-2-kanban.
- Depends on: taskboard/web/templates/index.html.
- Acceptance: `FS` opens `templates/index.html` with nonempty content.
- Verify: full phase gate above.

### ta-1vd: taskboard/web/templates/index.html
- Scope: real server-rendered HTML shell parsed by html/template.
- Architecture: heading `TaskBoard`, project input with `Add project` button, project selector fed from `/api/projects`.
- Depends on: none.
- Acceptance:
  - Contains the text `TaskBoard` and the project input with `Add project` button.
  - No innerHTML interpolation of user data.
- Verify: full phase gate above.

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

Wire order: config.Load → OpenSQLite or OpenPostgres → web.NewApp(store, web.FS) → http.Server on TASKBOARD_HOST:TASKBOARD_PORT → on SIGTERM Shutdown, then store.Close().

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
- Static asset embedding (`static/*`) is introduced in phase-2-kanban.
