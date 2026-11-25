# Test Plan — TaskBoard

## Requirements → tests

### phase-1-core
Requirement: Config reads TASKBOARD_DB_DRIVER, TASKBOARD_DB_PATH, TASKBOARD_DATABASE_URL, TASKBOARD_HOST, TASKBOARD_PORT with SPEC section 4 defaults; invalid driver, port, or missing PostgreSQL URL fail startup without echoing the URL.
Level: unit
Test file: taskboard/internal/config/config_test.go (conventional companion of taskboard/internal/config/config.go)
Bead ID: ta-367
Phase: phase-1-core
Scenarios:
- No env set yields sqlite, taskboard.db, 127.0.0.1, 8080, empty URL.
- Driver accepts exactly sqlite or postgres; any other value errors naming TASKBOARD_DB_DRIVER.
- Port 0, 65536, and non-numeric values error naming TASKBOARD_PORT; 1 and 65535 pass.
- postgres driver with empty TASKBOARD_DATABASE_URL errors with no SQLite fallback.
Assertions:
- Config fields equal expected values (t.Setenv per case).
- Error text does not contain the URL or password substring.

Requirement: Domain validation: project name trimmed to 1-80 code points; task title trimmed to 1-120; description verbatim up to 1000; status and priority enums exact.
Level: unit
Test file: taskboard/internal/model/model_test.go (conventional companion of taskboard/internal/model/model.go)
Bead ID: ta-qb9
Phase: phase-1-core
Scenarios:
- Boundaries at 0, 1, 80, 81, 120, 121, 1000, 1001 runes, including multibyte runes.
- Whitespace-only name and title are rejected; description is not trimmed.
- Each valid status and priority passes; one invalid value per enum fails.
Assertions:
- Exactly 80 multibyte runes pass for names; 81 fail (utf8.RuneCountInString semantics).
- Validators return nil for valid input and an error for invalid input.

### phase-1-app
Requirement: SQLite store contract: idempotent schema creation, foreign keys enforced on every pooled connection, 5000 ms busy timeout, project and task CRUD, ascending ID order, status/priority filters combined with AND, partial PATCH, project delete cascades to tasks, persistence across close and reopen.
Level: integration
Test file: taskboard/internal/store/store_test.go
Bead ID: ta-5rq
Phase: phase-1-app
Scenarios:
- Fresh file schema is empty; second init on the same file keeps existing rows and runs no DROP or reset.
- Create and list projects in ascending ID order; tasks default to todo and medium with UTC RFC3339 timestamps.
- Filters by status, priority, and both combine with AND; list order ascending ID.
- Patch changes only supplied fields; deleting a project cascades to its tasks.
- Insert task with unknown project_id fails with a foreign key error.
- Close and reopen the same t.TempDir() file preserves data.
- Unicode code-point boundaries; SQL-injection-style names stored as literal text.
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
- Test objects use unique names or an isolated schema and clean up only their own data.
Assertions:
- Same expected values as the SQLite contract tests.
- Error output does not include the connection URL.

Requirement: Project REST API: GET /api/health, GET /api/ready, GET and POST /api/projects, DELETE /api/projects/{project_id} with SPEC section 6 status codes and JSON shapes; GET / serves the embedded HTML shell.
Level: integration
Test file: taskboard/internal/web/api_test.go
Bead ID: ta-or0
Phase: phase-1-app
Scenarios:
- Health returns {"status":"ok"} with 200 independent of DB state.
- Ready returns {"status":"ready"} on healthy SQLite and 503 {"error":"database unavailable"} on failure.
- Project create returns 201 with JSON; blank name returns 400 with an error object.
- Project list returns [] when empty and ascending ID order otherwise; delete returns 204 with empty body, then 404 for missing.
- Malformed JSON and invalid IDs return documented 400 or 404.
- GET / returns 200 text/html containing the TaskBoard heading and the project form; Kanban column headers are asserted in phase-2-kanban (ui_content_test.go).
- Storage failure returns safe 500 JSON with no SQL text; project list failure returns 500, never an empty array.
Assertions:
- Decoded JSON values, keys, and types match SPEC; whitespace is not asserted.
- Status codes and Content-Type are exact; 204 body length is 0.

### phase-2-kanban
Requirement: Task REST API (list, create, patch, delete) with server-owned status, IDs, and timestamps, nonempty PATCH, explicit-null rejection, and AND-combined filters.
Level: integration
Test file: taskboard/internal/web/app_test.go
Bead ID: pending (phase-2-kanban bead not yet created; Planner to assign)
Phase: phase-2-kanban
Scenarios:
- Task create defaults status todo and priority medium; client-supplied status, id, and timestamps are ignored.
- PATCH with a nonempty subset updates only those fields; empty PATCH and explicit null values return 400.
- Task list filters by status, priority, and both; invalid filter values return 400.
- Missing project returns 404 on task list and create; missing task returns 404 on patch and delete.
- Over-length title (121 runes) and description (1001 runes) return 400 on create and PATCH; blank title returns 400.
- Task storage failure returns safe generic 500 JSON with no SQL or driver text.
- Missing static file (GET /static/missing.js) returns 404.
Assertions:
- Decoded task JSON matches expected values; updated_at changes on PATCH.
- Filtered lists contain exactly the expected IDs in ascending order.
- Rejections return 400 with a single {"error": ...} object.

Requirement: Rendered HTML contains the TaskBoard heading, project and task forms, and To Do, Doing, Done headings; static assets are served and nonempty.
Level: integration
Test file: taskboard/internal/web/ui_content_test.go
Bead ID: pending (phase-2-kanban bead not yet created; Planner to assign)
Phase: phase-2-kanban
Scenarios:
- Root page contains TaskBoard, the project input and Add project button, the task form with Add task, and the three column headings.
- Status filter options are exactly all, todo, doing, done; priority filter options are exactly all, low, medium, high.
- Root HTML references /static/app.js and /static/style.css.
- Static app.js and style.css return 200 with nonempty bodies.
Assertions:
- Status 200 and Content-Type text/html for the root page; expected substrings present.
- Static bodies have nonzero length and expected content types.
- Asset references in the HTML match the served /static paths exactly.

### phase-3-browser
Requirement: Browser E2E: real clicks, typing, and selections drive the UI; project and task creation, Move forward from To Do to Doing to Done, delete, and priority filter reset are asserted via DOM and API state.
Level: ui
Test file: taskboard/playwright/taskboard.spec.ts
Bead ID: pending (phase-3-browser bead not yet created; Planner to assign)
Phase: phase-3-browser
Scenarios:
- Initial page shows TaskBoard and three columns.
- Create a uniquely named project and task through the UI; card appears in To Do.
- Click Move forward to Doing, then Done; the forward control is hidden or disabled on done.
- Delete the task; it disappears from the DOM and the API list.
- Create two tasks of different priority, select high in the filter, verify only the high task is visible, reset to all, verify both visible.
Assertions:
- Visible DOM text and card counts change after each click or selection.
- API GET confirms persisted status and priority after each action.
- Test projects are cleaned up before and after each test.
