# Architecture for tasks

## Overview

TaskBoard is a small Go Kanban web application. Users create and delete projects, create, edit, move, filter, and delete tasks. Data persists in SQLite by default or in an existing PostgreSQL service selected at runtime. The design follows SPEC literally: Go 1.25 or newer, standard-library net/http ServeMux with method-aware patterns, encoding/json, html/template, embed, and database/sql. The frontend is server-rendered HTML with vanilla JavaScript and plain CSS, with no build step. There is no ORM, no migrations framework, no authentication, and no external API.

The sole Go module lives in taskboard/, which is the Gas Town layout_root. SPEC.md stays at the repository root. Every repository-relative path in this document carries the taskboard/ prefix. Application and test commands run from taskboard/ unless noted otherwise.

Major components:

- taskboard/internal/config/config.go parses TASKBOARD_DB_DRIVER, TASKBOARD_DB_PATH, TASKBOARD_DATABASE_URL, TASKBOARD_HOST, and TASKBOARD_PORT. It never prints the full PostgreSQL URL or any password.
- taskboard/internal/model/model.go holds pure domain types, enums, and validation (trimming, code-point length checks, the PATCH nonempty-subset rule). It imports no database or HTTP package.
- taskboard/internal/store/store.go defines the shared repository interface and sentinel errors. Dialect-specific code lives only in taskboard/internal/store/sqlite.go and taskboard/internal/store/postgres.go.
- taskboard/web/assets.go is the only go:embed location for templates and static files, because go:embed cannot traverse parent directories. It exports an fs.FS.
- taskboard/internal/web/app.go and taskboard/internal/web/api.go construct the app, register routes, encode and decode JSON, render the shell through html/template, and map errors to HTTP status codes.
- taskboard/cmd/taskboard/main.go handles startup and shutdown.
- The browser layer is taskboard/web/templates/index.html, taskboard/web/static/app.js, and taskboard/web/static/style.css. User text is inserted only with textContent or DOM APIs, never innerHTML.
- The deployment layer is taskboard/Dockerfile, taskboard/.dockerignore, and taskboard/deploy/k8s/. The image is a multi-stage CGO-free build. Kubernetes injects the PostgreSQL URL from a Secret.

Constraints carried through every phase:

- The app is built with the explicit constructor web.NewApp(store, assets), never a mutable global database handle.
- Schema creation is idempotent and safe under simultaneous PostgreSQL starts, using a transaction-scoped advisory lock.
- SQLite foreign keys and a 5000 ms busy timeout are set on every pooled connection through the DSN, not through a one-time PRAGMA.
- Timestamps are UTC RFC3339 with an explicit timezone. Tests never assert exact wall-clock values.
- Client input failures return 400, missing entities return 404, and storage failures return a generic 500 with no SQL text or credentials.
- Stores and handlers never convert database errors into empty lists.

## Planned file layout

Every path below is repository-relative with the taskboard/ prefix. Each file has exactly one first-delivery owner in the Delivery phases section.

- taskboard/go.mod: module with a Go 1.25 directive and requirements on modernc.org/sqlite and github.com/jackc/pgx/v5.
- taskboard/go.sum: produced by dependency resolution, never handwritten.
- taskboard/README.md: local SQLite run, external PostgreSQL run, tests, Playwright, Docker, k3s, and security notes.
- taskboard/.gitignore: excludes taskboard.db, *.db, binaries, node_modules, test-results, playwright-report, and real secret files.
- taskboard/.dockerignore: excludes database files, .git, node_modules, reports, build outputs, and real secret manifests.
- taskboard/cmd/taskboard/main.go: the process entrypoint.
- taskboard/internal/config/config.go: configuration loading and validation.
- taskboard/internal/model/model.go: domain types and pure validation.
- taskboard/internal/store/store.go: repository interface and sentinel errors.
- taskboard/internal/store/sqlite.go: SQLite implementation and DDL.
- taskboard/internal/store/postgres.go: PostgreSQL implementation, DDL, and advisory-locked schema initialization.
- taskboard/internal/store/store_test.go: shared store contract tests for SQLite (t.TempDir()) and PostgreSQL when configured.
- taskboard/internal/store/postgres_integration_test.go: PostgreSQL-only tests gated on TASKBOARD_TEST_POSTGRES_URL.
- taskboard/internal/web/app.go: App type, NewApp, route registration, root page, and static serving.
- taskboard/internal/web/api.go: JSON handlers for health, ready, projects, and tasks, plus error encoding.
- taskboard/internal/web/app_test.go: task CRUD, validation, filter, and error-path tests through httptest.
- taskboard/internal/web/api_test.go: API contract tests for health, ready, project round trips, validation, ordering, 204 bodies, 503 readiness, and database-failure safety.
- taskboard/internal/web/ui_content_test.go: GET / and static asset content assertions.
- taskboard/web/assets.go: the go:embed directive and the exported filesystem.
- taskboard/web/templates/index.html: the server-rendered HTML shell.
- taskboard/web/static/app.js: the vanilla JavaScript client.
- taskboard/web/static/style.css: plain CSS implementing the visual design system.
- Node manifest (declared in the phase-3-browser required_files entry): development dependency on @playwright/test, with no build scripts.
- Node lockfile (declared in the phase-3-browser required_files entry): generated by npm and committed.
- taskboard/playwright.config.ts: Playwright configuration with base URL http://127.0.0.1:8080.
- taskboard/playwright/taskboard.spec.ts: browser E2E specs.
- taskboard/Dockerfile: multi-stage container build.
- taskboard/deploy/k8s/deployment.yaml: single-replica Deployment.
- taskboard/deploy/k8s/service.yaml: ClusterIP Service.
- taskboard/deploy/k8s/secret.example.yaml: example Secret with placeholder values only.

No PostgreSQL Deployment, StatefulSet, chart, or PVC is planned for PostgreSQL mode. No database file, binary, or node_modules directory is committed.

## Module / bead ownership

| File | Owns (exported) | Must not define |
|---|---|---|
| taskboard/internal/model/model.go | Project, Task, TaskPatch types; ValidateProjectName, ValidateTaskTitle, ValidateDescription, ValidateStatus, ValidatePriority; TaskPatch.IsEmpty; ErrValidation | database or HTTP imports, SQL text |
| taskboard/internal/config/config.go | Config struct; Load(getenv func(string) string) (Config, error); Config.Redacted() string | listeners, database opens |
| taskboard/internal/store/store.go | Store interface (Ping, ListProjects, CreateProject, DeleteProject, ListTasks, CreateTask, UpdateTask, DeleteTask, Close); NewTask struct; ErrNotFound; ErrForeignKey | SQL text, driver imports |
| taskboard/internal/store/sqlite.go | OpenSQLite(ctx, path) (Store, error) | PostgreSQL DDL, pgx imports |
| taskboard/internal/store/postgres.go | OpenPostgres(ctx, url) (Store, error) | SQLite DDL, modernc imports |
| taskboard/web/assets.go | FS fs.FS rooted at the web directory | HTTP handlers |
| taskboard/internal/web/app.go | App struct; NewApp(store.Store, fs.FS) *App; route registration | SQL text, direct driver use |
| taskboard/internal/web/api.go | JSON handler methods on App; error encoder | SQL text, schema code |
| taskboard/cmd/taskboard/main.go | main() | business logic, SQL text |

Store methods take context.Context first and return sentinel errors so handlers map them uniformly to 404 and 500.

## HTTP + entrypoint integration

Routes match SPEC verbatim. Application-level 400 and 404 errors use exactly {"error":"human-readable message"}.

- GET /api/health returns 200 {"status":"ok"} and does not depend on the database.
- GET /api/ready returns 200 {"status":"ready"} when the database responds, otherwise 503 {"error":"database unavailable"}.
- GET /api/projects returns 200 with an ascending-ID array, or 500 on storage failure.
- POST /api/projects accepts {"name":"Demo"} and returns 201 project JSON, or 400.
- DELETE /api/projects/{project_id} returns 204 with an empty body, or 404.
- GET /api/projects/{project_id}/tasks returns 200 with an ascending-ID array, 404 for a missing project, or 400 for an invalid filter.
- POST /api/projects/{project_id}/tasks accepts {"title":"Build API","description":"Details","priority":"high"} and returns 201 task JSON, 404, or 400.
- PATCH /api/tasks/{task_id} accepts a nonempty subset of title, description, priority, and status, returns 200, 404, or 400.
- DELETE /api/tasks/{task_id} returns 204 with an empty body, or 404.

Malformed JSON, wrong types, and invalid IDs return 400. A malformed integer segment returns 400 when a route matches and 404 when none does. IDs must be positive. Unattributable database errors return a generic 500.

Project JSON: {"id":1,"name":"Demo","created_at":"2026-01-01T00:00:00Z"}.

Task JSON: {"id":1,"project_id":1,"title":"Build API","description":"Details","status":"todo","priority":"high","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}.

Task creation defaults description to "" and priority to medium, and always sets status to todo. Client-supplied IDs and timestamps are ignored. Explicit null PATCH fields are invalid. Project reassignment is forbidden. Empty lists serialize as [].

Runtime launch commands from SPEC, verbatim:

cd taskboard && go run ./cmd/taskboard

cd taskboard && TASKBOARD_DB_DRIVER=postgres TASKBOARD_DATABASE_URL='postgres://taskboard:example@postgres.default.svc.cluster.local:5432/taskboard?sslmode=require' TASKBOARD_HOST=0.0.0.0 go run ./cmd/taskboard

The sslmode must match the existing cluster. Production credentials come from Kubernetes Secrets only.

Wiring order: parse configuration, open the selected store (which initializes the schema), construct web.NewApp(store, web.FS), start an http.Server on the configured host and port, and on SIGTERM call Shutdown with a bounded timeout, then close the store. Static files are served under /static/ from the embedded filesystem with explicit content types. / renders index.html through html/template.

## Unit tests

Model tests cover trimming, code-point boundaries (80 for project names, 120 for titles, 1000 for descriptions), enum validation, and the nonempty PATCH rule. Config tests cover defaults, invalid driver, invalid port, empty PostgreSQL URL, and redaction. Store tests run the shared contract on SQLite in t.TempDir() and on PostgreSQL only when TASKBOARD_TEST_POSTGRES_URL is set. Web tests use httptest with a real SQLite store in a temporary directory.

## Integration and testing

The full Go suite runs with cd taskboard && go test ./... and does not require PostgreSQL. PostgreSQL integration tests run only when TASKBOARD_TEST_POSTGRES_URL is nonempty and must target a dedicated disposable database. Playwright runs against an already running server on 127.0.0.1:8080. Node dependencies are installed before test review, never during it.

## Docker & Deployment

The taskboard/Dockerfile is a two-stage build with these base images:

- Builder stage: `golang:1.25-alpine`, pinned to the Go 1.25 line. It sets CGO_ENABLED=0 and GOOS=linux, and builds the package at taskboard/cmd/taskboard into a static binary.
- Runtime stage: `alpine:3.20`, installing CA certificates and running as a non-root UID. It copies only the compiled binary to /app/taskboard, with no Go toolchain, Node runtime, or source tree.

The Dockerfile declares EXPOSE 8080, so the container listens on TCP port 8080 inside the image, and uses CMD ["/app/taskboard"] as its entrypoint. The container sets TASKBOARD_HOST=0.0.0.0. The build context is taskboard/. No image is pushed automatically.

The Deployment runs one replica with a non-root security context, requests and limits, a liveness probe on GET /api/health, a readiness probe on GET /api/ready, TASKBOARD_DB_DRIVER=postgres, and TASKBOARD_DATABASE_URL read from Secret key database-url. The Service is ClusterIP targeting port 8080. SQLite mode needs persistent storage and is not intended for multi-replica deployment.

## E2E / integration testing

taskboard/playwright/taskboard.spec.ts covers the initial shell, project and task creation through browser controls, Move forward from To Do to Doing to Done, delete, and the high-priority filter with reset. At least one test performs real clicks and selections and asserts DOM or API state changes.

## Acceptance mapping

Configuration and dialect rules map to taskboard/internal/config/config.go and the store files. The data model maps to taskboard/internal/model/model.go and the store implementations. The API contract maps to taskboard/internal/web/api.go and the API tests. The Kanban UI maps to the template, app.js, and style.css. Deployment maps to taskboard/Dockerfile and taskboard/deploy/k8s/.

## Delivery phases

This is the single canonical machine-readable delivery contract. Phase IDs match SPEC section 11 verbatim, and no other delivery table is authoritative. A generated profile that lists different phase IDs, extra files such as taskboard/docker-compose.yml, or docker compose QA commands is stale and must be regenerated from this block; this architecture does not adopt those entries.

```yaml
delivery_phases:
  - id: phase-1-core
    title: Go foundation, configuration, models, SQLite store
    spec_focus: Go foundation, configuration, models, SQLite store
    required_files:
      - taskboard/go.mod
      - taskboard/go.sum
      - taskboard/README.md
      - taskboard/.gitignore
      - taskboard/internal/config/config.go
      - taskboard/internal/model/model.go
      - taskboard/internal/store/store.go
      - taskboard/internal/store/sqlite.go
    qa_verify_command: 'cd taskboard && go mod verify && go test ./internal/config ./internal/model ./internal/store'
    depends_on: []
  - id: phase-1-app
    title: PostgreSQL store and runnable project API
    spec_focus: PostgreSQL store and runnable project API
    required_files:
      - taskboard/internal/store/postgres.go
      - taskboard/internal/store/store_test.go
      - taskboard/internal/store/postgres_integration_test.go
      - taskboard/cmd/taskboard/main.go
      - taskboard/internal/web/app.go
      - taskboard/internal/web/api.go
      - taskboard/web/assets.go
      - taskboard/web/templates/index.html
      - taskboard/internal/web/api_test.go
    qa_verify_command: 'cd taskboard && go build -o /tmp/taskboard-gate ./cmd/taskboard && go test ./... && (TASKBOARD_HOST=127.0.0.1 TASKBOARD_PORT=8080 /tmp/taskboard-gate >/tmp/taskboard-gate.log 2>&1 & PORT_PID=$!; ok=1; for i in $(seq 1 30); do curl -fsS http://127.0.0.1:8080/api/health >/dev/null 2>&1 && ok=0 && break; sleep 1; done; kill $PORT_PID 2>/dev/null; wait $PORT_PID 2>/dev/null; exit $ok)'
    depends_on: [phase-1-core]
    file_dependencies:
      taskboard/internal/store/postgres.go: [taskboard/internal/store/store.go]
      taskboard/cmd/taskboard/main.go: [taskboard/internal/config/config.go, taskboard/internal/store/sqlite.go]
  - id: phase-2-kanban
    title: Tasks, Kanban UI, and full Go tests
    spec_focus: Tasks, Kanban UI, and full Go tests
    required_files:
      - taskboard/web/static/app.js
      - taskboard/web/static/style.css
      - taskboard/internal/web/app_test.go
      - taskboard/internal/web/ui_content_test.go
    qa_verify_command: 'cd taskboard && go test ./...'
    depends_on: [phase-1-app]
    file_dependencies:
      taskboard/internal/web/ui_content_test.go: [taskboard/web/assets.go, taskboard/web/static/app.js]
  - id: phase-3-browser
    title: Playwright integration and browser QA
    spec_focus: Playwright integration and browser QA
    required_files:
      - taskboard/package.json
      - taskboard/package-lock.json
      - taskboard/playwright.config.ts
      - taskboard/playwright/taskboard.spec.ts
    qa_verify_command: 'cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright'
    depends_on: [phase-2-kanban]
  - id: phase-4-deploy
    title: Container build and k3s deployment artifacts
    spec_focus: Container build and k3s deployment artifacts
    required_files:
      - taskboard/Dockerfile
      - taskboard/.dockerignore
      - taskboard/deploy/k8s/deployment.yaml
      - taskboard/deploy/k8s/service.yaml
      - taskboard/deploy/k8s/secret.example.yaml
    qa_verify_command: 'cd taskboard && go test ./... && docker build -t taskboard:local .'
    depends_on: [phase-3-browser]
  - id: phase-5-release
    title: Final integration and release QA
    spec_focus: Final integration and release QA
    required_files: []
    qa_verify_command: 'cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright'
    depends_on: [phase-4-deploy]
```

## Requirements

### phase-1-core

Go foundation, configuration, models, and SQLite store. Complete when go mod verify and go test pass for config, model, and store, and packages without tests compile.

### phase-1-app

PostgreSQL store, runnable project API, server lifecycle, health and ready routes, embedded HTML shell, and project API tests. Complete when go test ./... passes and project create, list, and delete work over HTTP.

### phase-2-kanban

Task CRUD, validation, Kanban UI, filters, visible errors, empty states, and UI content tests. Complete when go test ./... passes and both static assets are nonempty.

### phase-3-browser

Playwright package, configuration, and E2E specs. The Node manifest and lockfile are both listed in required_files. Complete when the Playwright suite passes against the running server with real browser interactions.

### phase-4-deploy

Dockerfile, .dockerignore, and Kubernetes manifests. Complete when the image builds without CGO, runs as non-root, and serves health, ready, the UI, and static files.

### phase-5-release

Final integration and release QA across both backends, browser tests, container smoke test, and manifest review. No first-delivery files.
