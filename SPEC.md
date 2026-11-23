# TaskBoard — Go Kanban MVP (Gas Town rig-flow and k3s deployment)

## 1. Goal

Implement a small task management web application called **TaskBoard**. Users create and delete projects, create and edit tasks, move tasks through three Kanban states, filter by status and priority, and delete tasks. Data persists in **SQLite or PostgreSQL**, selected at runtime without changing application code.

This repository is an external test rig for Gas Town `rig-flow`. It must exercise architecture, planning, implementation, `test_plan`, `test_review`, and QA handoffs through deterministic, substantive tests. The completed application must also produce a container image deployable to an existing k3s cluster using an **existing PostgreSQL service**; do not deploy PostgreSQL as part of this project.

Keep the implementation literal and compact: **Go 1.25+, standard-library HTTP, `database/sql`, server-rendered HTML, vanilla JavaScript, plain CSS, no frontend build, no ORM, no authentication, and no external APIs**.

## 2. Fixed constraints

- Go >=1.25; use a pinned Go toolchain and committed `go.mod`/`go.sum`.
- HTTP routing uses `net/http` ServeMux with method-aware patterns; JSON uses `encoding/json`; HTML uses `html/template`; assets use `embed`.
- Database interface uses `database/sql`. SQLite driver: `modernc.org/sqlite` (pure Go, CGO disabled). PostgreSQL driver: `github.com/jackc/pgx/v5/stdlib`.
- Use explicit SQL and small repository functions; no ORM, migrations framework, dependency-injection framework, frontend framework, or code generator.
- Go tests use standard `testing` and `httptest`; Playwright uses `@playwright/test` only for browser E2E.
- Default bind address for local development: `127.0.0.1:8080`; container sets `TASKBOARD_HOST=0.0.0.0`.
- Do not use SSE, WebSockets, background jobs, external credentials, analytics, drag-and-drop, or authentication.
- Include a multi-stage Dockerfile, `.dockerignore`, Kubernetes Deployment/Service and Secret example. Do not include a PostgreSQL Deployment, StatefulSet, chart, or PVC for PostgreSQL mode.
- Never commit secrets, database files, compiled binaries, `node_modules`, Playwright reports, or generated test artifacts.
- Keep all canonical deliverables within the repository layout below. Do not introduce duplicate app implementations or wrapper projects.

## 3. Canonical project layout

`SPEC.md` lives at the rig/repository root. `taskboard/` is the sole Go module and Gas Town `layout_root`; all paths below are repository-relative. Run application/test commands from `taskboard/` unless noted.

```text
SPEC.md
taskboard/
  go.mod
  go.sum
  README.md
  .gitignore
  .dockerignore
  Dockerfile
  cmd/
    taskboard/
      main.go
  internal/
    config/
      config.go
    model/
      model.go
    store/
      store.go
      sqlite.go
      postgres.go
      store_test.go
      postgres_integration_test.go
    web/
      app.go
      api.go
      app_test.go
      api_test.go
      ui_content_test.go
  web/
    templates/
      index.html
    static/
      app.js
      style.css
  deploy/
    k8s/
      deployment.yaml
      service.yaml
      secret.example.yaml
  package.json
  package-lock.json
  playwright.config.ts
  playwright/
    taskboard.spec.ts
```

Do not create files merely to match a directory listing before their first-delivery phase. The `go.sum` and `package-lock.json` files are required lockfiles and must be committed. `go:embed` must embed templates and static files from a legal location: because `go:embed` cannot traverse `..`, place the embed directive in `taskboard/web/assets.go` (an additional canonical file, first owned in phase 1b), with `//go:embed templates/* static/*`. `internal/web` imports this asset package. Do not duplicate static files or templates.

All schema and dialect differences live in `internal/store`; pure validation lives in `internal/model`; HTTP routes and response encoding live in `internal/web`; environment parsing lives in `internal/config`; startup and shutdown live in `cmd/taskboard/main.go`. The HTTP app is constructed through `web.NewApp(store, assets)` or equivalent explicit constructor; no mutable global database handle.

## 4. Configuration

| Variable | Default | Behavior |
|---|---|---|
| `TASKBOARD_DB_DRIVER` | `sqlite` | Exactly `sqlite` or `postgres` |
| `TASKBOARD_DB_PATH` | `taskboard.db` | SQLite file path; ignored for PostgreSQL |
| `TASKBOARD_DATABASE_URL` | unset | Required for PostgreSQL; PostgreSQL URL or supported libpq connection string |
| `TASKBOARD_HOST` | `127.0.0.1` | HTTP listen host; set `0.0.0.0` in k3s |
| `TASKBOARD_PORT` | `8080` | HTTP listen port, integer 1–65535 |

Invalid configuration fails startup with a useful error and nonzero exit. Never print the full PostgreSQL connection URL or passwords in logs or error responses. Configuration is read once at process startup; no live driver switching is required. SQLite paths are relative to the process working directory unless absolute. For PostgreSQL, require a nonempty connection URL; do not silently fall back to SQLite on failure.

Example local startup:

```bash
cd taskboard
go run ./cmd/taskboard
```

Example PostgreSQL startup (illustrative only; do not commit real credentials):

```bash
TASKBOARD_DB_DRIVER=postgres \
TASKBOARD_DATABASE_URL='postgres://taskboard:example@postgres.default.svc.cluster.local:5432/taskboard?sslmode=require' \
TASKBOARD_HOST=0.0.0.0 \
go run ./cmd/taskboard
```

`sslmode` must be chosen to match the existing cluster's PostgreSQL configuration; do not assume TLS is disabled or enabled. Production credentials must be injected from Kubernetes Secrets.

## 5. Data model and invariants

The application supports one logical schema on both engines. Create tables if absent on startup; no general-purpose migration framework is required for this MVP. Schema initialization must be safe for repeated starts and simultaneous PostgreSQL starts: use transactional PostgreSQL advisory locking (or another explicitly tested equivalent) around schema creation. Do not use destructive schema resets.

### `projects`

| Field | Rules |
|---|---|
| `id` | Positive, database-generated 64-bit integer primary key |
| `name` | Required, trimmed, 1–80 Unicode code points |
| `created_at` | UTC ISO 8601 / RFC3339 timestamp with explicit timezone |

Project names need not be unique.

### `tasks`

| Field | Rules |
|---|---|
| `id` | Positive, database-generated 64-bit integer primary key |
| `project_id` | Required FK to `projects(id)` with `ON DELETE CASCADE` |
| `title` | Required, trimmed, 1–120 Unicode code points |
| `description` | String, at most 1000 Unicode code points, default `""` |
| `status` | Exactly `todo`, `doing`, `done`; default `todo` |
| `priority` | Exactly `low`, `medium`, `high`; default `medium` |
| `created_at` | UTC RFC3339 timestamp |
| `updated_at` | UTC RFC3339 timestamp |

Index `tasks.project_id`. Projects and tasks list in ascending `id` order; there is no custom ranking. SQLite must enable foreign keys **on every pooled connection** (for example via driver DSN configuration; a one-time PRAGMA on one connection is insufficient) and use a 5000 ms busy timeout. Configure a conservative SQLite connection pool if needed for locking correctness. PostgreSQL uses appropriate identity columns and `$1`-style placeholders; SQLite uses `?` placeholders and appropriate generated-ID retrieval. Never interpolate user input into SQL.

Use `utf8.RuneCountInString` for length validation, matching the original Python Unicode code-point behavior. Trim only project names and task titles; retain descriptions verbatim. A missing or `null` description on task **creation** becomes `""`; explicit `null` on PATCH is invalid. Unknown status/priority is HTTP 400. Creating a task in a nonexistent project returns 404. Deleting a project cascades to its tasks. Timestamps must have explicit UTC timezone; tests must not assert exact wall-clock time.

Use a shared repository contract, with SQLite and PostgreSQL implementations exposing identical behavior. Keep dialect-specific DDL and SQL confined to `internal/store`. Write operations use transactions where needed for atomicity. Use request contexts and bounded connection establishment. No database file is bundled in the container.

## 6. HTTP API contract

Application-level 400/404 errors have exactly `{"error":"human-readable message"}`. Malformed JSON, wrong JSON types, invalid IDs, and other client input failures return 400 in this Go implementation; unlike FastAPI, there is **no 422 requirement**. Unknown JSON properties may be ignored. For malformed integer path segments, return 400 (if matched) or 404 (if no route matches), consistently documented and tested. IDs must be positive.

| Method | Path | Success | Errors |
|---|---|---|---|
| GET | `/api/health` | 200 `{"status":"ok"}` | — |
| GET | `/api/ready` | 200 `{"status":"ready"}` when DB responds | 503 `{"error":"database unavailable"}` |
| GET | `/api/projects` | 200 JSON array ordered by ID | 500 on storage failure |
| POST | `/api/projects` | 201 project JSON | 400 invalid name |
| DELETE | `/api/projects/{project_id}` | 204 empty body | 404 missing |
| GET | `/api/projects/{project_id}/tasks` | 200 JSON array ordered by ID | 404 missing project; 400 invalid filter |
| POST | `/api/projects/{project_id}/tasks` | 201 task JSON | 404 missing project; 400 invalid fields |
| PATCH | `/api/tasks/{task_id}` | 200 updated task JSON | 404 missing; 400 invalid fields |
| DELETE | `/api/tasks/{task_id}` | 204 empty body | 404 missing |

`/api/health` is a lightweight liveness endpoint independent of database availability. `/api/ready` pings the selected database with a short timeout; it must not leak credentials or internal SQL errors. Database errors not attributable to client input return 500 with a safe generic JSON error. HTTP handlers must not silently convert database errors into empty lists.

Project JSON:

```json
{"id":1,"name":"Demo","created_at":"2026-01-01T00:00:00Z"}
```

Project POST: `{"name":"Demo"}`.

Task JSON:

```json
{"id":1,"project_id":1,"title":"Build API","description":"Details","status":"todo","priority":"high","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}
```

Task POST: `{"title":"Build API","description":"Details","priority":"high"}`. Description defaults to `""`, priority to `medium`, and status is **always** `todo` at creation, ignoring any client-supplied status, IDs, or timestamps. PATCH accepts any **nonempty subset** of `title`, `description`, `priority`, and `status`. Missing fields stay unchanged; explicitly null patched fields are invalid; project reassignment is forbidden. Unknown PATCH properties do not count as a change. Filters `status` and `priority` are optional, validated enums and combine with AND. Empty lists serialize as `[]`, never `null`. DELETE 204 responses have no body.

## 7. Browser/UI contract

- `GET /` serves real server-rendered HTML using `html/template` and the embedded `web/templates/index.html`.
- `GET /static/app.js` and `/static/style.css` serve the embedded actual assets with appropriate content types.
- Heading `TaskBoard`; project input and `Add project` button.
- Project selector populated via `/api/projects` on page load; selecting a project displays its tasks.
- Selected-project task form: required title, optional description, priority select `low`/`medium`/`high`, and `Add task`.
- Three visible Kanban columns: `To Do`, `Doing`, `Done`.
- Each card shows title and priority, with `Move forward`, `Edit`, and `Delete`. Moving follows `todo → doing → done`; the forward control is hidden or disabled for `done`.
- Editing supports title, description, priority through native controls. The API may patch status directly, but the UI changes status only via `Move forward`.
- Status filter (`all`, `todo`, `doing`, `done`) and priority filter (`all`, `low`, `medium`, `high`) refresh the list using API query parameters; filtered-out cards are not rendered.
- Project deletion requires `window.confirm`, cascades to tasks, and refreshes the selection.
- Show useful empty states for no projects and no matching tasks; show API failures in a visible error region.
- Insert user-provided text using `textContent` or safe DOM APIs; never interpolate user text into `innerHTML`.
- Vanilla JavaScript and CSS only; no drag-and-drop, frameworks, build step, animations, or icon dependencies. Layout remains usable at desktop and narrow widths.

## 8. Testing contract

Tests must assert actual state changes, response bodies, ordering, validation, and persistence, not merely HTTP 200. Use isolated temporary SQLite database files via `t.TempDir()`, never the runtime `taskboard.db`. Use Go `testing` and `httptest`; tests must not depend on execution order. Shared store contract tests should run against both dialects when configured.

### Store tests (`internal/store/store_test.go`)

1. Fresh schema is empty; repeated initialization is idempotent.
2. Create/list projects with ascending IDs.
3. Create tasks with default status and priority and valid UTC timestamps.
4. Filter by status, priority, and both together.
5. Patch all mutable fields and preserve omitted fields.
6. Delete project and verify cascading task deletion.
7. Confirm foreign-key enforcement and persistence across close/reopen.
8. Verify Unicode code-point boundaries and safe SQL parameterization.

### API tests (`internal/web/api_test.go`)

1. Exact liveness JSON and ready response against healthy SQLite.
2. Project create/list/delete round trip.
3. Task create/list/patch/delete round trip.
4. Reject blank project name/title, invalid enum values, overlong fields, empty PATCH, and explicit PATCH nulls.
5. Return 404 for missing entities; malformed payload/IDs produce documented client errors.
6. Verify empty lists are `[]`, 204 bodies are empty, and list order is ascending ID.
7. Verify combined filters use AND and reject invalid filters.
8. Verify client-provided task status/IDs/timestamps cannot override server-created values.
9. Verify safe error response on database failure and readiness 503.

### UI content tests (`internal/web/ui_content_test.go`)

1. GET `/` contains TaskBoard, both forms, and all three Kanban headings.
2. Static JS and CSS return 200 with nonempty real content.

### PostgreSQL integration tests (`internal/store/postgres_integration_test.go`)

Run only when `TASKBOARD_TEST_POSTGRES_URL` is nonempty. Require a **dedicated disposable PostgreSQL test database**; never use the production database. Tests must use an isolated schema or uniquely named test objects, clean up only their own data, and exercise the same store contract as SQLite, including cascade deletion, ordering, filters, timestamp encoding, and schema idempotency. Avoid destructive operations on shared schemas. Test startup readiness and reconnect behavior. The default Gas Town gate must not depend on PostgreSQL availability; a configured PostgreSQL test gate is mandatory before claiming PostgreSQL deployment readiness.

### Playwright E2E (`playwright/taskboard.spec.ts`)

1. Initial page displays TaskBoard and all three columns.
2. Create a project and task via browser controls, verify `To Do`, click `Move forward` to `Doing` and then `Done`, delete task, verify disappearance.
3. Create two tasks with different priorities, filter to `high`, verify only high-priority task is visible, reset filter, verify both return.

Each test uses a unique project name and cleans up projects it creates before/after as appropriate; tests cannot rely on prior runs. At least one test **must perform real browser input/click/selection and assert resulting DOM or API state change**. Static content-only E2E is insufficient and QA must reject it. Use a host-accessible running application; do not create a second application container for browser tests.

## 9. Container and k3s requirements

### Docker image

- Multi-stage Dockerfile: pinned Go builder, `CGO_ENABLED=0`, `GOOS=linux`, architecture chosen by target platform, build `./cmd/taskboard`.
- Small final image with CA certificates, a non-root UID, and embedded HTML/JS/CSS; no Go toolchain or Node runtime in final image.
- Entrypoint runs the compiled Go binary. Expose/document TCP 8080. `TASKBOARD_HOST=0.0.0.0` in the container.
- Support `linux/amd64` and `linux/arm64` builds as appropriate for the k3s nodes; do not hardcode an incompatible architecture.
- Build context is `taskboard/`. `.dockerignore` excludes databases, secrets, `.git`, Node dependencies, test reports, and build outputs.
- Include exact `docker build` and smoke-test instructions in README. No container image must be pushed automatically.

### Kubernetes manifests

`deploy/k8s/deployment.yaml` must define a single-replica Deployment by default, a container on port 8080, non-root security context, reasonable resource requests/limits, liveness probe `GET /api/health`, readiness probe `GET /api/ready`, and PostgreSQL configuration via a Secret reference. Use a clearly replaceable image name/tag; do not assume a registry. No privileged container, host networking, or hostPath volume.

`deploy/k8s/service.yaml` must define an internal `ClusterIP` Service targeting port 8080. Ingress is out of scope because clusters differ; README may explain how to add an existing ingress controller's configuration.

`deploy/k8s/secret.example.yaml` is an **example only**, with obvious placeholder values and clear instructions not to apply it unchanged or commit real credentials. Prefer creating the actual Secret out of band. The Deployment reads `TASKBOARD_DB_DRIVER=postgres` and `TASKBOARD_DATABASE_URL` from a Secret key named `database-url`. The example uses no live password. Never log the URL. PostgreSQL service address and TLS options are cluster-specific.

In PostgreSQL mode, no TaskBoard PVC is required. Document that SQLite mode needs persistent storage for data durability and is not intended for multi-replica deployment; do not provide a misleading ephemeral SQLite production deployment. PostgreSQL supports later horizontal scaling, but replicas remain one for this MVP and schema initialization must tolerate concurrent starts. Ensure graceful SIGTERM shutdown with an HTTP server shutdown timeout and DB close.

## 10. Delivery phases, ownership, and gates

**Hard rule:** Every delivery phase/subphase owns at most **10 unique first-delivery files**. Files may be extended in later phases but have exactly one first-delivery owner. Do not add aliases or duplicate implementations to satisfy a generated profile. Deliver subphases in the order below; preserve explicit dependencies and keep tests green after each gate.

### Phase 1a — Go foundation, configuration, models, SQLite store

**First-delivery files (8):**

1. `taskboard/go.mod`
2. `taskboard/go.sum`
3. `taskboard/README.md`
4. `taskboard/.gitignore`
5. `taskboard/internal/config/config.go`
6. `taskboard/internal/model/model.go`
7. `taskboard/internal/store/store.go`
8. `taskboard/internal/store/sqlite.go`

Implement configuration parsing, validation, domain models, shared store contract, SQLite schema and CRUD, and database connection rules. `go.sum` must be generated by dependency resolution, not handwritten. No web server required yet. This subphase may create its repository tests only in the next subphase; compile the packages and use a small temporary local smoke harness if needed, without committing extra deliverables.

**Gate:** `cd taskboard && go mod verify && go test ./internal/config ./internal/model ./internal/store` (packages without tests must compile).

### Phase 1b — PostgreSQL store and runnable project API

**Depends on:** Phase 1a.

**First-delivery files (9):**

1. `taskboard/internal/store/postgres.go`
2. `taskboard/internal/store/store_test.go`
3. `taskboard/internal/store/postgres_integration_test.go`
4. `taskboard/cmd/taskboard/main.go`
5. `taskboard/internal/web/app.go`
6. `taskboard/internal/web/api.go`
7. `taskboard/web/assets.go`
8. `taskboard/web/templates/index.html`
9. `taskboard/internal/web/api_test.go`

Implement both dialects, idempotent schema setup, Go server lifecycle, embedded real HTML shell, health/ready routes, project routes, and substantive project tests. `api.go` may initially contain project routes only. Make sure the embedded asset patterns are valid: the static directory and its assets are first introduced in Phase 2; for Phase 1b use an embed directive for `templates/*` only, then extend it in Phase 2 to include `static/*`. The HTML page is real, not a placeholder; the complete task controls arrive in Phase 2.

**Gate:** `cd taskboard && go test ./...` and manual smoke: `GET /api/health`, `GET /`, create/list/delete project. If `TASKBOARD_TEST_POSTGRES_URL` is configured, also run the PostgreSQL integration tests. Tests must not install dependencies during test review.

### Phase 2 — Tasks, Kanban UI, and full Go tests

**Depends on:** Phase 1b.

**First-delivery files (4):**

1. `taskboard/web/static/app.js`
2. `taskboard/web/static/style.css`
3. `taskboard/internal/web/app_test.go`
4. `taskboard/internal/web/ui_content_test.go`

Extend the existing store implementations, models, HTTP handlers, templates, and tests in place. Implement task CRUD, validation, project selection, Kanban columns, move-forward behavior, editing, deletion, filters, visible errors, and empty states. Extend `web/assets.go` to embed the static files. Expand store and API tests with substantive assertions. No new frontend build tool.

**Gate:** `cd taskboard && go test ./...`; manual browser smoke demonstrates project/task creation, movement, edit, filter, and deletion. Verify both `/static/*` resources are nonempty and the real UI renders.

### Phase 3 — Playwright integration and browser QA

**Depends on:** Phase 2.

**First-delivery files (4):**

1. `taskboard/package.json`
2. `taskboard/package-lock.json`
3. `taskboard/playwright.config.ts`
4. `taskboard/playwright/taskboard.spec.ts`

Use `@playwright/test` only as a development dependency. Preserve a canonical generated Playwright config if the host integration scaffold has already supplied it; otherwise create a minimal compatible config. Do not introduce npm frontend build scripts, duplicate configs/specs, wrapper files, or a second application container. Browser tests use the already-running host-accessible TaskBoard server. Update README with setup and E2E instructions.

**Gate:** With the server already running on `127.0.0.1:8080`, run `cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`. Install Node dependencies **before** test review, not during test review. QA must observe real browser interaction and state change.

### Phase 4 — Container build and k3s deployment artifacts

**Depends on:** Phase 3.

**First-delivery files (5):**

1. `taskboard/Dockerfile`
2. `taskboard/.dockerignore`
3. `taskboard/deploy/k8s/deployment.yaml`
4. `taskboard/deploy/k8s/service.yaml`
5. `taskboard/deploy/k8s/secret.example.yaml`

Build a production container using embedded assets and a pure-Go SQLite driver. Add non-root runtime, graceful shutdown, liveness/readiness probes, Secret-backed PostgreSQL URL, resource settings, and a ClusterIP Service. Document exact build/run/k3s instructions and the requirement for a dedicated PostgreSQL database/user. Do not install or manage PostgreSQL. Do not deploy automatically to a live cluster.

**Gate:** `cd taskboard && go test ./... && docker build -t taskboard:local .`; run a local container smoke test (`/api/health`, `/`, static files). Validate YAML structure and selectors, Secret key wiring, port matching, probes, and non-root settings. If Docker or Kubernetes tooling is unavailable, report the missing verification as **not run**, not passed. For a release claim of PostgreSQL readiness, run tests against a dedicated PostgreSQL test database and demonstrate the image starts in PostgreSQL mode with readiness healthy.

### Phase 5 — Final integration and release QA

**Depends on:** Phase 4. **First-delivery files: none.** Extend existing files only for bug fixes, documentation, and test improvements.

Run all gates, verify both DB backends when test PostgreSQL is available, inspect actual browser interactions, ensure the Docker image contains the expected embedded UI, and review manifests for secret safety and k3s suitability. Verify restart persistence using the same PostgreSQL database and no schema reset. Confirm no secrets, SQLite DBs, binaries, caches, `node_modules`, or reports are committed.

**Gate:** `go test ./...`, Playwright suite against running app, Docker build and smoke test, PostgreSQL integration gate when configured, and deployment manifest review. Explicitly report any unavailable infrastructure checks. Never claim `all_passed` when mandatory browser interactions or available required test gates fail.

### Suggested bead decomposition

| Bead | Responsibility | Depends on |
|---|---|---|
| B1 | Go module, config, models, SQLite schema | — |
| B2 | Shared repository CRUD and validation | B1 |
| B3 | PostgreSQL dialect and shared store tests | B2 |
| B4 | HTTP server, embed shell, health and projects API | B2, B3 |
| B5 | Task store/API and Go integration tests | B4 |
| B6 | Project UI and Kanban task interactions | B4, B5 |
| B7 | Filters, error handling, UI-content tests | B6 |
| B8 | Playwright real-interaction E2E and QA | B7 |
| B9 | Docker image and smoke checks | B8 |
| B10 | k3s manifests, PostgreSQL deployment configuration | B9 |
| B11 | Final cross-backend, browser, and release QA | B10 |

Planner may adapt bead IDs but must preserve dependencies, file ownership, and verify gates. Avoid parallel beads that independently rewrite `internal/web/api.go`, `internal/store/store.go`, or `web/templates/index.html` without an explicit integration sequence.

## 11. Gas Town workflow integration

- **Architect:** Produce `architecture.md` with module boundaries, database dialect strategy, embed path, runtime config, deployment topology, and machine-readable delivery phases/file dependencies.
- **Planner:** Produce `plan.md` and executable beads, assigning every canonical file to exactly one first-delivery phase; each phase owns no more than 10 files. Use the phase gates above.
- **Tester `test_plan`:** Produce `TEST_PLAN.md` mapping active-phase acceptance requirements to substantive Go/Playwright tests and bead IDs, including optional PostgreSQL test prerequisites.
- **Polecat:** Implement and commit actual code and tests; no feature is complete without a runnable assertion.
- **Tester `test_review`:** Execute the active phase's verification command, confirm substantive assertions and correct test isolation; **never install dependencies in test review**.
- **QA:** Smoke-test real HTTP responses, UI interactions and state changes, and container behavior in Phase 4; report failures with file and bead ID. Report unavailable external Docker/PostgreSQL/k3s checks accurately.

When `layout_root=taskboard`, machine-readable `delivery_phases` ownership and `file_dependencies` must use repository-relative paths with the `taskboard/` prefix. The canonical phase IDs are `phase-1-core`, `phase-1-app`, `phase-2-kanban`, `phase-3-browser`, `phase-4-deploy`, and `phase-5-release`; `phase-1-app` depends on `phase-1-core`. Do not move prerequisites into later phases or create path aliases to satisfy a generated profile. If a generated profile disagrees with this SPEC, correct/regenerate the profile before architecture validation rather than changing the architecture to match it.

## 12. Definition of done

1. Fresh checkout resolves Go dependencies from `taskboard/` and `go test ./...` passes without an external database.
2. SQLite is the default and persists data across application restarts; foreign keys and cascading deletes work.
3. With a dedicated test PostgreSQL URL, the same repository contract and API behaviors pass; PostgreSQL startup initializes schema safely and persists across restarts.
4. `go run ./cmd/taskboard` starts on `127.0.0.1:8080` with no required environment variables or manual schema steps.
5. `GET /` renders real TaskBoard UI; static assets are accessible; users can create/delete projects and create/edit/move/filter/delete tasks.
6. HTTP API response shapes, validation, ordering, errors, and UTC timestamps follow this SPEC on both backends.
7. Playwright tests pass against a running app and include real browser control interactions with asserted DOM/API state changes.
8. Docker image builds without CGO, runs as non-root, serves embedded assets, and responds to health/readiness endpoints.
9. Kubernetes Deployment and Service are internally consistent, inject PostgreSQL URL from Secret, and do not deploy PostgreSQL or require a TaskBoard PVC in PostgreSQL mode.
10. Documentation covers local SQLite, external PostgreSQL, Go tests, Playwright, Docker, and k3s deployment with no hardcoded live credentials.
11. Gas Town phase ownership, dependencies, test reviews, and QA complete successfully; unavailable infrastructure verification is explicitly recorded and not falsely marked passed.

## 13. Explicitly out of scope

Accounts, authentication, authorization, teams, assignees, file uploads, comments, due dates, drag-and-drop, notifications, audit logs, analytics, real-time streams, AI/LLM features, cloud-hosted APIs, automatic image publication, registry management, Ingress/TLS provisioning, PostgreSQL installation/administration/backups, general-purpose migrations, Helm charts, autoscaling, and production-grade multi-tenant security.

**Security note:** This is a local/internal MVP with no authentication. Do not expose it to the public Internet without an appropriate external access-control layer.
