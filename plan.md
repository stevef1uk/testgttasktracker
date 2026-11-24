# Implementation plan — phase-1-core (rework amendment)

Active phase: phase-1-core (Go foundation, configuration, models, SQLite store).
Layout root: taskboard/. All paths are repository-relative and match required_files exactly.

## Review response

QA found that the test repairs for ta-zmd, ta-93p, and ta-i6d were not on disk. The test files are named here as acceptance for their owning beads. They are not new required_files, and no new beads are created.

- ta-zmd (taskboard/internal/store/sqlite.go): acceptance is verified by taskboard/internal/store/sqlite_test.go. That file must hold project and task CRUD tests, idempotent reinit, status/priority filter tests (including the combined AND case), cascade delete, UTC timestamp format, and close/reopen persistence. Keep the four existing tests (Ping, schema tables, foreign_keys, busy_timeout).
- ta-93p (taskboard/internal/config/config.go): acceptance is verified by taskboard/internal/config/config_test.go. The postgres-with-empty-URL error must not echo any URL value. config_test.go must cover defaults (sqlite, 127.0.0.1, 8080) and port boundaries 0, 1, 65535, and 65536.
- ta-i6d (taskboard/internal/model/model.go): acceptance is verified by taskboard/internal/model/model_test.go. It must cover the 80/81 name boundary, the 120/121 title boundary, the 1000-code-point description limit, status and priority enum acceptance and rejection, and multi-byte code-point counting.

If the active phase must include these test files as required_files, report a plan_gap with evidence. Do not silently add them as extra implement beads.

## Bead map

### ta-92w: taskboard/go.mod
- Scope: module taskboard, go 1.26.1, modernc.org/sqlite dependency.
- Architecture: pure-Go SQLite driver, CGO disabled.
- Interfaces: none (manifest).
- Depends on: <none>
- Consumed by: ta-451, ta-93p, ta-i6d, ta-xea, ta-zmd.
- Acceptance: module path is taskboard; requires modernc.org/sqlite; no CGO-only dependency.
- Verify: cd taskboard && go mod verify

### ta-451: taskboard/go.sum
- Scope: checksums from dependency resolution, not handwritten.
- Depends on: taskboard/go.mod (ta-92w).
- Consumed by: all Go packages.
- Acceptance: go mod verify succeeds.
- Verify: cd taskboard && go mod verify

### ta-f1f: taskboard/README.md
- Scope: configuration table for the five TASKBOARD_* variables, phase-1 test commands, package layout.
- Acceptance: documents default port 8080 and the postgres URL requirement; no real credentials.
- Verify: grep -q 'TASKBOARD_PORT' taskboard/README.md

### ta-b55: taskboard/.gitignore
- Scope: ignore binaries, *.db files, node_modules, Playwright reports.
- Acceptance: *.db, the taskboard binary, and node_modules are ignored.
- Verify: cd taskboard && git check-ignore -q taskboard.db

### ta-93p: taskboard/internal/config/config.go
- Scope: Config struct and LoadConfig reading TASKBOARD_DB_DRIVER, TASKBOARD_DB_PATH, TASKBOARD_DATABASE_URL, TASKBOARD_HOST, TASKBOARD_PORT.
- Architecture: SPEC section 4.
- Interfaces: type Config struct; func LoadConfig() (Config, error).
- Depends on: taskboard/go.mod (ta-92w).
- Consumed by: taskboard/internal/store/sqlite.go, taskboard/cmd/taskboard/main.go (later phase).
- Acceptance:
  - driver defaults to sqlite; only sqlite or postgres are accepted.
  - postgres with empty TASKBOARD_DATABASE_URL errors without echoing any URL.
  - port defaults to 8080; 0, 65536, and non-integers are rejected; 1 and 65535 are accepted.
  - host defaults to 127.0.0.1.
- Test file (acceptance evidence): taskboard/internal/config/config_test.go.
- Verify: cd taskboard && go test ./internal/config

### ta-i6d: taskboard/internal/model/model.go
- Scope: Project and Task structs; MaxProjectNameLen = 80; MaxTaskTitleLen = 120; MaxDescriptionLen = 1000; ValidateProjectName, ValidateTaskTitle, ValidateDescription, ValidateStatus, ValidatePriority.
- Architecture: SPEC section 5; lengths counted with utf8.RuneCountInString.
- Interfaces: the constants and validators above.
- Depends on: taskboard/go.mod (ta-92w).
- Consumed by: taskboard/internal/store/sqlite.go, taskboard/internal/store/store.go.
- Acceptance:
  - names are trimmed and limited to 80 code points; titles are trimmed and limited to 120.
  - 80 passes and 81 fails for names; 120 passes and 121 fails for titles.
  - descriptions are not trimmed and are limited to 1000 code points.
  - status accepts only todo, doing, done; priority accepts only low, medium, high.
  - multi-byte characters are counted as code points.
- Test file (acceptance evidence): taskboard/internal/model/model_test.go.
- Verify: cd taskboard && go test ./internal/model

### ta-xea: taskboard/internal/store/store.go
- Scope: Store interface with context-first methods (Ping, Close, project and task CRUD, filters) and SchemaSQLite DDL.
- Depends on: taskboard/internal/model/model.go (ta-i6d).
- Consumed by: taskboard/internal/store/sqlite.go, taskboard/internal/web/api.go (later phase).
- Acceptance: every method takes context.Context first; the interface includes ListProjects, CreateProject, DeleteProject, ListTasks (status and priority filters), CreateTask, UpdateTask, DeleteTask.
- Verify: cd taskboard && go vet ./internal/store

### ta-zmd: taskboard/internal/store/sqlite.go
- Scope: NewSQLiteStore; foreign keys enabled per connection via DSN; busy_timeout 5000 ms; project and task CRUD; status and priority filters combined with AND; cascade delete.
- Depends on: taskboard/internal/store/store.go (ta-xea), taskboard/internal/model/model.go (ta-i6d), taskboard/internal/config/config.go (ta-93p).
- Consumed by: taskboard/internal/web/api.go (later phase).
- Acceptance:
  - schema creation is idempotent; a fresh database has no rows.
  - projects list in ascending id order; create returns a positive id.
  - tasks default to status todo and priority medium; timestamps are UTC RFC3339.
  - filters by status, by priority, and by both return only matching tasks.
  - updates change only the supplied fields.
  - deleting a project removes its tasks; a task with a missing project_id fails.
  - user input is passed as parameters only.
- Test file (acceptance evidence): taskboard/internal/store/sqlite_test.go.
- Verify: cd taskboard && go test ./internal/store

## Future considerations

- Phase 1-app (PostgreSQL store, HTTP API, main entrypoint) is owned by later beads and is not in this phase.

## Phase verification

cd taskboard && go mod verify && go test ./internal/config ./internal/model ./internal/store
