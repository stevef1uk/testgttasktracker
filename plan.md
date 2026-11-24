# Implementation plan

Active phase: `phase-1-core` (Go foundation, configuration, models, SQLite store). Only the 8 `required_files` below are in scope. Later phases (web, Postgres, k8s, Playwright) get their own beads.

## Bead map

### ta-a9w: taskboard/go.mod
- Scope: Module manifest `module taskboard`, Go 1.25+ directive, requires `modernc.org/sqlite` (pure Go, no CGO) and `github.com/jackc/pgx/v5` (stdlib driver for later phase).
- Architecture: Planned file layout; Overview (SQLite via modernc.org/sqlite, PostgreSQL via pgx/v5/stdlib).
- Interfaces: none (manifest).
- Depends on: `<none>`
- Consumed by: taskboard/go.sum, taskboard/internal/store/sqlite.go
- Acceptance:
  - `go mod verify` succeeds once go.sum is present.
  - No `cgo` requirement: the build of `./internal/store/...` succeeds with `CGO_ENABLED=0`.
  - Module path is exactly `taskboard`, so imports resolve as `taskboard/internal/...`.
- Verify: `cd taskboard && go mod verify`

### ta-lhb: taskboard/go.sum
- Scope: Checksums for every module required by taskboard/go.mod.
- Architecture: Planned file layout.
- Interfaces: none (checksum file).
- Depends on: taskboard/go.mod
- Consumed by: all Go packages under taskboard/internal.
- Acceptance:
  - `go mod verify` reports all modules verified.
  - Checksums match the versions pinned in go.mod (no stale entries).
- Verify: `cd taskboard && go mod verify`

### ta-far: taskboard/README.md
- Scope: Documents the phase-1 configuration table (`TASKBOARD_DB_DRIVER` default `sqlite`, `TASKBOARD_DB_PATH` default `taskboard.db`, `TASKBOARD_DATABASE_URL`, `TASKBOARD_HOST` default `127.0.0.1`, `TASKBOARD_PORT` default `8080`) and the test command.
- Architecture: Overview; SPEC configuration section.
- Interfaces: none (documentation).
- Depends on: taskboard/internal/config/config.go (variable names and defaults must match)
- Consumed by: none.
- Acceptance:
  - Every environment variable named in README exists in config.go with the same default.
  - Test command listed matches the phase verification command.
- Verify: `cd taskboard && go mod verify && go test ./internal/config ./internal/model ./internal/store`

### ta-ama: taskboard/.gitignore
- Scope: Ignore build output (`/bin/`, `/taskboard`), local SQLite files (`*.db`, `*.db-journal`, `*.db-wal`, `*.db-shm`), Node/Playwright artifacts, and editor files.
- Architecture: Planned file layout.
- Interfaces: none.
- Depends on: `<none>`
- Consumed by: none.
- Acceptance:
  - `git check-ignore taskboard/taskboard.db` reports the file as ignored.
  - Source files under `taskboard/internal/` are not ignored.
- Verify: `cd taskboard && go mod verify && go test ./internal/config ./internal/model ./internal/store`

### ta-367: taskboard/internal/config/config.go
- Scope: Loads TASKBOARD_* environment variables and validates them.
- Architecture: `taskboard/internal/config/config.go` parses configuration from environment variables; Config struct and LoadConfig function.
- Interfaces: `type Config struct { DBDriver, DBPath, DatabaseURL, Host string; Port int }`; `func LoadConfig() (Config, error)`.
- Depends on: `<none>`
- Consumed by: taskboard/internal/store/sqlite.go (DB path and driver), taskboard/cmd/taskboard/main.go (later phase).
- Acceptance:
  - Defaults: driver `sqlite`, DB path `taskboard.db`, host `127.0.0.1`, port `8080`.
  - Driver other than `sqlite` or `postgres` returns an error naming `TASKBOARD_DB_DRIVER`.
  - Driver `postgres` with empty `TASKBOARD_DATABASE_URL` returns an error that does not echo any URL value.
  - Port outside 1-65535 or non-numeric returns an error naming `TASKBOARD_PORT`.
- Verify: `cd taskboard && go mod tidy && go test -timeout 30s -count=1 ./internal/config/...`

### ta-qb9: taskboard/internal/model/model.go
- Scope: Project and Task structs, Unicode code-point length validation, and status and priority enum validation.
- Architecture: `taskboard/internal/model/model.go` defines Project and Task structs and Unicode length and enum validation.
- Interfaces: Project and Task structs as defined in architecture; validation helpers as named there.
- Depends on: `<none>`
- Consumed by: taskboard/internal/store/store.go, taskboard/internal/store/sqlite.go.
- Acceptance:
  - Length checks count Unicode code points, not bytes (multi-byte names of the exact limit are accepted; one code point over is rejected).
  - Status values are limited to the Kanban states in SPEC; any other value is rejected.
  - Priority values are limited to the SPEC set; any other value is rejected.
- Verify: `cd taskboard && go mod tidy && go test -timeout 30s -count=1 ./internal/model/...`

### ta-f1g: taskboard/internal/store/store.go
- Scope: Shared `Store` interface, repository error constants, and SQL schema DDL constants.
- Architecture: `taskboard/internal/store/store.go` defines the shared Store interface, repository errors, and DDL statements.
- Interfaces: `Store` interface with `context.Context`-taking List, Create, Delete (and task methods as in architecture); repository error sentinels; DDL constants consumed by `InitSchema`.
- Depends on: taskboard/internal/model/model.go
- Consumed by: taskboard/internal/store/sqlite.go, taskboard/internal/store/store_test.go (later phase).
- Acceptance:
  - Interface method signatures match SPEC (context-first).
  - Error sentinels are exported and usable with `errors.Is`.
  - DDL includes foreign-key cascade from tasks to projects.
- Verify: `cd taskboard && go mod tidy && go build ./internal/store/...`

### ta-me0: taskboard/internal/store/sqlite.go
- Scope: SQLite implementation of Store with `PRAGMA foreign_keys=ON` and a 5000ms busy timeout; `InitSchema` applies the DDL.
- Architecture: `taskboard/internal/store/sqlite.go` implements SQLite storage with foreign key PRAGMA enforcement and 5000ms busy timeout.
- Interfaces: constructor opening a DB path from config; methods satisfying the Store interface; `InitSchema`.
- Depends on: taskboard/go.mod, taskboard/internal/config/config.go, taskboard/internal/model/model.go, taskboard/internal/store/store.go
- Consumed by: later phase web layer and cmd entrypoint.
- Acceptance:
  - Opening a store applies `foreign_keys=ON` and busy timeout 5000ms (verified by PRAGMA query).
  - Deleting a project cascades to its tasks.
  - Tests use `:memory:` plus `InitSchema`, not a shared file.
  - Task create/list/update/delete round-trip with UTC timestamps.
- Verify: `cd taskboard && go mod tidy && go test -timeout 30s -count=1 ./internal/store/...`

## Phase context

Phase verification: `cd taskboard && go mod verify && go test ./internal/config ./internal/model ./internal/store`. Do not implement any path outside the 8 above in this phase.
