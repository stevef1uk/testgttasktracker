# Implementation plan

Active phase: `phase-5-release`. This plan covers only the 30 `required_files` of this phase. Paths carry the `taskboard/` prefix. Symbols come from architecture.md's module ownership table.

## Integration contract

Entrypoint: `taskboard/cmd/taskboard/main.go`. Order: `config.Load` → `store.OpenSQLite` or `store.OpenPostgres` (schema init inside open) → `web.NewApp(st, web.FS)` → `http.Server` on `Config.Host:Config.Port` → on SIGTERM/SIGINT, `Shutdown` with a bounded timeout, then `st.Close()`.

Routes (SPEC verbatim, registered in taskboard/internal/web/app.go, handled in taskboard/internal/web/api.go): GET /api/health, GET /api/ready, GET and POST /api/projects, DELETE /api/projects/{project_id}, GET and POST /api/projects/{project_id}/tasks, PATCH /api/tasks/{task_id}, DELETE /api/tasks/{task_id}. Static files under /static/ from `web.FS`. GET / renders index.html via html/template.

Exports: config.go exports `Config`, `Load`, `Config.Redacted`. model.go exports `Project`, `Task`, `TaskPatch`, the Validate* functions, `TaskPatch.IsEmpty`, `ErrValidation`. store.go exports `Store`, `NewTask`, `ErrNotFound`, `ErrForeignKey`. sqlite.go exports `OpenSQLite(ctx, path)`. postgres.go exports `OpenPostgres(ctx, url)`. assets.go exports `FS`. app.go exports `App`, `NewApp`. api.go exports the JSON handler methods on `*App`.

## Bead map

### ta-hwt: taskboard/go.mod
- Scope: module with `go 1.25`, requiring modernc.org/sqlite and github.com/jackc/pgx/v5.
- Depends on: none. Consumed by: all Go files.
- Acceptance: both requirements present; `go mod tidy` succeeds.

### ta-bgl: taskboard/go.sum
- Scope: generated checksums, never handwritten.
- Depends on: taskboard/go.mod. Consumed by: taskboard/Dockerfile.
- Acceptance: `go mod verify` passes.

### ta-duy: taskboard/README.md
- Scope: SQLite and PostgreSQL run, tests, Playwright, Docker, k3s, security notes.
- Depends on: taskboard/cmd/taskboard/main.go. Consumed by: operators.
- Acceptance: contains both SPEC launch commands; states credentials come from Kubernetes Secrets only.

### ta-x5d: taskboard/.gitignore
- Scope: excludes taskboard.db, *.db, binaries, node_modules, test-results, playwright-report, real secrets.
- Depends on: none.
- Acceptance: every pattern present; deploy/k8s/secret.example.yaml stays committable.

### ta-1zs: taskboard/.dockerignore
- Scope: excludes database files, .git, node_modules, reports, build outputs, real secret manifests.
- Depends on: none. Consumed by: taskboard/Dockerfile.
- Acceptance: every exclusion present.

### ta-xlw: taskboard/internal/config/config.go
- Scope: parse the five TASKBOARD_* variables and validate them.
- Interfaces: `Config`, `Load(getenv func(string) string) (Config, error)`, `Config.Redacted() string`.
- Depends on: taskboard/go.mod. Consumed by: taskboard/cmd/taskboard/main.go.
- Acceptance: defaults apply; invalid driver, port, or empty URL errors; Redacted never shows the password.

### ta-ne6: taskboard/internal/model/model.go
- Scope: domain types and pure validation (trimming, 80/120/1000 code-point limits, nonempty PATCH).
- Interfaces: `Project`, `Task`, `TaskPatch`, Validate* functions, `TaskPatch.IsEmpty`, `ErrValidation`.
- Depends on: taskboard/go.mod. Consumed by: store and web packages.
- Acceptance: no database or HTTP imports; explicit null PATCH fields invalid.

### ta-29f: taskboard/internal/store/store.go
- Scope: repository interface and sentinel errors.
- Interfaces: `Store` (Ping, ListProjects, CreateProject, DeleteProject, ListTasks, CreateTask, UpdateTask, DeleteTask, Close), `NewTask`, `ErrNotFound`, `ErrForeignKey`.
- Depends on: taskboard/internal/model/model.go. Consumed by: sqlite.go, postgres.go, web package.
- Acceptance: every method takes context.Context first; no SQL or driver imports.

### ta-nlw: taskboard/internal/store/sqlite.go
- Scope: SQLite implementation and DDL; foreign keys and 5000 ms busy timeout in the DSN.
- Interfaces: `OpenSQLite(ctx context.Context, path string) (Store, error)`.
- Depends on: taskboard/internal/store/store.go. Consumed by: store and web tests, main.go.
- Acceptance: idempotent schema; missing rows return ErrNotFound; bad project references return ErrForeignKey.

### ta-5el: taskboard/internal/store/postgres.go
- Scope: PostgreSQL implementation, DDL, advisory-locked schema init.
- Interfaces: `OpenPostgres(ctx context.Context, url string) (Store, error)`.
- Depends on: taskboard/internal/store/store.go. Consumed by: postgres_integration_test.go, main.go.
- Acceptance: transaction-scoped advisory lock; errors never include the URL or password.

### ta-en7: taskboard/internal/store/store_test.go
- Scope: shared store contract tests; SQLite in t.TempDir(), PostgreSQL only when TASKBOARD_TEST_POSTGRES_URL is set.
- Depends on: store.go, sqlite.go, postgres.go.
- Acceptance: covers create, ascending-ID order, delete, defaults (medium, todo), PATCH subset, ErrNotFound, ErrForeignKey; no always-pass tests.

### ta-8pf: taskboard/internal/store/postgres_integration_test.go
- Scope: PostgreSQL-only tests gated on TASKBOARD_TEST_POSTGRES_URL.
- Depends on: postgres.go, store_test.go.
- Acceptance: skips when the variable is empty; targets a disposable database when set.

### ta-uu7: taskboard/internal/web/app.go
- Scope: App type, NewApp, method-aware route registration, root page, static serving with explicit content types.
- Interfaces: `App`, `NewApp(store.Store, fs.FS) *App`.
- Depends on: store.go, taskboard/web/assets.go. Consumed by: api.go, web tests, main.go.
- Acceptance: all nine SPEC routes registered; GET / renders index.html via html/template.

### ta-rod: taskboard/internal/web/api.go
- Scope: JSON handlers and the `{"error":"..."}` encoder; 400, 404, generic 500 mapping.
- Depends on: app.go, store.go, model.go. Consumed by: app.go, web tests.
- Acceptance: empty lists serialize as []; DELETE returns 204 with an empty body; 500 never leaks SQL text.

### ta-is1: taskboard/internal/web/api_test.go
- Scope: API contract tests via httptest against SQLite in a temp directory.
- Depends on: api.go, app.go, sqlite.go.
- Acceptance: health 200 without DB; ready 200 and 503 with `{"error":"database unavailable"}`; project round trips; 204 bodies.

### ta-cqs: taskboard/internal/web/app_test.go
- Scope: task CRUD, validation, filter, and error paths via httptest.
- Depends on: api.go, app.go, sqlite.go.
- Acceptance: default description "" and priority medium; status forced to todo; PATCH subset; 400 and 404 mapping.

### ta-omt: taskboard/internal/web/ui_content_test.go
- Scope: GET / and static asset content assertions.
- Depends on: app.go, assets.go, app.js.
- Acceptance: index shell present; app.js and style.css served with correct content types.

### ta-19o: taskboard/web/assets.go
- Scope: the only go:embed location; exports `FS fs.FS` rooted at the web directory.
- Depends on: templates/index.html, static/app.js, static/style.css. Consumed by: app.go, ui_content_test.go, main.go.
- Acceptance: embeds templates and static files; contains no HTTP handlers.

### ta-6aq: taskboard/web/templates/index.html
- Scope: server-rendered HTML shell.
- Depends on: static/style.css, static/app.js. Consumed by: assets.go, app.go.
- Acceptance: references assets under /static/; no unescaped server-side user text.

### ta-tba: taskboard/web/static/app.js
- Scope: vanilla JavaScript client; user text inserted only with textContent or DOM APIs.
- Depends on: the /api routes in api.go. Consumed by: index.html, assets.go.
- Acceptance: calls the nine API routes; never uses innerHTML; supports create, move, delete, and the high-priority filter.

### ta-ofs: taskboard/web/static/style.css
- Scope: plain CSS for the Kanban layout; no build step.
- Depends on: none. Consumed by: index.html.
- Acceptance: three-column Kanban layout.

### ta-xlo: taskboard/cmd/taskboard/main.go
- Scope: main() only; wiring order from the Integration contract; graceful SIGTERM shutdown.
- Depends on: config.go, sqlite.go, postgres.go, app.go, assets.go. Consumed by: README.md, Dockerfile.
- Acceptance: logs only Config.Redacted(); closes the store after shutdown; no SQL text.

### ta-2r6: taskboard/Dockerfile
- Scope: two-stage build; golang:1.25-alpine builder with CGO_ENABLED=0; alpine:3.20 non-root runtime; EXPOSE 8080; CMD ["/app/taskboard"].
- Depends on: go.mod, go.sum, main.go, .dockerignore. Consumed by: deployment.yaml.
- Acceptance: runtime image holds only the binary and CA certificates; sets TASKBOARD_HOST=0.0.0.0.

### ta-w3w: taskboard/deploy/k8s/deployment.yaml
- Scope: single-replica Deployment; non-root context; liveness on /api/health; readiness on /api/ready; TASKBOARD_DB_DRIVER=postgres; URL from Secret key database-url.
- Depends on: Dockerfile, secret.example.yaml. Consumed by: service.yaml.
- Acceptance: requests and limits set; no inline credentials.

### ta-bmz: taskboard/deploy/k8s/service.yaml
- Scope: ClusterIP Service targeting port 8080.
- Depends on: deployment.yaml. Consumed by: operators.
- Acceptance: type ClusterIP; port 8080.

### ta-31x: taskboard/deploy/k8s/secret.example.yaml
- Scope: example Secret with placeholder values only.
- Depends on: none. Consumed by: deployment.yaml.
- Acceptance: contains no real credentials; key database-url.

### ta-4r9: taskboard/package.json
- Scope: development dependency on @playwright/test; no build scripts.
- Depends on: none. Consumed by: package-lock.json, playwright.config.ts.
- Acceptance: no build scripts.

### ta-1jk: taskboard/package-lock.json
- Scope: lockfile generated by npm and committed.
- Depends on: package.json. Consumed by: Playwright runs.
- Acceptance: generated, not handwritten.

### ta-aug: taskboard/playwright.config.ts
- Scope: Playwright configuration with base URL http://127.0.0.1:8080.
- Depends on: package.json. Consumed by: taskboard.spec.ts.
- Acceptance: base URL matches the spec.

### ta-ud0: taskboard/playwright/taskboard.spec.ts
- Scope: browser E2E specs for the shell, project and task creation, Move forward, delete, and the high-priority filter with reset.
- Depends on: playwright.config.ts, app.js, index.html.
- Acceptance: real clicks and selections; asserts DOM or API state changes.
