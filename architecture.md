# Architecture for TaskBoard

## Overview
TaskBoard is a task management web application written in Go (Go 1.25+). Users create and delete projects, create and edit tasks, move tasks through Kanban states, filter by status and priority, and delete tasks. Data persistence supports SQLite (via modernc.org/sqlite) and PostgreSQL (via github.com/jackc/pgx/v5/stdlib), selectable dynamically at runtime via environment variables without changing application code.

Components:
- `taskboard/internal/config/config.go` parses configuration settings from environment variables.
- `taskboard/internal/model/model.go` defines Project and Task structs and Unicode length and enum validation.
- `taskboard/internal/store/store.go` defines the shared Store interface, repository errors, and SQL schema DDL statements.
- `taskboard/internal/store/sqlite.go` implements SQLite storage with foreign key PRAGMA enforcement and 5000ms busy timeout.
- `taskboard/internal/store/postgres.go` implements PostgreSQL storage with transactional advisory locks for schema setup.
- `taskboard/web/assets.go` embeds HTML templates and static assets.
- `taskboard/internal/web/app.go` and `taskboard/internal/web/api.go` build the HTTP router and REST API endpoints.
- `taskboard/cmd/taskboard/main.go` serves as the entrypoint for startup, configuration loading, database initialization, and graceful SIGTERM shutdown.
- `taskboard/deploy/k8s/deployment.yaml`, `taskboard/deploy/k8s/service.yaml`, and `taskboard/deploy/k8s/secret.example.yaml` provide Kubernetes manifests for deployment.
- `taskboard/playwright/taskboard.spec.ts` provides Playwright browser E2E test scenarios.

Runtime launch command from SPEC:
`cd taskboard && go run ./cmd/taskboard`

PostgreSQL runtime launch command from SPEC:
`TASKBOARD_DB_DRIVER=postgres TASKBOARD_DATABASE_URL='postgres://taskboard:example@postgres.default.svc.cluster.local:5432/taskboard?sslmode=require' TASKBOARD_HOST=0.0.0.0 go run ./cmd/taskboard`

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
| `taskboard/internal/model/model.go` | Project struct, Task struct, validation functions | SQL DDL statements, HTTP responses |
| `taskboard/internal/store/store.go` | Store interface, repository error constants, DDL constants | HTTP routing logic |
| `taskboard/internal/store/sqlite.go` | NewSQLiteStore function | Postgres SQL dialect syntax |
| `taskboard/internal/store/postgres.go` | NewPostgresStore function | SQLite PRAGMA commands |
| `taskboard/web/assets.go` | FS embed variable | Business logic |
| `taskboard/internal/web/app.go` | App struct, NewApp function | SQL statements |
| `taskboard/internal/web/api.go` | API route handler methods on App struct | Template rendering |
| `taskboard/cmd/taskboard/main.go` | main entrypoint function | Core package helper routines |
| `taskboard/web/templates/index.html` | HTML layout template | Client-side JavaScript |
| `taskboard/web/static/app.js` | Browser JavaScript UI state handling | Server-side template compilation |
| `taskboard/web/static/style.css` | Plain CSS layout styles | JavaScript DOM manipulators |
| `taskboard/deploy/k8s/deployment.yaml` | Kubernetes Deployment specification | Application code |
| `taskboard/deploy/k8s/service.yaml` | Kubernetes Service specification | Secrets |
| `taskboard/deploy/k8s/secret.example.yaml` | Kubernetes Secret example template | Real production passwords |
| `taskboard/playwright.config.ts` | Playwright test runner configuration | Backend Go unit tests |
| `taskboard/playwright/taskboard.spec.ts` | Playwright E2E browser tests | Go unit test assertions |

## HTTP + entrypoint integration

### Standardized HTTP API Table

| HTTP Method | Route / Endpoint | Description |
|---|---|---|
| `GET` | `/api/health` | Liveness check (returns JSON `{"status":"ok"}`) |
| `GET` | `/api/ready` | Readiness check (verifies database ping/connection) |
| `GET` | `/api/projects` | List all projects |
| `POST` | `/api/projects` | Create a new project (JSON body: `{"name":"..."}`) |
| `DELETE` | `/api/projects/{project_id}` | Delete project and all associated tasks |
| `GET` | `/api/projects/{project_id}/tasks` | List tasks for a project (optional query params: `status`, `priority`) |
| `POST` | `/api/projects/{project_id}/tasks` | Create task under project (JSON: `{"title":"...","description":"...","priority":"..."}`) |
| `PATCH` | `/api/tasks/{task_id}` | Update task title, description, status, or priority |
| `DELETE` | `/api/tasks/{task_id}` | Delete a task |
| `GET` | `/` | Serve server-rendered HTML page (`index.html` via `html/template`) |
| `GET` | `/static/*` | Serve embedded static assets (`app.js`, `style.css`) |

### Server Wiring Pattern
1. Entrypoint `taskboard/cmd/taskboard/main.go` parses configuration settings from environment variables using `taskboard/internal/config/config.go`.
2. Based on TASKBOARD_DB_DRIVER, instantiates SQLite or PostgreSQL store and runs schema setup.
3. Instantiates HTTP app using `taskboard/internal/web/app.go` and `taskboard/web/assets.go`.
4. Registers REST routes and static file handler on standard library `net/http` router.
5. Starts HTTP server listening on configured host and port.
6. Listens for SIGINT/SIGTERM to trigger graceful server shutdown.

## Docker & Deployment

### Application Dockerfile (`taskboard/Dockerfile`)
1. Multi-stage build process:
   - Builder stage: Pinned Go builder image (`golang:1.25-alpine`). Compiles `./cmd/taskboard` with `CGO_ENABLED=0` and `GOOS=linux`, supporting target architectures (`linux/amd64`, `linux/arm64`).
   - Final runtime stage: Base image (`alpine:3.20`) with CA certificates installed (`ca-certificates`). Copies compiled static binary, sets non-root user UID 10001, and embeds runtime assets.
2. Exposed port: `EXPOSE 8080`.
3. Sets environment variable `TASKBOARD_HOST=0.0.0.0`.
4. Entrypoint directive: `ENTRYPOINT ["/app/taskboard"]` or `CMD ["/app/taskboard"]`.

### Kubernetes Setup
- `taskboard/deploy/k8s/deployment.yaml` defines single replica deployment referencing TASKBOARD_DB_DRIVER and TASKBOARD_DATABASE_URL from Secret database-url. Liveness probe targets `GET /api/health` and readiness probe targets `GET /api/ready`.
- `taskboard/deploy/k8s/service.yaml` exposes port 8080 as ClusterIP Service.
- `taskboard/deploy/k8s/secret.example.yaml` provides example Kubernetes secret template.

## Unit tests
- `taskboard/internal/store/store_test.go`: SQLite table initialization, CRUD for projects and tasks, cascading deletions, Unicode length checks, and filtering.
- `taskboard/internal/web/api_test.go`: HTTP REST endpoint behavior, status codes (200, 201, 204, 400, 404, 500, 503), error JSON responses, filtering combinations.
- `taskboard/internal/web/app_test.go`: HTML template rendering on `GET /`, content types for static assets.
- `taskboard/internal/web/ui_content_test.go`: Specific HTML structure checks for forms, headings, and Kanban columns.

## Integration and testing
### Strategy and Unit Test Structure
The testing strategy separates fast, isolated backend Go tests from browser-based E2E scenarios:
- Unit tests in `taskboard/internal/store/store_test.go` execute against isolated SQLite in-memory/temporary file instances using t.TempDir(), validating CRUD operations, foreign key cascades, Unicode code-point lengths (utf8.RuneCountInString), enum validation, and combined query filtering.
- Web API unit tests in `taskboard/internal/web/api_test.go` use net/http/httptest to test router status codes, JSON payload formatting, non-overridable server-generated fields (created_at, updated_at, status on create), error structures ({"error": "message"}), and status code invariants (200, 201, 204, 400, 404, 500, 503).
- UI content tests in `taskboard/internal/web/ui_content_test.go` assert template rendering, required HTML DOM elements (TaskBoard title, project creation forms, To Do / Doing / Done Kanban columns), and static file delivery content types.

### Runtime Smoke Test
A runtime smoke test validates the built application server end-to-end:
1. Launch application binary or container: `cd taskboard && go run ./cmd/taskboard`
2. Verify liveness endpoint: `curl -s http://127.0.0.1:8080/api/health` returns `{"status":"ok"}` with HTTP status 200.
3. Verify readiness endpoint: `curl -s http://127.0.0.1:8080/api/ready` returns `{"status":"ready"}` with HTTP status 200.
4. Perform project creation round trip: `curl -s -X POST -H "Content-Type: application/json" -d '{"name":"Smoke Test"}' http://127.0.0.1:8080/api/projects` returns HTTP 201 with project JSON.
5. Verify HTML shell delivery: `curl -s http://127.0.0.1:8080/` returns HTML page containing "TaskBoard" heading and Kanban column headers.

Command to run all Go tests:
`cd taskboard && go test ./...`

## E2E / integration testing
### Playwright Browser E2E Test Suite
Browser-based E2E testing is configured via `taskboard/playwright.config.ts` and implemented in `taskboard/playwright/taskboard.spec.ts` using `@playwright/test`:
- Tests execute against a running TaskBoard application instance on host 127.0.0.1:8080.
- `taskboard/playwright/taskboard.spec.ts` tests real user workflows including creating a project via UI input controls, adding a task, advancing task status from `todo` to `doing` and `done` using the `Move forward` button, priority filtering (`high`/`medium`/`low`), and task deletion.
- Every test creates isolated test data with unique names and cleans up after completion.

Command to execute Playwright E2E tests:
`cd taskboard && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`

### PostgreSQL Integration Testing
- Integration tests in `taskboard/internal/store/postgres_integration_test.go` execute when `TASKBOARD_TEST_POSTGRES_URL` environment variable is defined.
- Exercises the store interface against a live PostgreSQL database, checking transactional advisory locks during schema setup, PostgreSQL identity primary key generation, UTC timestamp precision, parameter placeholder `$1` behavior, and foreign key cascade deletion.

Command to build container image:
`cd taskboard && docker build -t taskboard:local .`

## Delivery phases

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
    file_dependencies: {}

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
    qa_verify_command: 'cd taskboard && go test ./...'
    depends_on:
      - phase-1-core
    file_dependencies:
      taskboard/internal/store/postgres.go:
        - taskboard/internal/store/store.go
      taskboard/internal/store/store_test.go:
        - taskboard/internal/store/sqlite.go
      taskboard/internal/store/postgres_integration_test.go:
        - taskboard/internal/store/postgres.go
      taskboard/cmd/taskboard/main.go:
        - taskboard/internal/web/app.go
      taskboard/internal/web/app.go:
        - taskboard/internal/store/store.go
        - taskboard/web/assets.go
      taskboard/internal/web/api.go:
        - taskboard/internal/web/app.go
      taskboard/web/assets.go:
        - taskboard/web/templates/index.html
      taskboard/internal/web/api_test.go:
        - taskboard/internal/web/api.go

  - id: phase-2-kanban
    title: Tasks, Kanban UI, and full Go tests
    spec_focus: Tasks, Kanban UI, and full Go tests
    required_files:
      - taskboard/web/static/app.js
      - taskboard/web/static/style.css
      - taskboard/internal/web/app_test.go
      - taskboard/internal/web/ui_content_test.go
    qa_verify_command: 'cd taskboard && go test ./...'
    depends_on:
      - phase-1-app
    file_dependencies:
      taskboard/web/static/app.js:
        - taskboard/web/assets.go
      taskboard/web/static/style.css:
        - taskboard/web/assets.go
      taskboard/internal/web/app_test.go:
        - taskboard/internal/web/app.go
      taskboard/internal/web/ui_content_test.go:
        - taskboard/internal/web/app.go

  - id: phase-3-browser
    title: Playwright integration and browser QA
    spec_focus: Playwright integration and browser QA
    required_files:
      - taskboard/package.json
      - taskboard/package-lock.json
      - taskboard/playwright.config.ts
      - taskboard/playwright/taskboard.spec.ts
    qa_verify_command: 'cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright'
    depends_on:
      - phase-2-kanban
    file_dependencies:
      taskboard/playwright/taskboard.spec.ts:
        - taskboard/playwright.config.ts

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
    depends_on:
      - phase-3-browser
    file_dependencies:
      taskboard/Dockerfile:
        - taskboard/cmd/taskboard/main.go
      taskboard/deploy/k8s/deployment.yaml:
        - taskboard/Dockerfile

  - id: phase-5-release
    title: Final integration and release QA
    spec_focus: Final integration and release QA
    required_files: []
    qa_verify_command: 'cd taskboard && go test ./...'
    depends_on:
      - phase-4-deploy
    file_dependencies: {}
```

## Requirements

### phase-1-core
- Create project layout and Go module in `taskboard/go.mod` and `taskboard/go.sum`.
- Create configuration parsing in `taskboard/internal/config/config.go` handling environment variables TASKBOARD_DB_DRIVER, TASKBOARD_DB_PATH, TASKBOARD_DATABASE_URL, TASKBOARD_HOST, and TASKBOARD_PORT.
- Implement core domain models and unicode length validation rules in `taskboard/internal/model/model.go`.
- Define shared Store repository interface and DDL constants in `taskboard/internal/store/store.go`.
- Implement SQLite persistence with foreign keys PRAGMA and busy timeout in `taskboard/internal/store/sqlite.go`.

### phase-1-app
- Implement PostgreSQL database engine with transactional advisory lock table creation in `taskboard/internal/store/postgres.go`.
- Implement unit tests for SQLite store contract in `taskboard/internal/store/store_test.go` and optional PostgreSQL integration tests in `taskboard/internal/store/postgres_integration_test.go`.
- Construct web server application setup and project REST API routes in `taskboard/internal/web/app.go` and `taskboard/internal/web/api.go`.
- Implement embed directive in `taskboard/web/assets.go` embedding `taskboard/web/templates/*`.
- Provide HTML layout structure in `taskboard/web/templates/index.html`.
- Create main application entrypoint in `taskboard/cmd/taskboard/main.go` with signal handling for graceful shutdown.
- Add API project integration tests in `taskboard/internal/web/api_test.go`.

### phase-2-kanban
- Implement client-side vanilla JavaScript in `taskboard/web/static/app.js` handling project creation/selection, task CRUD, status transitions (todo -> doing -> done), and filters.
- Implement CSS stylesheet in `taskboard/web/static/style.css` for Kanban columns and cards.
- Extend `taskboard/web/assets.go` to embed `taskboard/web/static/*` assets.
- Add Go web router integration tests in `taskboard/internal/web/app_test.go` and UI template assertions in `taskboard/internal/web/ui_content_test.go`.

### phase-3-browser
- Configure Playwright dependencies in `taskboard/package.json` and `taskboard/package-lock.json`.
- Add test configuration in `taskboard/playwright.config.ts`.
- Implement E2E browser test suite in `taskboard/playwright/taskboard.spec.ts` testing project creation, task movement, priority filtering, and task deletion.

### phase-4-deploy
- Write multi-stage Dockerfile in `taskboard/Dockerfile` building non-root binary.
- Add build exclusions in `taskboard/.dockerignore`.
- Create Kubernetes deployment manifest in `taskboard/deploy/k8s/deployment.yaml` with liveness/readiness probes and Secret references.
- Create Kubernetes service manifest in `taskboard/deploy/k8s/service.yaml`.
- Create Kubernetes secret example manifest in `taskboard/deploy/k8s/secret.example.yaml`.

### phase-5-release
- Execute full integration regression suite.
- Verify clean checkout build, test coverage, and documentation completeness across both SQLite and PostgreSQL configurations.

## Acceptance mapping
- Multi-tenant/multi-project task tracking: satisfied by `taskboard/internal/model/model.go` and `taskboard/internal/store/store.go`.
- Dual database support (SQLite + PostgreSQL): satisfied by `taskboard/internal/store/sqlite.go` and `taskboard/internal/store/postgres.go`.
- Embedded assets & web application: satisfied by `taskboard/web/assets.go`, `taskboard/internal/web/app.go`, `taskboard/web/templates/index.html`, `taskboard/web/static/app.js`, and `taskboard/web/static/style.css`.
- E2E Playwright verification: satisfied by `taskboard/playwright/taskboard.spec.ts`.
- Docker & Kubernetes deployability: satisfied by `taskboard/Dockerfile`, `taskboard/deploy/k8s/deployment.yaml`, `taskboard/deploy/k8s/service.yaml`, and `taskboard/deploy/k8s/secret.example.yaml`.
