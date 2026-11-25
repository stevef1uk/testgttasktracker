# Test Plan — TaskBoard

## Requirements → tests

### phase-1-core
Requirement: Domain validation: project name trimmed to 1-80 code points; task title trimmed to 1-120; description verbatim up to 1000; status and priority enums exact.
Level: unit
Test file: taskboard/internal/model/model_test.go
Bead ID: ta-4rb
Phase: phase-1-core
Scenarios:
- Boundaries at 0, 1, 80, 81, 120, 121, 1000, 1001 runes, including multibyte runes.
- Whitespace-only name and title are rejected; description is not trimmed.
- Each valid status and priority passes; one invalid value per enum fails.
Assertions:
- Exactly 80 multibyte runes pass for names; 81 fail (utf8.RuneCountInString semantics).
- Validators return nil for valid input and an error for invalid input.

Requirement: Configuration parsing: TASKBOARD_DB_DRIVER (sqlite|postgres, default sqlite), TASKBOARD_DB_PATH (default taskboard.db), TASKBOARD_DATABASE_URL (required for postgres), TASKBOARD_HOST (default 127.0.0.1), TASKBOARD_PORT (integer 1-65535, default 8080); invalid values fail startup.
Level: unit
Test file: taskboard/internal/config/config_test.go
Bead ID: ta-gd2
Phase: phase-1-core
Scenarios:
- Defaults apply when all variables are unset.
- Invalid driver, empty postgres URL with driver postgres, and port 0, 65536, or non-numeric values are rejected.
- Valid overrides populate every Config field.
Assertions:
- LoadConfig returns an error naming the invalid variable and never echoes the database URL.
- Returned Config fields equal the expected values for valid input.

### phase-1-app
Requirement: SQLite store contract: idempotent schema creation, foreign keys enforced on every pooled connection, 5000 ms busy timeout, project and task CRUD, ascending ID order, status/priority filters combined with AND, partial PATCH, project delete cascades to tasks, persistence across close and reopen.
Level: integration
Test file: taskboard/internal/store/store_test.go
Bead ID: ta-5rq
Phase: phase-1-app
Scenarios:
- Fresh file schema is empty; second init on the same file keeps existing rows.
- Create and list projects in ascending ID order; tasks default to todo and medium with UTC RFC3339 timestamps.
- Filters by status, priority, and both combine with AND.
- Patch changes only supplied fields; deleting a project cascades to its tasks.
- Insert task with unknown project_id fails with a foreign key error.
- Close and reopen the same t.TempDir() file preserves data.
Assertions:
- Returned rows equal expected values; timestamps parse as UTC RFC3339.
- Test DB path is under t.TempDir(), never taskboard.db.
- PRAGMA foreign_keys is 1 and busy_timeout is 5000 on pooled connections.

Requirement: PostgreSQL store runs the same contract against a dedicated disposable test database when TASKBOARD_TEST_POSTGRES_URL is set; skipped otherwise.
Level: integration
Test file: taskboard/internal/store/postgres_integration_test.go
Bead ID: ta-9ik
Phase: phase-1-app
Scenarios:
- Skipped with t.Skip when TASKBOARD_TEST_POSTGRES_URL is empty.
- Concurrent schema init is safe under advisory lock.
- Cascade delete, ordering, filters, UTC timestamps, and identity IDs match SQLite.
Assertions:
- Same expected values as the SQLite contract tests.
- Error output does not include the connection URL.

Requirement: Project REST API: GET /api/health, GET /api/ready, GET and POST /api/projects, DELETE /api/projects/{project_id} with SPEC section 6 status codes and JSON shapes; GET / serves the embedded HTML shell.
Level: integration
Test file: taskboard/internal/web/api_test.go
Bead ID: ta-83w
Phase: phase-1-app
Scenarios:
- Health returns {"status":"ok"} with 200 independent of DB state.
- Ready returns {"status":"ready"} on healthy SQLite and 503 {"error":"database unavailable"} on failure.
- Project create returns 201 with JSON; blank name returns 400 with an error object.
- Project list returns [] when empty and ascending ID order otherwise; delete returns 204 with empty body, then 404 for missing.
- Storage failure returns safe 500 JSON with no SQL text; project list failure returns 500, never an empty array.
Assertions:
- Decoded JSON values, keys, and types match SPEC; whitespace is not asserted.
- Status codes and Content-Type are exact; 204 body length is 0.

### phase-2-kanban
Requirement: Task REST API (list, create, patch, delete) with server-owned status, IDs, and timestamps, nonempty PATCH, explicit-null rejection, and AND-combined filters.
Level: integration
Test file: taskboard/internal/web/app_test.go
Bead ID: ta-s3j
Phase: phase-2-kanban
Scenarios:
- Task create defaults status todo and priority medium; client-supplied status, id, and timestamps are ignored.
- PATCH with a nonempty subset updates only those fields; empty PATCH and explicit null values return 400.
- Task list filters by status, priority, and both; invalid filter values return 400.
- Missing project returns 404 on task list and create; missing task returns 404 on patch and delete.
- Over-length title (121 runes) and description (1001 runes) return 400; blank title returns 400.
- Missing static file (GET /static/missing.js) returns 404.
Assertions:
- Decoded task JSON matches expected values; updated_at changes on PATCH.
- Filtered lists contain exactly the expected IDs in ascending order.
- Rejections return 400 with a single {"error": ...} object.

Requirement: Static UI content: GET / serves real TaskBoard HTML with the TaskBoard heading, the project and task forms, and To Do, Doing, and Done columns; GET /static/app.js and GET /static/style.css return 200 with nonempty content and correct content types.
Level: integration
Test file: taskboard/internal/web/ui_content_test.go
Bead ID: ta-5s6
Phase: phase-2-kanban
Scenarios:
- Index page contains TaskBoard, the project form, the task form, and all three column headings.
- app.js and style.css each return 200 with nonempty bodies and expected content types.
Assertions:
- Response bodies contain the required literal strings; status codes and Content-Type are exact.

### phase-3-browser
Requirement: Browser E2E: real clicks, typing, and selections drive the UI; project and task creation, Move forward from To Do to Doing to Done, delete, and priority filter reset are asserted via DOM and API state.
Level: ui
Test file: taskboard/playwright/taskboard.spec.ts
Bead ID: ta-xmb
Phase: phase-3-browser
Scenarios:
- Create a uniquely named project and task through the UI; card appears in To Do.
- Click Move forward to Doing, then Done; the forward control is hidden or disabled on done.
- Delete the task; it disappears from the DOM and the API list.
- Create two tasks of different priority, select high in the filter, verify only the high task is visible, reset to all, verify both visible.
Assertions:
- Visible DOM text and card counts change after each click or selection.
- API GET confirms persisted status and priority after each action.
- Test projects are cleaned up before and after each test.
