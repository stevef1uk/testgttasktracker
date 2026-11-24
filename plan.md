# Implementation plan — phase-1-app (rework)

Scope: PostgreSQL store and runnable project API. Only the nine `required_files` below are in scope for this phase. Paths under `taskboard/` are exact. Bead IDs come from `bd list --limit=0 ` in this session.

## Bead map

### ta-11e: taskboard/internal/store/postgres.go
- Scope: PostgreSQL implementation of the Store interface in store.go using database/sql with the pgx/v5 stdlib driver.
- Architecture: taskboard/internal/store/postgres.go owns NewPostgresStore; schema setup runs under a transactional advisory lock.
- Interfaces: NewPostgresStore(dsn string) (Store, error) (from architecture).
- Depends on: taskboard/internal/store/store.go (Store interface, ErrNotFound, DDL constants).
- Consumed by: taskboard/cmd/taskboard/main.go, taskboard/internal/store/postgres_integration_test.go.
- Acceptance: all Store methods return ErrNotFound for missing rows; ListProjects and ListTasks return non-nil slices; no SQLite PRAGMA syntax.
- Verify: cd taskboard && go test ./internal/store/...

### ta-5qb: taskboard/internal/store/postgres_integration_test.go
- Scope: integration tests for the PostgreSQL store, skipped unless TASKBOARD_DATABASE_URL is set.
- Architecture: standard testing package only.
- Depends on: taskboard/internal/store/postgres.go.
- Acceptance: schema setup is idempotent under concurrent start; project and task CRUD round-trip; cascade delete of tasks when a project is deleted.
- Verify: cd taskboard && go test ./internal/store/...

### ta-nw9: taskboard/internal/store/store_test.go
- Scope: shared Store contract tests run against the SQLite store.
- Depends on: taskboard/internal/store/sqlite.go, taskboard/internal/store/store.go.
- Acceptance: TestStoreContractMissingRowsReturnErrNotFound covers GetProject, GetTask, UpdateTask, DeleteProject, DeleteTask; TestStoreContractListReturnsNonNilSlices covers empty projects and empty task lists.
- Verify: cd taskboard && go test ./internal/store/...

### ta-83w: taskboard/internal/web/api.go
- Scope: REST handlers for the SPEC API routes listed in the Integration contract below.
- Interfaces: route handler methods on *App; JSON via writeJSON defined in app.go.
- Depends on: taskboard/internal/web/app.go, taskboard/internal/store/store.go.
- Consumed by: taskboard/internal/web/api_test.go, taskboard/cmd/taskboard/main.go.
- Acceptance: POST /api/projects and task create apply default status todo and priority medium; PATCH accepts a subset of fields and rejects null or empty title; list filters on status and priority; unknown IDs return 404; storage failures return a generic 500 without leaking driver text.
- Verify: cd taskboard && go test ./internal/web/...

### ta-8l7: taskboard/internal/web/api_test.go
- Scope: httptest coverage of the task REST surface through the App router. Rework is required because the review found gaps.
- Architecture: REST routes table in architecture.md.
- Depends on: taskboard/internal/web/api.go, taskboard/internal/web/app.go, taskboard/internal/store/store.go.
- Acceptance (from review list):
  - create task defaults: status todo, priority medium.
  - PATCH subset updates; PATCH with null or empty title is rejected.
  - list filters by status and priority; ordering is ascending id.
  - 404 for unknown project or task IDs.
  - title length limit 120 and description limit 1000 (Unicode characters) enforced.
  - generic 500 when the store returns a storage error.
- Note: taskboard/internal/web/app_test.go already contains project and task REST coverage per the review; re-verify it against this list rather than duplicating it here.
- Verify: cd taskboard && go test -count=1 ./internal/web/...

### ta-12r: taskboard/internal/web/app.go
- Scope: App struct, NewApp(s store.Store), Handler, the page route for the index template, the static asset file server, and the writeJSON helper.
- Interfaces: NewApp(s store.Store) *App; (*App).Handler() http.Handler.
- Depends on: taskboard/web/assets.go, taskboard/internal/store/store.go.
- Consumed by: taskboard/internal/web/api.go, taskboard/cmd/taskboard/main.go.
- Acceptance: the index page is served from the embedded template; static assets such as /static/app.js are served from the embed FS; unknown methods are rejected by ServeMux method patterns.
- Verify: cd taskboard && go test ./internal/web/...

### ta-qfs: taskboard/web/assets.go
- Scope: embed FS for templates and static assets.
- Interfaces: var FS embed.FS with //go:embed templates/* static/*.
- Depends on: taskboard/web/templates/index.html.
- Consumed by: taskboard/internal/web/app.go.
- Acceptance: the embed directive lists only templates/ and static/; no static file is duplicated elsewhere.
- Verify: cd taskboard && go build ./...

### ta-dw6: taskboard/web/templates/index.html
- Scope: single-page layout with project panel, task panel, task form, and status and priority filter selects.
- Architecture: server-rendered HTML with no client-side templating.
- Acceptance: form controls use ids task-form, task-title, task-status, task-priority, filter-status, filter-priority; status options todo, doing, done; priority options low, medium, high; filter selects include an "all" option with an empty value.
- Verify: cd taskboard && go test ./internal/web/...

### ta-5kn: taskboard/cmd/taskboard/main.go
- Scope: entrypoint that loads config, opens the store by TASKBOARD_DB_DRIVER, builds the app, and serves with graceful SIGTERM shutdown.
- Depends on: taskboard/internal/store/postgres.go, taskboard/internal/store/store.go, taskboard/internal/web/app.go.
- Acceptance: the process starts with the SQLite driver by default; SIGTERM triggers http.Server.Shutdown.
- Verify: cd taskboard && go build ./cmd/taskboard

## Future considerations
- taskboard/web/static/app.js and taskboard/web/static/style.css belong to phase-2-kanban. The go:embed pattern static/* in taskboard/web/assets.go needs matching files at compile time; if phase-1-app builds fail on that pattern, report architecture_failure rather than adding those files here.
- taskboard/internal/web/ui_content_test.go and the ta-s7n exact option-set assertions are outside phase-1-app required_files and belong to the later phase that owns them.

## Integration contract

Wiring order: config.LoadConfig() → store.NewSQLiteStore or store.NewPostgresStore by TASKBOARD_DB_DRIVER → web.NewApp(dbStore) → http.Server with graceful SIGTERM shutdown. Dependency packages are planned before the entrypoint (ta-11e, ta-nw9, ta-83w, ta-12r, then ta-5kn).

SPEC HTTP API routes, registered by web.NewApp through the method-aware ServeMux (owner ta-83w, taskboard/internal/web/api.go):

- `GET /api/health`
- `GET /api/ready`
- `GET /api/projects`
- `POST /api/projects`
- `DELETE /api/projects/{project_id}`
- `GET /api/projects/{project_id}/tasks`
- `POST /api/projects/{project_id}/tasks`
- `PATCH /api/tasks/{task_id}`
- `DELETE /api/tasks/{task_id}`

Page and asset routes (owner ta-12r, taskboard/internal/web/app.go): the index page at `GET /`, and static assets under `/static/` such as `/static/app.js`, served from the embed FS in taskboard/web/assets.go (ta-qfs).

Exported symbols per file:
- taskboard/internal/store/postgres.go (ta-11e): NewPostgresStore.
- taskboard/internal/web/app.go (ta-12r): App, NewApp, Handler, writeJSON.
- taskboard/internal/web/api.go (ta-83w): route handler methods on *App.
- taskboard/web/assets.go (ta-qfs): FS.
- taskboard/cmd/taskboard/main.go (ta-5kn): main.
