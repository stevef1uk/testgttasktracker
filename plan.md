# Implementation plan

Active delivery phase: `phase-1-core` (Go foundation, configuration, models, SQLite store).
Phase verification: `cd taskboard && go mod verify && go test ./internal/config ./internal/model ./internal/store`
Phase-1-core owns only the 8 paths in `required_files`. It creates no test files. Packages without tests must still compile under the phase gate. Repository and behavior tests belong to `phase-1-app`. No HTTP routes, PostgreSQL code, or entrypoint belong to this phase.

## Bead map

### ta-92w: taskboard/go.mod
- Scope: Go module manifest for the sole module `taskboard`, with a `go` directive pinning Go 1.26.1 (satisfies SPEC's Go >=1.25 and the committed pinned-toolchain requirement).
- Architecture: `taskboard/go.mod` in Planned file layout; SPEC §2 requires a committed pinned toolchain and committed go.mod/go.sum.
- Interfaces: module path `taskboard`; direct requirement `modernc.org/sqlite v1.63.0`.
- Depends on: `<none>`
- Consumed by: ta-451, ta-93p, ta-i6d, ta-xea, ta-zmd, ta-f1f, ta-b55
- Acceptance:
  - `go.mod` declares `module taskboard` and `go 1.26.1` (1.25 or newer).
  - No separate `toolchain` line is required; the `go` directive is the pin.
  - `modernc.org/sqlite` is the only SQLite driver requirement; builds run with `CGO_ENABLED=0`.
  - No PostgreSQL driver (`pgx`) is required in phase-1-core.
- Verify: `cd taskboard && go mod tidy && go mod verify`

### ta-451: taskboard/go.sum
- Scope: Checksum lockfile produced by dependency resolution for the `taskboard` module.
- Architecture: `taskboard/go.sum` in Planned file layout; SPEC §1 says go.sum must be generated, not handwritten.
- Interfaces: none (lockfile).
- Depends on: ta-92w
- Consumed by: ta-zmd, ta-93p, ta-i6d, ta-xea
- Acceptance:
  - `go.sum` is produced by `go mod tidy`; no hand-edited checksum lines.
  - Every module in `go.mod` has matching `h1:` entries; `go mod verify` reports all modules verified.
  - No PostgreSQL driver entries (`pgx`) appear in phase-1-core.
- Verify: `cd taskboard && go mod tidy && go mod verify`

### ta-f1f: taskboard/README.md
- Scope: Developer documentation for phase-1-core: module layout, configuration variables, and the local test gate.
- Architecture: `taskboard/README.md` in Planned file layout; SPEC §4 lists the configuration table.
- Interfaces: documents `TASKBOARD_DB_DRIVER`, `TASKBOARD_DB_PATH`, `TASKBOARD_DATABASE_URL`, `TASKBOARD_HOST`, `TASKBOARD_PORT` with defaults `sqlite`, `taskboard.db`, unset, `127.0.0.1`, `8080`.
- Depends on: ta-92w
- Consumed by: none
- Acceptance:
  - README lists every configuration variable with its default and behavior from SPEC §4.
  - README states the PostgreSQL URL must never be printed in logs or errors.
  - README gives the phase gate `cd taskboard && go mod verify && go test ./internal/config ./internal/model ./internal/store`.
  - README contains no real credentials.
- Verify: `cd taskboard && go mod verify && go test ./internal/config ./internal/model ./internal/store`

### ta-b55: taskboard/.gitignore
- Scope: Ignore rules for local databases, binaries, and Node/Playwright artifacts.
- Architecture: `taskboard/.gitignore` in Planned file layout; SPEC §2 forbids committing secrets, database files, binaries, node_modules, and reports.
- Interfaces: none.
- Depends on: ta-92w
- Consumed by: none
- Acceptance:
  - Ignores `*.db`, `*.db-journal`, `*.db-wal`, `*.db-shm`, and the default `taskboard.db`.
  - Ignores compiled binaries, `node_modules/`, `playwright-report/`, and `test-results/`.
  - Does not ignore `go.mod`, `go.sum`, or `package-lock.json`.
- Verify: `cd taskboard && go mod verify && go test ./internal/config ./internal/model ./internal/store`

### ta-93p: taskboard/internal/config/config.go
- Scope: Parse and validate the five TASKBOARD_* environment variables into a Config struct once at startup.
- Architecture: `LoadConfig` owned by `taskboard/internal/config/config.go`; SPEC §4 defines defaults and validation.
- Interfaces: `type Config struct` with driver, SQLite path, database URL, host, and port fields; `func LoadConfig() (Config, error)`.
- Depends on: ta-92w, ta-451
- Consumed by: ta-zmd (driver and DB path)
- Acceptance:
  - With no environment set, defaults are driver `sqlite`, DB path `taskboard.db`, host `127.0.0.1`, port `8080`, and empty database URL.
  - `TASKBOARD_DB_DRIVER` accepts exactly `sqlite` or `postgres`; any other value returns an error.
  - `TASKBOARD_PORT` must be an integer 1–65535; `0`, `65536`, and non-numeric values return an error.
  - `postgres` driver with an empty `TASKBOARD_DATABASE_URL` returns an error; no silent SQLite fallback.
  - Error messages never include the full database URL or password text.
  - The package compiles under the phase gate; this bead adds no test files.
- Verify: `cd taskboard && go test -timeout 30s -count=1 ./internal/config/...`

### ta-i6d: taskboard/internal/model/model.go
- Scope: Project and Task domain structs plus pure validation and enum checks.
- Architecture: `Project` and `Task` structs and validation functions owned by `taskboard/internal/model/model.go`; SPEC §5 defines field rules.
- Interfaces: `type Project struct` (ID, Name, CreatedAt); `type Task struct` (ID, ProjectID, Title, Description, Status, Priority, CreatedAt, UpdatedAt); validators for name, title, description, status, priority.
- Depends on: ta-92w
- Consumed by: ta-xea (Store signatures), ta-zmd (row mapping)
- Acceptance:
  - Project name is trimmed, then must be 1–80 code points; empty or whitespace-only names are rejected.
  - Task title is trimmed, then must be 1–120 code points.
  - Task description is retained verbatim and must be at most 1000 code points.
  - Length uses `utf8.RuneCountInString`; a multibyte name of exactly 80 runes passes and 81 fails.
  - Status accepts only `todo`, `doing`, `done`; priority accepts only `low`, `medium`, `high`.
  - The package compiles under the phase gate; this bead adds no test files.
- Verify: `cd taskboard && go test -timeout 30s -count=1 ./internal/model/...`

### ta-xea: taskboard/internal/store/store.go
- Scope: Shared `Store` repository interface, repository error constants, and dialect-neutral DDL for `projects` and `tasks`.
- Architecture: `taskboard/internal/store/store.go` owns the Store interface, repository errors, and DDL constants; no HTTP routing logic.
- Interfaces: `type Store interface` with context-aware methods to list, create, and delete projects, and list, create, patch, and delete tasks with status and priority filters; exported error constants for not-found and validation failures; DDL constants for both tables and the `tasks.project_id` index.
- Depends on: ta-92w, ta-451, ta-i6d
- Consumed by: ta-zmd (implements Store)
- Acceptance:
  - `Store` is the only repository contract; no SQLite-specific types appear in it.
  - DDL defines `projects(id, name, created_at)` and `tasks(id, project_id, title, description, status, priority, created_at, updated_at)` with `project_id REFERENCES projects(id) ON DELETE CASCADE`.
  - DDL includes an index on `tasks(project_id)`.
  - Error constants are distinct so callers can map not-found to 404 and validation to 400.
  - The package compiles under the phase gate; this bead adds no test files and no test-only database modes.
- Verify: `cd taskboard && go mod tidy && go test -timeout 30s -count=1 ./internal/store/...`

### ta-zmd: taskboard/internal/store/sqlite.go
- Scope: SQLite implementation of the Store interface using `modernc.org/sqlite`.
- Architecture: `NewSQLiteStore` owned by `taskboard/internal/store/sqlite.go`; SQLite PRAGMA and busy timeout live here only.
- Interfaces: `func NewSQLiteStore(path string) (*SQLiteStore, error)` returning a value satisfying `Store`; schema is created from the store.go DDL on construction.
- Depends on: ta-92w, ta-451, ta-i6d, ta-xea, ta-93p
- Consumed by: none in phase-1-core
- Acceptance:
  - Foreign keys are enabled on every pooled connection through the DSN (for example `_pragma=foreign_keys(1)`), not a single PRAGMA on one connection.
  - Busy timeout is 5000 ms through the DSN.
  - All SQL uses `?` placeholders; no user input is concatenated into SQL.
  - Schema creation is idempotent: running it twice on the same file succeeds and keeps existing rows.
  - Deleting a project removes its tasks (cascade); inserting a task with an unknown `project_id` fails with a foreign key error.
  - Filters by status, priority, and both together combine with AND; list order is ascending `id`.
  - Timestamps are stored as UTC ISO-8601 strings.
  - The package compiles under the phase gate; behavioral tests for these rules belong to `phase-1-app` (`taskboard/internal/store/store_test.go`), and this bead adds no test files.
- Verify: `cd taskboard && go mod tidy && go test -timeout 30s -count=1 ./internal/store/...`

## Integration contract

Not required this phase. The active phase `required_files` contain no entrypoint and no HTTP routes; those belong to a later phase.

## Future considerations

- PostgreSQL store, advisory-lock schema setup, and integration tests belong to `phase-1-app`.
- Repository tests (`taskboard/internal/store/store_test.go`) and their SQLite fixtures are owned by `phase-1-app`, not this phase.
