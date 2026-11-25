# Architecture for TaskBoard

## Overview
TaskBoard is a task management web application (Kanban style) written in Go (Go 1.25+), implementing a clean architecture with standard library HTTP routing and dual-database support. Users create and delete projects, create and edit tasks, move tasks through three Kanban states (todo, doing, done), filter by status (all, todo, doing, done) and priority (all, low, medium, high), and delete tasks. Data persistence supports both SQLite (via modernc.org/sqlite) and PostgreSQL (via github.com/jackc/pgx/v5/stdlib), selectable dynamically at runtime via environment variables without changing application code.

Components:
- `taskboard/internal/config/config.go` parses configuration settings from environment variables (TASKBOARD_DB_DRIVER, TASKBOARD_DB_PATH, TASKBOARD_DATABASE_URL, TASKBOARD_HOST, TASKBOARD_PORT).
- `taskboard/internal/model/model.go` defines Project and Task structs and Unicode code-point validation using utf8.RuneCountInString.
- `taskboard/internal/store/store.go` defines the shared Store interface, repository errors, and SQL DDL statements.
- `taskboard/internal/store/sqlite.go` implements SQLite storage with foreign key enforcement on every connection (_pragma=foreign_keys(1)) and 5000ms busy timeout.
- `taskboard/internal/store/postgres.go` implements PostgreSQL storage with transactional advisory locks for schema setup.
- `taskboard/web/assets.go` embeds HTML templates in phase-1-app using a go:embed directive for the templates directory, then extends the directive to the static directory in phase-2-kanban once the static files exist.
- `taskboard/internal/web/app.go` and `taskboard/internal/web/api.go` build the HTTP router (net/http.ServeMux) and REST API endpoints.
- `taskboard/cmd/taskboard/main.go` serves as the CLI entrypoint for startup, configuration loading, database initialization, and graceful SIGTERM shutdown.
- `taskboard/deploy/k8s/deployment.yaml`, `taskboard/deploy/k8s/service.yaml`, and `taskboard/deploy/k8s/secret.example.yaml` provide Kubernetes manifests for deployment.
- `taskboard/playwright/taskboard.spec.ts` provides Playwright browser E2E test scenarios.

Runtime launch command from SPEC:
`cd taskboard && go run ./cmd/taskboard`

PostgreSQL runtime launch command from SPEC:
```bash
TASKBOARD_DB_DRIVER=postgres \
TASKBOARD_DATABASE_URL='postgres://taskboard:example@postgres.default.svc.cluster.local:5432/taskboard?sslmode=require' \
TASKBOARD_HOST=0.0.0.0 \
go run ./cmd/taskboard
```

## Planned file layout
- `taskboard/go.mod`
- `taskboard/go.sum`
- `taskboard/README.md`
- `taskboard/.gitignore`
- `taskboard/.dockerignore`
- `taskboard/Dockerfile`
- `taskboard/cmd/taskboard/main.go`
- `taskboard/internal/config/config.go`
- `taskboard/internal/model/model.go`
- `taskboard/internal/store/store.go`
- `taskboard/internal/store/sqlite.go`
- `taskboard/internal/store/postgres.go`
- `taskboard/internal/store/store_test.go`
- `taskboard/internal/store/postgres_integration_test.go`
- `taskboard/internal/web/app.go`
- `taskboard/internal/web/api.go`
- `taskboard/internal/web/app_test.go`
- `taskboard/internal/web/api_test.go`
- `taskboard/internal/web/ui_content_test.go`
- `taskboard/web/assets.go`
- `taskboard/web/templates/index.html`
- `taskboard/web/static/app.js`
- `taskboard/web/static/style.css`
- `taskboard/deploy/k8s/deployment.yaml`
- `taskboard/deploy/k8s/service.yaml`
- `taskboard/deploy/k8s/secret.example.yaml`
- `taskboard/package.json`
- `taskboard/package-lock.json`
- `taskboard/playwright.config.ts`
- `taskboard/playwright/taskboard.spec.ts`

## Module / bead ownership

| File | Owns (exported) | Must not define |
| --- | --- | --- |
| `taskboard/internal/config/config.go` | Config struct, LoadConfig function | Database queries, HTTP handlers |
| `taskboard/internal/model/model.go` | Project struct, Task struct, Status and Priority types, TaskFilter struct, ValidateProject, ValidateTask functions | SQL DDL statements, HTTP responses |
| `taskboard/internal/store/store.go` | Store interface, ErrNotFound and repository error constants, DDL schema constants | HTTP routing logic |
| `taskboard/internal/store/sqlite.go` | SQLiteStore struct, NewSQLiteStore function | Postgres SQL dialect syntax |
| `taskboard/internal/store/postgres.go` | PostgresStore struct, NewPostgresStore function | SQLite PRAGMA commands |
| `taskboard/internal/store/store_test.go` | SQLite integration and unit tests for Store using isolated t.TempDir() database files (never the runtime taskboard.db) | Postgres integration tests |
| `taskboard/internal/store/postgres_integration_test.go` | Postgres integration tests using container or DSN | SQLite-specific tests |
| `taskboard/web/assets.go` | Embedded FS assets (FS) | Handlers or DB connections |
| `taskboard/web/templates/index.html` | UI Kanban HTML template | Dynamic JS behavior |
| `taskboard/web/static/app.js` | Frontend JS dynamics, fetch calls, status updates, filters | Template structure |
| `taskboard/web/static/style.css` | UI styles and column responsive design | JS behavior |
| `taskboard/internal/web/app.go` | App struct, NewApp, Server lifecycle | Direct SQL statements |
| `taskboard/internal/web/api.go` | HTTP REST API handlers (projects, tasks, status, health, ready) | Direct DB drivers |
| `taskboard/internal/web/app_test.go` | Tests for HTTP routing and server initialization | Direct store integration tests |
| `taskboard/internal/web/api_test.go` | Unit/integration tests for REST API routes | Direct SQL queries |
| `taskboard/internal/web/ui_content_test.go` | Unit tests asserting embedded UI assets and templates render properly | REST API JSON tests |
| `taskboard/cmd/taskboard/main.go` | main function entrypoint | Handlers or store implementation |
| `taskboard/package.json` | E2E test dependencies and npm scripts | Server code |
| `taskboard/package-lock.json` | Locked npm dependency tree | Server code |
| `taskboard/playwright.config.ts` | Playwright test configuration | Unit test code |
| `taskboard/playwright/taskboard.spec.ts` | E2E browser automation tests for UI | Store implementations |
| `taskboard/Dockerfile` | Multi-stage Docker build recipe | Kubernetes runtime manifests |
| `taskboard/.dockerignore` | Build context exclusions for Docker | Deployment configuration |
| `taskboard/deploy/k8s/deployment.yaml` | Kubernetes Deployment configuration | Service endpoints |
| `taskboard/deploy/k8s/service.yaml` | Kubernetes Service configuration | Secret values |
| `taskboard/deploy/k8s/secret.example.yaml` | Kubernetes Secret template | Pod spec |

## HTTP + entrypoint integration

### HTTP Routes
- The home page at the root path renders the embedded template with the projects list and main board (SPEC §7).
- `GET /static/app.js` serves the embedded JavaScript asset; the `/static/` prefix file server (SPEC §7) serves the embedded stylesheet `taskboard/web/static/style.css` with a CSS content type.
- GET /api/health — Lightweight health check returning status ok JSON.
- GET /api/ready — Database readiness check returning status ready JSON when database ping succeeds or 503 error database unavailable.
- GET /api/projects — List all projects ordered by ID ascending.
- POST /api/projects — Create a project (JSON body: {"name": "..."}).
- DELETE /api/projects/{project_id} — Delete project and its tasks (returns 204 No Content).
- GET /api/projects/{project_id}/tasks — List tasks for a project ordered by ID ascending (query params: status, priority).
- POST /api/projects/{project_id}/tasks — Create a task for a project (JSON body: {"title": "...", "description": "...", "priority": "..."}).
- PATCH /api/tasks/{task_id} — Update task details (JSON body subset: title, description, priority, status).
- DELETE /api/tasks/{task_id} — Delete a task (returns 204 No Content).

### Entrypoint Wiring Pattern
1. Entrypoint `taskboard/cmd/taskboard/main.go` parses configuration settings via `taskboard/internal/config/config.go`.
2. Based on DB driver setting, creates SQLite store via `taskboard/internal/store/sqlite.go` or PostgreSQL store via `taskboard/internal/store/postgres.go`.
3. Calls schema initialization with transactional advisory locks for PostgreSQL setup.
4. Passes store instance and embedded assets from `taskboard/web/assets.go` to router constructor in `taskboard/internal/web/app.go`.
5. Registers static asset serving and REST API endpoints in `taskboard/internal/web/api.go`.
6. Starts HTTP server listening on configured host and port, supporting graceful SIGTERM shutdown.

## Unit tests
- `taskboard/internal/store/store_test.go`: unit tests for SQLite store initialization, task and project CRUD, foreign key enforcement, and filters.
- `taskboard/internal/store/postgres_integration_test.go`: integration tests against real PostgreSQL database when TASKBOARD_TEST_POSTGRES_URL is set.
- `taskboard/internal/web/api_test.go`: test REST API endpoints, validation, error JSON structures, and query filter combinations.
- `taskboard/internal/web/app_test.go`: test router setup, static asset routing, and server lifecycle.
- `taskboard/internal/web/ui_content_test.go`: assert index template and static assets render valid non-empty responses.

## Integration and testing
- Backend unit and integration test command:
  `cd taskboard && go test ./...`
- Frontend Playwright E2E test command:
  `cd taskboard && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`

## Docker & Deployment
- Docker multi-stage build recipe in `taskboard/Dockerfile`:
  - Stage 1 (Builder): golang:1.25-alpine - compiles binary `CGO_ENABLED=0 GOOS=linux go build -o /app/taskboard ./cmd/taskboard`.
  - Stage 2 (Final): alpine:3.20 - non-root user execution, copying compiled `/app/taskboard`.
  - Exposed Port: EXPOSE 8080
  - Runtime contract: TASKBOARD_HOST=0.0.0.0 in the container; CA certificates installed; non-root numeric UID; embedded HTML, JS and CSS; no Go toolchain or Node runtime in the final image; build context is taskboard/ with .dockerignore excluding databases, secrets, .git, node_modules, test reports and build outputs; supports linux/amd64 and linux/arm64 builds.
  - Command: `CMD ["/app/taskboard"]`
- Kubernetes Deployment (`taskboard/deploy/k8s/deployment.yaml`):
  - Target port 8080.
  - Liveness probe on GET /api/health.
  - Readiness probe on GET /api/ready.
  - Injects TASKBOARD_DB_DRIVER and TASKBOARD_DATABASE_URL from Secret database-url.
- Kubernetes Service (`taskboard/deploy/k8s/service.yaml`): ClusterIP exposing port 8080.
- Kubernetes Secret Example (`taskboard/deploy/k8s/secret.example.yaml`): Template for database-url Secret.

## E2E / integration testing
Playwright E2E tests in `taskboard/playwright/taskboard.spec.ts`:
- Loads the home UI page at the root path.
- Creates project and task via real UI interactions.
- Moves task through To Do -> Doing -> Done using Move forward control.
- Filters tasks by status (todo, doing, done) and priority (low, medium, high).
- Deletes tasks and projects with DOM state verification.

## Acceptance mapping
- Modular store interface abstraction with SQLite and PostgreSQL implementations matching SPEC.
- Standard Go net/http router with Go 1.25+ method-aware routing without third-party frameworks.
- Server-rendered HTML templates with embedded static files via go:embed.
- JSON REST API contract conforming strictly to SPEC error and response formatting.
- Multi-stage Docker build and Kubernetes manifests for k3s cluster integration.

## Delivery phases

```yaml
delivery_phases:
  - id: phase-1-core
    title: "Go foundation, configuration, models, SQLite store"
    spec_focus: "Phase 1a — Go foundation, configuration, models, SQLite store"
    required_files:
      - taskboard/go.mod
      - taskboard/go.sum
      - taskboard/README.md
      - taskboard/.gitignore
      - taskboard/internal/config/config.go
      - taskboard/internal/model/model.go
      - taskboard/internal/store/store.go
      - taskboard/internal/store/sqlite.go
    qa_verify_command: "cd taskboard && go mod verify && go test ./internal/config ./internal/model ./internal/store"
    depends_on: []

  - id: phase-1-app
    title: "PostgreSQL store and runnable project API"
    spec_focus: "Phase 1b — PostgreSQL store and runnable project API"
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
    qa_verify_command: "cd taskboard && go test ./..."
    depends_on:
      - phase-1-core
    file_dependencies:
      taskboard/internal/store/postgres.go:
        - taskboard/internal/store/store.go
      taskboard/internal/store/store_test.go:
        - taskboard/internal/store/sqlite.go
      taskboard/internal/store/postgres_integration_test.go:
        - taskboard/internal/store/postgres.go
      taskboard/web/assets.go:
        - taskboard/web/templates/index.html
      taskboard/internal/web/app.go:
        - taskboard/internal/config/config.go
        - taskboard/internal/store/store.go
        - taskboard/web/assets.go
      taskboard/internal/web/api.go:
        - taskboard/internal/web/app.go
      taskboard/cmd/taskboard/main.go:
        - taskboard/internal/config/config.go
        - taskboard/internal/store/sqlite.go
        - taskboard/internal/store/postgres.go
        - taskboard/internal/web/app.go
      taskboard/internal/web/api_test.go:
        - taskboard/internal/web/api.go

  - id: phase-2-kanban
    title: "Tasks, Kanban UI, and full Go tests"
    spec_focus: "Phase 2 — Tasks, Kanban UI, and full Go tests"
    required_files:
      - taskboard/web/static/app.js
      - taskboard/web/static/style.css
      - taskboard/internal/web/app_test.go
      - taskboard/internal/web/ui_content_test.go
    qa_verify_command: "cd taskboard && go test ./..."
    depends_on:
      - phase-1-app
    file_dependencies:
      taskboard/internal/web/app_test.go:
        - taskboard/web/static/app.js
        - taskboard/web/static/style.css
        - taskboard/internal/web/app.go
      taskboard/internal/web/ui_content_test.go:
        - taskboard/web/static/app.js
        - taskboard/web/static/style.css
        - taskboard/web/assets.go

  - id: phase-3-browser
    title: "Playwright integration and browser QA"
    spec_focus: "Phase 3 — Playwright integration and browser QA"
    required_files:
      - taskboard/package.json
      - taskboard/package-lock.json
      - taskboard/playwright.config.ts
      - taskboard/playwright/taskboard.spec.ts
    qa_verify_command: "cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright"
    depends_on:
      - phase-2-kanban
    file_dependencies:
      taskboard/playwright/taskboard.spec.ts:
        - taskboard/playwright.config.ts
        - taskboard/package.json

  - id: phase-4-deploy
    title: "Container build and k3s deployment artifacts"
    spec_focus: "Phase 4 — Container build and k3s deployment artifacts"
    required_files:
      - taskboard/Dockerfile
      - taskboard/.dockerignore
      - taskboard/deploy/k8s/deployment.yaml
      - taskboard/deploy/k8s/service.yaml
      - taskboard/deploy/k8s/secret.example.yaml
    qa_verify_command: "cd taskboard && go test ./... && docker build -t taskboard:local ."
    depends_on:
      - phase-3-browser
    file_dependencies:
      taskboard/Dockerfile:
        - taskboard/go.mod
        - taskboard/go.sum
        - taskboard/cmd/taskboard/main.go
        - taskboard/internal/web/app.go
        - taskboard/internal/web/api.go
        - taskboard/web/assets.go
        - taskboard/web/templates/index.html
        - taskboard/web/static/app.js
        - taskboard/web/static/style.css

  - id: phase-5-release
    title: "Final integration and release QA"
    spec_focus: "Phase 5 — Final integration and release QA"
    required_files: []
    qa_verify_command: "cd taskboard && go test ./..."
    depends_on:
      - phase-4-deploy
```

## Requirements

### phase-1-core
- Parse environment variables into Config struct in `taskboard/internal/config/config.go`.
- Define Project, Task, and enum validation in `taskboard/internal/model/model.go`.
- Define Store interface in `taskboard/internal/store/store.go`.
- Implement SQLite driver with PRAGMAs in `taskboard/internal/store/sqlite.go`.

### phase-1-app
- Implement PostgreSQL driver with advisory locking in `taskboard/internal/store/postgres.go`.
- Implement web router and HTTP API in `taskboard/internal/web/app.go` and `taskboard/internal/web/api.go`.
- Embed HTML templates in `taskboard/web/assets.go` and skeleton HTML in `taskboard/web/templates/index.html`.
- Phase-1 static boundary: `taskboard/web/assets.go` uses only the directive `//go:embed templates/*`; `taskboard/web/templates/index.html` is a skeleton shell with no `/static/` script or stylesheet references; `taskboard/internal/web/app.go` registers no static file route in phase-1-app. Static files `taskboard/web/static/app.js` and `taskboard/web/static/style.css` remain phase-2-kanban owned and are not prerequisites of phase-1-app.
- Wire process startup and shutdown in `taskboard/cmd/taskboard/main.go`.
- Add backend store and API unit tests in `taskboard/internal/store/store_test.go`, `taskboard/internal/store/postgres_integration_test.go`, and `taskboard/internal/web/api_test.go`.

### phase-2-kanban
- Build interactive frontend client logic in `taskboard/web/static/app.js` and CSS styling in `taskboard/web/static/style.css`.
- Add web server unit tests in `taskboard/internal/web/app_test.go` and asset content tests in `taskboard/internal/web/ui_content_test.go`.
- Extend `taskboard/web/templates/index.html` in place (first owned by phase-1-app) to reference `/static/style.css` and `/static/app.js`; extend `taskboard/web/assets.go` to the directive `//go:embed templates/* static/*`; enable the `/static/` file server in `taskboard/internal/web/app.go`. These phase-2-kanban changes satisfy the static-file dependency that phase-1-app no longer carries.

### phase-3-browser
- Configure Playwright in `taskboard/playwright.config.ts` and `taskboard/package.json`.
- Implement end-to-end browser tests in `taskboard/playwright/taskboard.spec.ts`.

### phase-4-deploy
- Write multi-stage Docker build recipe in `taskboard/Dockerfile` and exclusions in `taskboard/.dockerignore`.
- Create Kubernetes deployment manifests in `taskboard/deploy/k8s/deployment.yaml`, `taskboard/deploy/k8s/service.yaml`, and `taskboard/deploy/k8s/secret.example.yaml`.

### phase-5-release
- Execute final cross-dialect, E2E browser, and release QA checks.
