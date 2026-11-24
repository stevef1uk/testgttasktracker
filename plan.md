# Implementation plan — phase-1-core (rework)

Active phase: phase-1-core (Go foundation, configuration, models, SQLite store).
Layout root: taskboard/. All paths below are repository-relative and match required_files exactly.

## Rework scope from test_review

- ta-zmd (taskboard/internal/store/sqlite.go): add the project and task CRUD methods, the status/priority filters, and the cascade delete. Add the missing methods to store.go, and cover them in sqlite tests.
- ta-i6d (taskboard/internal/model/model.go): MaxProjectNameLen must be 80 and MaxTaskTitleLen must be 120 (the cached file has 100 and 200, which is wrong). Add ValidateDescription with a 1000 code-point limit, and add status and priority enum validators. Add boundary tests.
- ta-93p (taskboard/internal/config/config.go): when TASKBOARD_DB_DRIVER=postgres and TASKBOARD_DATABASE_URL is empty, return an error that does not echo any URL. Add tests for the default port 8080 and for port range validation.

## Bead map

### ta-92w: taskboard/go.mod
- Scope: module taskboard, go 1.26.1, and the modernc.org/sqlite dependency.
- Architecture: Go foundation; SQLite driver is pure Go (CGO disabled).
- Interfaces: none (manifest).
- Depends on: <none>
- Consumed by: ta-451 (taskboard/go.sum), ta-93p, ta-i6d, ta-xea, ta-zmd.
- Acceptance: module path is taskboard; requires modernc.org/sqlite; no CGO-only dependencies.
- Verify: cd taskboard && go mod verify

### ta-451: taskboard/go.sum
- Scope: dependency checksums produced by go mod tidy; not handwritten.
- Architecture: committed lockfile.
- Interfaces: none.
- Depends on: taskboard/go.mod (ta-92w).
- Consumed by: all Go packages.
- Acceptance: go mod verify succeeds.
- Verify: cd taskboard && go mod verify

### ta-f1f: taskboard/README.md
- Scope: configuration table (five TASKBOARD_* variables), Go test commands for phase 1, and package layout.
- Architecture: documentation deliverable for phase-1-core.
- Interfaces: none.
- Depends on: <none>
- Consumed by: none.
- Acceptance: documents the default port 8080 and the postgres URL requirement; no real credentials.
- Verify: grep -q 'TASKBOARD_PORT' taskboard/README.md

### ta-b55: taskboard/.gitignore
- Scope: ignore binaries, *.db files, node_modules, and Playwright reports.
- Architecture: SPEC section 2 forbids committing secrets, databases, and binaries.
- Interfaces: none.
- Depends on: <none>
- Consumed by: none.
- Acceptance: *.db, taskboard binary, and node_modules are ignored.
- Verify: cd taskboard && git check-ignore -q taskboard.db

### ta-93p: taskboard/internal/config/config.go
- Scope: Config struct and LoadConfig reading TASKBOARD_DB_DRIVER, TASKBOARD_DB_PATH, TASKBOARD_DATABASE_URL, TASKBOARD_HOST, and TASKBOARD_PORT.
- Architecture: SPEC section 4 configuration table.
- Interfaces: type Config struct; func LoadConfig() (Config, error).
- Depends on: taskboard/go.mod (ta-92w).
- Consumed by: taskboard/internal/store/sqlite.go (DB path), taskboard/cmd/taskboard/main.go (later phase).
- Acceptance:
  - driver defaults to sqlite; only sqlite or postgres are accepted.
  - postgres with empty TASKBOARD_DATABASE_URL returns an error. The error text must not contain the URL value.
  - port defaults to 8080 when unset; values outside 1–65535 or non-integer values are rejected.
  - host defaults to 127.0.0.1.
- Verify: cd taskboard && go test ./internal/config

### ta-i6d: taskboard/internal/model/model.go
- Scope: Project and Task structs, MaxProjectNameLen = 80, MaxTaskTitleLen = 120, MaxDescriptionLen = 1000, ValidateProjectName, ValidateTaskTitle, ValidateDescription, ValidateStatus, ValidatePriority.
- Architecture: SPEC section 5 field rules; lengths counted with utf8.RuneCountInString.
- Interfaces: constants MaxProjectNameLen, MaxTaskTitleLen, MaxDescriptionLen; the validators above.
- Depends on: taskboard/go.mod (ta-92w).
- Consumed by: taskboard/internal/store/sqlite.go, taskboard/internal/store/store.go.
- Acceptance:
  - project names are trimmed and limited to 80 code points; titles are trimmed and limited to 120.
  - a 80-code-point name passes and an 81-code-point name fails; the same boundary applies to titles at 120/121.
  - descriptions are not trimmed and are limited to 1000 code points.
  - status accepts only todo, doing, done; priority accepts only low, medium, high.
  - multi-byte characters are counted as code points, not bytes.
- Verify: cd taskboard && go test ./internal/model

### ta-xea: taskboard/internal/store/store.go
- Scope: Store interface with context-first methods (Ping, Close, project and task CRUD, filters) and the SchemaSQLite DDL constant.
- Architecture: shared repository contract for SQLite and PostgreSQL.
- Interfaces: type Store interface; const SchemaSQLite.
- Depends on: taskboard/internal/model/model.go (ta-i6d).
- Consumed by: taskboard/internal/store/sqlite.go, taskboard/internal/web/api.go (later phase).
- Acceptance: every method takes context.Context as its first argument; the interface includes ListProjects, CreateProject, DeleteProject, ListTasks (with status and priority filters), CreateTask, UpdateTask, DeleteTask.
- Verify: cd taskboard && go vet ./internal/store

### ta-zmd: taskboard/internal/store/sqlite.go
- Scope: NewSQLiteStore; foreign keys enabled per connection through the DSN; busy_timeout 5000 ms; full project and task CRUD; status and priority filters combined with AND; cascade delete of tasks when a project is deleted.
- Architecture: SPEC section 5 and section 8 store tests.
- Interfaces: func NewSQLiteStore(path string) (*SQLiteStore, error) and the Store methods.
- Depends on: taskboard/internal/store/store.go (ta-xea), taskboard/internal/model/model.go (ta-i6d), taskboard/internal/config/config.go (ta-93p).
- Consumed by: taskboard/internal/store/store_test.go (later phase), taskboard/internal/web/api.go (later phase).
- Acceptance:
  - schema creation is idempotent; a fresh database has no rows.
  - projects list in ascending id order; creating a project returns a positive id.
  - tasks get status todo and priority medium by default; created_at and updated_at are UTC RFC3339.
  - a filter by status, by priority, and by both returns only matching tasks.
  - PATCH-style updates change only the supplied fields.
  - deleting a project removes its tasks; a task insert with a missing project_id fails.
  - user input is passed as parameters; no string interpolation into SQL.
- Verify: cd taskboard && go test ./internal/store

## Future considerations

- Phase 1b (PostgreSQL store, HTTP API, main entrypoint) is owned by later beads and is not in this phase.

## Phase verification

cd taskboard && go mod verify && go test ./internal/config ./internal/model ./internal/store
