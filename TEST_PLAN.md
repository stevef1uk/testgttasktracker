# Test Plan — TaskBoard

## Requirements → tests

### phase-1-core
Requirement: Config reads TASKBOARD_DB_DRIVER, TASKBOARD_DB_PATH, TASKBOARD_DATABASE_URL, TASKBOARD_HOST, TASKBOARD_PORT with SPEC section 4 defaults; invalid values fail startup without leaking the URL.
Level: unit
Test file: taskboard/internal/config/config_test.go
Bead ID: ta-93p
Phase: phase-1-core
Scenarios:
- No env set yields sqlite, taskboard.db, 127.0.0.1, 8080, empty URL.
- Driver accepts exactly sqlite or postgres; any other value errors.
- Port 0, 65536, and non-numeric values error; 1 and 65535 pass.
- postgres driver with empty TASKBOARD_DATABASE_URL errors with no SQLite fallback.
Assertions:
- Config fields equal expected values (t.Setenv per case).
- Error text does not contain the URL or password substring.

Requirement: Domain validation: project name trimmed to 1-80 code points; task title trimmed to 1-120; description verbatim up to 1000; status and priority enums exact.
Level: unit
Test file: taskboard/internal/model/model_test.go
Bead ID: ta-i6d
Phase: phase-1-core
Scenarios:
- Boundaries at 0, 1, 80, 81, 120, 121, 1000, 1001 runes, including multibyte runes.
- Whitespace-only name and title are rejected; description is not trimmed.
- Each valid status and priority passes; one invalid value per enum fails.
Assertions:
- Exactly 80 multibyte runes pass for names; 81 fail (utf8.RuneCountInString semantics).
- Validators return nil for valid input and an error for invalid input.

Requirement: SQLite store creates schema idempotently, enforces foreign keys on every pooled connection, uses a 5000 ms busy timeout, and persists across reopen.
Level: integration
Test file: taskboard/internal/store/sqlite_test.go
Bead ID: ta-zmd
Phase: phase-1-core
Scenarios:
- Fresh file schema is empty; second init on the same file keeps existing rows.
- Create and list projects in ascending ID order; tasks default to todo and medium.
- Filters by status, priority, and both combine with AND; list order ascending ID.
- Patch changes only supplied fields; deleting a project cascades to its tasks.
- Insert task with unknown project_id fails with a foreign key error.
- Close and reopen the same t.TempDir() file preserves data.
Assertions:
- Returned rows equal expected values; timestamps parse as UTC RFC3339.
- Test DB path is under t.TempDir(), never taskboard.db.
- Queries use ? placeholders; user input is passed only as parameters.

### phase-1-app
Requirement: Shared store contract (project and task CRUD, filters, cascade, Unicode checks, parameterization) behaves identically on SQLite; PostgreSQL uses advisory-lock schema setup and the same contract when configured.
Level: integration
Test file: taskboard/internal/store/store_test.go
Bead ID: ta-444
Phase: phase-1-app
Scenarios:
- Fresh schema empty; repeated init idempotent.
- Project create and list ascending; task defaults; filters; patch preserves omitted fields; project delete cascades.
- Foreign keys enforced; persistence across close and reopen.
- Unicode code-point boundaries; SQL-injection-style names stored as literal text.
Assertions:
- Store results equal expected values and ordering.
- No DROP or schema reset statement runs during repeated init.

Requirement: PostgreSQL store runs the same contract against a dedicated disposable test database when TASKBOARD_TEST_POSTGRES_URL is set; skipped otherwise.
Level: integration
Test file: taskboard/internal/store/postgres_integration_test.go
Bead ID: ta-pn0
Phase: phase-1-app
Scenarios:
- Skipped with t.Skip when TASKBOARD_TEST_POSTGRES_URL is empty.
- Concurrent schema init is safe under advisory lock.
- Cascade delete, ordering, filters, UTC timestamps, and identity IDs match SQLite.
- Test objects use unique names or an isolated schema and clean up only their own data.
Assertions:
- Same expected values as the SQLite contract tests.
- Error output does not include the connection URL.

Requirement: Project REST API: GET api/health, GET api/ready, GET and POST api/projects, DELETE api/projects/{project_id} with SPEC section 6 status codes and JSON shapes.
Level: integration
Test file: taskboard/internal/web/api_test.go
Bead ID: ta-dyt
Phase: phase-1-app
Scenarios:
- Health returns {"status":"ok"} with 200 independent of DB state.
- Ready returns {"status":"ready"} on healthy SQLite and 503 {"error":"database unavailable"} on failure.
- Project create returns 201 with JSON; blank name returns 400 with an error object.
- Project list returns [] when empty and ascending ID order otherwise; delete returns 204 with empty body, then 404 for missing.
- Malformed JSON and invalid IDs return documented 400 or 404.
- Storage failure returns safe 500 JSON with no SQL text.
Assertions:
- Decoded JSON values, keys, and types match SPEC; whitespace is not asserted.
- Status codes and Content-Type are exact; 204 body length is 0.

### phase-2-kanban
Requirement: Task REST API (list, create, patch, delete) with server-owned status, IDs, and timestamps, nonempty PATCH, explicit-null rejection, and AND-combined filters.
Level: integration
Test file: taskboard/internal/web/app_test.go
Bead ID: ta-8qv
Phase: phase-2-kanban
Scenarios:
- Task create defaults status todo and priority medium; client-supplied status, id, and timestamps are ignored.
- PATCH with a nonempty subset updates only those fields; empty PATCH and explicit null values return 400.
- Task list filters by status, priority, and both; invalid filter values return 400.
- Missing project returns 404 on task list and create; missing task returns 404 on patch and delete.
Assertions:
- Decoded task JSON matches expected values; updated_at changes on PATCH.
- Filtered lists contain exactly the expected IDs in ascending order.

Requirement: Rendered HTML contains the TaskBoard heading, project and task forms, and To Do, Doing, Done headings; static assets are served and nonempty.
Level: integration
Test file: taskboard/internal/web/ui_content_test.go
Bead ID: ta-jl3
Phase: phase-2-kanban
Scenarios:
- Root page contains TaskBoard, the project input and Add project button, the task form with Add task, and the three column headings.
- Static app.js and style.css return 200 with nonempty bodies.
Assertions:
- Status 200 and Content-Type text/html for the root page; expected substrings present.
- Static bodies have nonzero length and expected content types.

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
