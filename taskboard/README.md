# TaskBoard

TaskBoard is a small Go task management web application. Users create and delete projects, create and edit tasks, move tasks through Kanban states, filter by status and priority, and delete tasks.

Data persists in SQLite or PostgreSQL, selected at runtime without changing application code.

## Requirements

- Go 1.25+
- No CGO required: the SQLite driver is `modernc.org/sqlite` (pure Go).
- PostgreSQL is optional; the driver is `github.com/jackc/pgx/v5/stdlib`.

## Configuration

Settings are read from environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `TASKBOARD_DB_DRIVER` | `sqlite` | Exactly `sqlite` or `postgres` |
| `TASKBOARD_DB_PATH` | `taskboard.db` | SQLite database file path |
| `TASKBOARD_DATABASE_URL` | _(empty)_ | PostgreSQL connection URL, used when the driver is `postgres` |
| `TASKBOARD_HOST` | `127.0.0.1` | Interface the HTTP server binds to |
| `TASKBOARD_PORT` | `8080` | HTTP server port (1-65535) |

## Development

Run the test suite for the phase-1 packages:

go mod verify
go test ./internal/config ./internal/model ./internal/store

## Layout

- `internal/config`: environment-variable configuration loading and validation.
- `internal/model`: domain models and Unicode code-point length validation.
- `internal/store`: shared `Store` interface, DDL constants, and the SQLite implementation.