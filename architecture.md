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
| `taskboard/internal/model/model.go` | Project struct, Task struct, Status and Priority types/constants, TaskFilter struct, ValidateProject, ValidateTask functions | SQL DDL statements, HTTP responses |
| `taskboard/internal/store/store.go` | Store interface, ErrNotFound and repository error constants, DDL schema constants | HTTP routing logic |
| `taskboard/internal/store/sqlite.go` | SQLiteStore struct, NewSQLiteStore function | Postgres SQL dialect syntax |
| `taskboard/internal/store/postgres.go` | PostgresStore struct, NewPostgresStore function | SQLite PRAGMA commands |
| `taskboard/web/assets.go` | FS embed variable | Business logic |
| `taskboard/internal/web/app.go` | App struct, NewApp function, Handler method | SQL statements |
| `taskboard/internal/web/api.go` | API route handler methods on App struct | Template rendering |
| `taskboard/cmd/taskboard/main.go` | main function | Business domain logic |

## Store interface signatures

The shared database contract is defined in `taskboard/internal/store/store.go`:

```go
type Store interface {
	Ping(ctx context.Context) error
	GetProject(ctx context.Context, id int64) (*model.Project, error)
	CreateProject(ctx context.Context, p *model.Project) error
	ListProjects(ctx context.Context) ([]*model.Project, error)
	DeleteProject(ctx context.Context, id int64) error
	CreateTask(ctx context.Context, t *model.Task) error
	ListTasks(ctx context.Context, filter model.TaskFilter) ([]*model.Task, error)
	UpdateTask(ctx context.Context, t *model.Task) error
	DeleteTask(ctx context.Context, id int64) error
	Close() error
}
```

## HTTP + entrypoint integration

HTML UI is served at GET / using embedded templates from `taskboard/web/templates/index.html`. Static assets are served at GET /static/app.js and static stylesheet from embedded assets in `taskboard/web/static/app.js` and `taskboard/web/static/style.css`.

| Method | Path | Success | Errors |
| --- | --- | --- | --- |
| GET | `/api/health` | 200 `{"status":"ok"}` | — |
| GET | `/api/ready` | 200 `{"status":"ready"}` when DB responds | 503 `{"error":"database unavailable"}` |
| GET | `/api/projects` | 200 JSON array ordered by ID | 500 on storage failure |
| POST | `/api/projects` | 201 project JSON | 400 invalid name |
| DELETE | `/api/projects/{project_id}` | 204 empty body | 404 missing |
| GET | `/api/projects/{project_id}/tasks` | 200 JSON array ordered by ID | 404 missing project; 400 invalid filter |
| POST | `/api/projects/{project_id}/tasks` | 201 task JSON | 404 missing project; 400 invalid fields |
| PATCH | `/api/tasks/{task_id}` | 200 updated task JSON | 404 missing; 400 invalid fields |
| DELETE | `/api/tasks/{task_id}` | 204 empty body | 404 missing |

Server entrypoint (`taskboard/cmd/taskboard/main.go`):
1. Loads configuration via config.LoadConfig().
2. Selects database driver based on TASKBOARD_DB_DRIVER (sqlite or postgres) and initializes sqlite.NewSQLiteStore(ctx, cfg.DBPath) or postgres.NewPostgresStore(ctx, cfg.DatabaseURL).
3. Constructs web application instance app := web.NewApp(s). web.NewApp configures all page routes, embedded static file handlers from web.FS, and API routes on its internal http.ServeMux.
4. Passes app.Handler() directly as the handler for http.Server (without registering routes on http.DefaultServeMux).
5. Starts HTTP server on configured host and port (cfg.Host:cfg.Port).
6. Handles graceful SIGINT/SIGTERM shutdown, closing the server with timeout context and calling s.Close().

## Requirements

### phase-1-core
- Configuration loading from environment variables in `taskboard/internal/config/config.go`:
  - TASKBOARD_PORT (default: 8080)
  - TASKBOARD_HOST (default: 127.0.0.1)
  - TASKBOARD_DB_DRIVER (supported: sqlite, postgres; default: sqlite)
  - TASKBOARD_DB_PATH (default: taskboard.db)
  - TASKBOARD_DATABASE_URL (default: empty string)
- Domain models in `taskboard/internal/model/model.go`:
  - Project struct (id int64, name string, created_at time.Time)
  - Task struct (id int64, project_id int64, title string, description string, status string, priority string, created_at time.Time, updated_at time.Time)
  - TaskFilter struct (project_id int64, status string, priority string)
  - Unicode character count validation using utf8.RuneCountInString: Project name (1-80 runes), Task title (1-120 runes), Task description (max 1000 runes).
  - Status enum validation: todo, doing, done.
  - Priority enum validation: low, medium, high.
- Store interface in `taskboard/internal/store/store.go` with exact methods: Ping, GetProject, CreateProject, ListProjects, DeleteProject, CreateTask, ListTasks, UpdateTask, DeleteTask, Close.
- SQLite store implementation in `taskboard/internal/store/sqlite.go`:
  - Enables foreign keys (PRAGMA foreign_keys = ON;) on connection setup.
  - Configures 5000ms busy timeout.
  - Auto-creates projects and tasks tables if not existing.
  - Supports cascading deletion of tasks when a project is deleted.

### phase-1-app
- PostgreSQL store implementation in `taskboard/internal/store/postgres.go`:
  - Auto-creates projects and tasks tables using transactional advisory locks (pg_advisory_xact_lock) for safe concurrent execution.
- Substantive store unit tests (`taskboard/internal/store/store_test.go`) and PostgreSQL integration tests (`taskboard/internal/store/postgres_integration_test.go`) asserting actual CRUD operations, foreign key constraints, filtering, and cascade behavior.
- Application entrypoint in `taskboard/cmd/taskboard/main.go`, web router setup in `taskboard/internal/web/app.go`, project API endpoints in `taskboard/internal/web/api.go`, embedded assets in `taskboard/web/assets.go` (using `//go:embed templates/*` initially), HTML template in `taskboard/web/templates/index.html`, and API unit tests in `taskboard/internal/web/api_test.go`.

### phase-2-kanban
- Tasks API endpoints and dynamic rendering of Kanban dashboard.
- Dynamic asset embedding extending `taskboard/web/assets.go` to `//go:embed templates/* static/*` with static assets `taskboard/web/static/app.js` and `taskboard/web/static/style.css`.
- App unit tests (`taskboard/internal/web/app_test.go`) and UI content unit tests (`taskboard/internal/web/ui_content_test.go`).
- UI JavaScript logic (`taskboard/web/static/app.js`) supporting task creation/editing, status progression across Kanban columns, and filtering by status and priority.
- CSS styling (`taskboard/web/static/style.css`) for responsive Kanban column layout.

### phase-3-browser
- Playwright E2E integration test suite (`taskboard/playwright/taskboard.spec.ts`), dependencies (`taskboard/package.json`, `taskboard/package-lock.json`), and configuration (`taskboard/playwright.config.ts`).

### phase-4-deploy
- Multi-stage Dockerfile (`taskboard/Dockerfile`) building Go binary in builder stage and producing non-root minimal runtime image.
- Dockerignore configuration in `taskboard/.dockerignore`.
- Kubernetes deployment manifests (`taskboard/deploy/k8s/deployment.yaml`, `taskboard/deploy/k8s/service.yaml`, `taskboard/deploy/k8s/secret.example.yaml`).

### phase-5-release
- Final integration, end-to-end verification, browser QA checks, and container image deployment validation without new first-delivery files.

## Docker & Deployment
- Multi-stage build in `taskboard/Dockerfile`:
  - Builder stage uses golang:1.25-alpine to compile binary with CGO_ENABLED=0 GOOS=linux.
  - Final runtime stage uses alpine:3.20 or scratch/distroless with non-root user execution.
- Copies compiled binary /app/taskboard and embedded static assets from builder stage.
- Exposed port: EXPOSE 8080.
- Runtime entrypoint: CMD ["/app/taskboard"].

## Unit tests
- `taskboard/internal/store/store_test.go`: SQLite table creation, Project CRUD, Task CRUD with filters, cascade deletion, foreign key validation.
- `taskboard/internal/store/postgres_integration_test.go`: PostgreSQL integration testing for Store methods when TASKBOARD_TEST_POSTGRES_URL is provided.
- `taskboard/internal/web/api_test.go`: REST API route handler tests for status codes (200, 201, 204, 400, 404, 500), payload validation, filter parameters.
- `taskboard/internal/web/app_test.go`: Web app routing, middleware, and lifecycle tests.
- `taskboard/internal/web/ui_content_test.go`: Tests verifying template parsing and embedded static content delivery.

## Integration and testing

### Test Strategy
The testing framework comprises unit, integration, and Playwright E2E tiers:
- Domain model tests: validate utf8.RuneCountInString bounds (project name 1-80, task title 1-120, task description max 1000) and strict enums (todo, doing, done, low, medium, high).
- Store contract tests: verify DDL idempotency, ascending ID ordering, parameterized queries, transactional atomicity, combined status/priority AND filtering, and CASCADE deletion on SQLite (t.TempDir()) and PostgreSQL.
- HTTP API integration tests: exercise handlers using httptest.Server and httptest.ResponseRecorder for standard status codes (200, 201, 204, 400, 404, 500, 503) and JSON payload verification.
- UI & Static Asset tests: assert go:embed asset loading for `taskboard/web/templates/index.html`, `taskboard/web/static/app.js`, and `taskboard/web/static/style.css`.
- Browser E2E tests: automation via Playwright validating real control interactions, Kanban column transitions, filtering, and project deletion.

### Unit Test Structure
- `taskboard/internal/store/store_test.go`: SQLite storage unit tests for DDL, project/task CRUD, filters, and cascade delete.
- `taskboard/internal/store/postgres_integration_test.go`: PostgreSQL integration tests executed when TASKBOARD_TEST_POSTGRES_URL is set.
- `taskboard/internal/web/api_test.go`: REST API route handler tests for status codes, validation, filters, and JSON response bodies.
- `taskboard/internal/web/app_test.go`: Web router setup, asset embedding, and server lifecycle unit tests.
- `taskboard/internal/web/ui_content_test.go`: Template parsing and static asset content tests.

### Runtime Smoke Test Procedure
1. Start TaskBoard server locally with default SQLite configuration:
   `cd taskboard && go run ./cmd/taskboard`
2. Verify liveness and readiness endpoints:
   - curl -s http://127.0.0.1:8080/api/health -> {"status":"ok"}
   - curl -s http://127.0.0.1:8080/api/ready -> {"status":"ready"}
3. Verify HTML UI dashboard and static assets:
   - curl -s http://127.0.0.1:8080/ -> HTML page containing TaskBoard heading
   - curl -s http://127.0.0.1:8080/static/app.js -> Non-empty JavaScript asset
   - curl -s http://127.0.0.1:8080/static/style.css -> Non-empty CSS asset
4. Execute REST API CRUD flow:
   - Create project: curl -s -X POST -H "Content-Type: application/json" -d '{"name":"Demo"}' http://127.0.0.1:8080/api/projects
   - List projects: curl -s http://127.0.0.1:8080/api/projects
   - Create task: curl -s -X POST -H "Content-Type: application/json" -d '{"title":"Task 1","priority":"high"}' http://127.0.0.1:8080/api/projects/1/tasks
   - List tasks: curl -s http://127.0.0.1:8080/api/projects/1/tasks
   - Patch task: curl -s -X PATCH -H "Content-Type: application/json" -d '{"status":"doing"}' http://127.0.0.1:8080/api/tasks/1
   - Delete task: curl -s -X DELETE http://127.0.0.1:8080/api/tasks/1
   - Delete project: curl -s -X DELETE http://127.0.0.1:8080/api/projects/1
5. Run full Go unit and integration test suite:
   `cd taskboard && go test ./...`

## E2E / integration testing
- Playwright suite in `taskboard/playwright/taskboard.spec.ts` verifies end-to-end web UI interactions against a running server:
  - Project creation and selection.
  - Task creation in To Do column.
  - Moving task state from To Do to In Progress to Done.
  - Task filtering by status and priority.
  - Project deletion and cascading task removal.

## Acceptance mapping
- Modular store abstraction satisfies requirement for dynamic SQLite and PostgreSQL persistence without code modifications.
- Complete HTTP route suite satisfies SPEC endpoints for REST API and HTML dashboard.
- Containerization and Kubernetes manifests provide ready-to-deploy specifications for k3s cluster environments.

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
    file_dependencies:
      taskboard/internal/config/config.go: [taskboard/go.mod]
      taskboard/internal/model/model.go: [taskboard/go.mod]
      taskboard/internal/store/store.go: [taskboard/go.mod, taskboard/internal/model/model.go]
      taskboard/internal/store/sqlite.go: [taskboard/go.mod, taskboard/internal/store/store.go]

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
    depends_on: [phase-1-core]
    file_dependencies:
      taskboard/internal/store/postgres.go: [taskboard/internal/store/store.go]
      taskboard/internal/store/store_test.go: [taskboard/internal/store/sqlite.go]
      taskboard/internal/store/postgres_integration_test.go: [taskboard/internal/store/postgres.go]
      taskboard/internal/web/app.go: [taskboard/web/assets.go, taskboard/internal/store/store.go]
      taskboard/internal/web/api.go: [taskboard/internal/web/app.go, taskboard/internal/model/model.go, taskboard/internal/store/store.go]
      taskboard/web/assets.go: [taskboard/web/templates/index.html]
      taskboard/cmd/taskboard/main.go: [taskboard/internal/config/config.go, taskboard/internal/store/sqlite.go, taskboard/internal/web/app.go]

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
      taskboard/web/static/app.js: [taskboard/web/assets.go]
      taskboard/web/static/style.css: [taskboard/web/assets.go]
      taskboard/internal/web/app_test.go: [taskboard/internal/web/app.go]
      taskboard/internal/web/ui_content_test.go: [taskboard/web/assets.go]

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
    file_dependencies:
      taskboard/package-lock.json: [taskboard/package.json]
      taskboard/playwright.config.ts: [taskboard/package.json]
      taskboard/playwright/taskboard.spec.ts: [taskboard/playwright.config.ts]

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
    file_dependencies:
      taskboard/Dockerfile: [taskboard/cmd/taskboard/main.go, taskboard/web/assets.go]
      taskboard/deploy/k8s/deployment.yaml: [taskboard/Dockerfile]

  - id: phase-5-release
    title: Final integration and release QA
    spec_focus: Final integration and release QA
    required_files: []
    qa_verify_command: 'cd taskboard && go test ./... && docker build -t taskboard:local .'
    depends_on: [phase-4-deploy]
