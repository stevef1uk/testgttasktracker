# Implementation plan

Active phase: phase-1-core (Go foundation, configuration, models, SQLite store).
Phase verify: `cd taskboard && go mod verify && go test ./internal/config ./internal/model ./internal/store`

Deferred to phase-1-app (not in this plan): `taskboard/internal/store/postgres.go` (bead ta-11e) and the PostgreSQL-dependent tests. Failures about `NewPostgresStore` in `postgres_test.go` belong to that phase and are out of scope here.

## Bead map

### ta-a9w: taskboard/go.mod
- Scope: Go module `taskboard` requiring `modernc.org/sqlite` and `github.com/jackc/pgx/v5`.
- Architecture: Planned file layout; SPEC fixed constraint of a pure-Go SQLite driver (no CGO).
- Interfaces: module path `taskboard`; no exported symbols.
- Depends on: <none>
- Consumed by: taskboard/go.sum, taskboard/internal/config/config.go, taskboard/internal/model/model.go, taskboard/internal/store/store.go, taskboard/internal/store/sqlite.go
- Acceptance: `go mod verify` passes; module path is `taskboard`; no CGO dependency.
- Verify: cd taskboard && go mod verify

### ta-lhb: taskboard/go.sum
- Scope: Lockfile produced by dependency resolution, not handwritten.
- Architecture: Planned file layout.
- Interfaces: none.
- Depends on: taskboard/go.mod
- Consumed by: all phase-1-core Go packages.
- Acceptance: `go mod verify` succeeds against the committed go.mod.
- Verify: cd taskboard && go mod verify

### ta-far: taskboard/README.md
- Scope: Phase-1 documentation covering Go requirements, the configuration table, and the phase-1 test command.
- Architecture: SPEC configuration section; phase-1 gate.
- Interfaces: none.
- Depends on: taskboard/internal/config/config.go
- Consumed by: <none>
- Acceptance: documents TASKBOARD_DB_DRIVER, TASKBOARD_DB_PATH, TASKBOARD_DATABASE_URL, TASKBOARD_HOST, and TASKBOARD_PORT with defaults; lists the phase-1 test command; contains no real credentials.
- Verify: cd taskboard && test -f README.md

### ta-ama: taskboard/.gitignore
- Scope: Ignore build output, SQLite databases, node_modules, Playwright reports, and editor files.
- Architecture: SPEC rule to never commit secrets, databases, binaries, node_modules, or reports.
- Interfaces: none.
- Depends on: <none>
- Consumed by: <none>
- Acceptance: ignores *.db, *.db-wal, *.db-shm, /bin/, node_modules/, playwright-report/, and test-results/.
- Verify: cd taskboard && git check-ignore -q taskboard.db

### ta-367: taskboard/internal/config/config.go
- Scope: Environment-variable configuration with validation; no database or HTTP code.
- Architecture: Module ownership table (Config struct, LoadConfig function).
- Interfaces: `type Config struct` with DBDriver, DBPath, DatabaseURL, Host, Port; `func LoadConfig() (Config, error)`.
- Depends on: taskboard/go.mod
- Consumed by: taskboard/cmd/taskboard/main.go (phase-1-app), taskboard/internal/config/config_test.go
- Acceptance: defaults are sqlite, taskboard.db, 127.0.0.1, and 8080; drivers other than sqlite or postgres are rejected; a nonempty TASKBOARD_DATABASE_URL is required when the driver is postgres; ports outside 1-65535 are rejected; error messages never echo the database URL.
- Verify: cd taskboard && go test ./internal/config

### ta-qb9: taskboard/internal/model/model.go
- Scope: Domain structs, Status and Priority types with constants, and Unicode-aware validation. No SQL and no HTTP code.
- Architecture: Module ownership table names Project struct, Task struct, Status and Priority types/constants, ValidateProject, and ValidateTask functions. The architecture gives no signatures for the validators, so the implementer defines them from the acceptance rules below and must not change any other file to fit them.
- Interfaces: `type Project struct`; `type Task struct`; `type Status`, `type Priority` with their constants; `func ValidateProject` and `func ValidateTask` (names only; signatures set by the implementer, consistent with the store callers).
- Depends on: taskboard/go.mod
- Consumed by: taskboard/internal/store/store.go, taskboard/internal/store/sqlite.go
- Acceptance: project name and task title are trimmed and limited to 1-80 and 1-120 Unicode code points (utf8.RuneCountInString); description is limited to 1000 code points and kept verbatim; unknown status or priority values are rejected; default status is todo and default priority is medium.
- Verify: cd taskboard && go test ./internal/model

### ta-e5c: taskboard/internal/store/store.go
- Scope: Shared Store interface, repository error values, and dialect-neutral DDL constants for the SQLite schema.
- Architecture: Module ownership table (Store interface, ErrNotFound, DDL schema constants).
- Interfaces: `type Store interface` (project and task CRUD and filtered listing); `var ErrNotFound`; schema constant for projects and tasks with an index on tasks.project_id.
- Depends on: taskboard/internal/model/model.go
- Consumed by: taskboard/internal/store/sqlite.go, taskboard/internal/web/app.go (phase-1-app)
- Acceptance: the interface covers every store operation in the SPEC; DDL uses ON DELETE CASCADE on tasks.project_id; the package defines no HTTP types.
- Verify: cd taskboard && go build ./internal/store

### ta-me0: taskboard/internal/store/sqlite.go
- Scope: SQLite implementation of the Store interface using modernc.org/sqlite.
- Architecture: Module ownership table (SQLiteStore struct, NewSQLiteStore); foreign keys on every pooled connection and a 5000 ms busy timeout.
- Interfaces: `type SQLiteStore struct`; `func NewSQLiteStore(path string) (*SQLiteStore, error)`; compile-time check `var _ Store = (*SQLiteStore)(nil)`.
- Depends on: taskboard/internal/store/store.go, taskboard/internal/model/model.go
- Consumed by: taskboard/internal/store/store_test.go (phase-1-app)
- Acceptance: foreign keys are enabled on every pooled connection via DSN (_pragma=foreign_keys(1)); busy timeout is 5000 ms; only `?` placeholders are used, with no interpolated user input; list queries order by ascending id; empty lists are returned as empty slices.
- Verify: cd taskboard && go test ./internal/store

## Integration contract

Not required for phase-1-core. It has no cmd/.../main entrypoint; that belongs to phase-1-app.

## Future considerations

- PostgreSQL store, its integration tests, and the server entrypoint are scheduled for phase-1-app.
