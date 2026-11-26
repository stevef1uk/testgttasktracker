# Test Plan — TaskBoard

## Requirements → tests

### phase-1-core
Requirement: Config parses TASKBOARD_* env vars with defaults, rejects invalid driver, port, and empty PostgreSQL URL, and redacts passwords; model enforces trimmed Unicode code-point limits (name 80, title 120, description 1000), enum status/priority, and nonempty PATCH; SQLite store initializes schema idempotently, enforces foreign keys, filters, patches, cascades deletes, and persists across reopen.
Level: unit
Test file: taskboard/internal/model/model_test.go, taskboard/internal/config/config_test.go, taskboard/internal/store/sqlite_test.go
Bead ID: model_test.go → ta-ne6 (model.go); config_test.go → ta-xlw (config.go); sqlite_test.go → ta-nlw (sqlite.go, reopened for these tests); store.go contract exercised via ta-29f
Phase: phase-1-core
Scenarios:
- Model: names and titles are trimmed; 80/120/1000 code-point boundaries accepted and rejected at +1 (multi-byte runes counted once).
- Model: unknown status or priority rejected; empty TaskPatch rejected; PATCH with only unknown fields counts as empty.
- Config: no env vars yields sqlite, taskboard.db, 127.0.0.1, 8080.
- Config: invalid driver, port 0 or 65536, and postgres with empty URL return errors.
- Config: Redacted() output never contains the password from the URL.
- SQLite: fresh schema is empty; a second OpenSQLite on the same file is idempotent.
- SQLite: projects list in ascending ID order; empty list is non-nil.
- SQLite: new tasks default to status todo and priority medium; created_at and updated_at parse as RFC3339 in UTC.
- SQLite: ListTasks filters by status only, priority only, and both combined (AND).
- SQLite: TaskPatch changes only supplied fields and preserves omitted fields.
- SQLite: deleting a project removes its tasks (cascade).
- SQLite: task insert referencing a missing project returns ErrForeignKey.
- SQLite: data persists after Close and reopen on the same t.TempDir() file.
Assertions:
- Returned errors match sentinel values via errors.Is (ErrNotFound, ErrForeignKey).
- Timestamps parse as RFC3339 with UTC offset; no test asserts exact wall-clock time.
- Lists compare ID order and are empty slices, not nil, when no rows.
- Config errors and Redacted output do not contain the raw password substring.
- Each SQLite test uses its own t.TempDir() database; no shared taskboard.db.

### phase-1-app
Requirement: PostgreSQL store implements the same contract with advisory-locked idempotent schema init; cmd/taskboard starts, shuts down on SIGTERM, and serves api health, api ready, project routes, and the embedded index.html shell; project create/list/delete round-trips over HTTP.
Level: integration
Test file: taskboard/internal/store/store_test.go, taskboard/internal/store/postgres_integration_test.go, taskboard/internal/web/api_test.go
Bead ID: pending — no phase-1-app bead exists yet in bd list --limit=0 ; planning must create one owner bead for each test file above
Phase: phase-1-app
Scenarios:
- Shared store contract runs against SQLite always and PostgreSQL only when TASKBOARD_TEST_POSTGRES_URL is set.
- PostgreSQL integration uses a dedicated disposable database with uniquely named objects and cleans up only its own data.
- GET api health returns 200 with exact body {"status":"ok"} without touching the database.
- GET api ready returns 200 {"status":"ready"} on healthy SQLite and 503 {"error":"database unavailable"} when the store ping fails.
- POST api projects returns 201 with project JSON; blank or overlong name returns 400 with {"error":...}.
- GET api projects returns ascending ID order, and [] when empty.
- DELETE api projects by ID returns 204 with empty body; missing ID returns 404.
- Malformed integer path segment returns 400 when the route matches.
- GET root returns the rendered index.html shell.
Assertions:
- Status code, content type, and decoded JSON body are checked for every route.
- Error bodies decode to exactly one "error" key with no SQL text or DSN content.
- After delete, a follow-up list no longer contains the project.
- Store-level failure on list returns 500, not an empty array.

### phase-2-kanban
Requirement: Task CRUD over HTTP with validation, server-controlled status/IDs/timestamps on create, AND-combined filters, Kanban UI with project selector, forms, move-forward, edit, delete, visible errors, and empty states; embedded static assets served with correct content types.
Level: integration
Test file: taskboard/internal/web/app_test.go, taskboard/internal/web/ui_content_test.go
Bead ID: pending — phase-2-kanban beads to be created by planning (app_test.go and ui_content_test.go each need a unique owner bead)
Phase: phase-2-kanban
Scenarios:
- POST project tasks returns 201 with status todo even when the client sends status, id, or timestamps.
- POST with missing description yields "" and missing priority yields medium; explicit null description on create is accepted as "".
- PATCH with a nonempty subset updates only those fields; explicit null PATCH field returns 400; empty PATCH returns 400.
- PATCH or POST with unknown status or priority returns 400; project reassignment is ignored or rejected.
- GET project tasks with status and priority together applies AND; invalid filter returns 400.
- Missing task on PATCH or DELETE returns 404; DELETE returns 204 with empty body.
- Overlong title (121 code points) and description (1001 code points) return 400.
- GET root contains the TaskBoard heading, project form, task form, and To Do, Doing, Done headings.
- GET static app.js and static style.css return 200 with nonempty bodies and correct content types.
Assertions:
- Decoded task JSON equals expected field values, not just status 200.
- Lists are in ascending ID order and empty lists serialize as [].
- Database failure path returns generic 500 body without SQL text.

### phase-3-browser
Requirement: Playwright E2E drives the running app through real browser controls: initial shell with three columns; create project and task, Move forward To Do to Doing to Done, delete task; filter to high priority and reset.
Level: ui
Test file: taskboard/playwright/taskboard.spec.ts
Bead ID: pending — phase-3-browser beads to be created by planning
Phase: phase-3-browser
Scenarios:
- Initial page shows TaskBoard heading and To Do, Doing, Done columns.
- Create a uniquely named project and a task through the browser form; the task appears in the To Do column.
- Click Move forward twice; the card moves to Doing, then Done; the forward control is hidden or disabled on Done.
- Click Delete on the task; it disappears from the DOM and the API no longer lists it.
- Create two tasks with high and low priority; select the high filter; only the high-priority card is rendered; reset to all; both return.
- Projects created by each test are deleted in cleanup.
Assertions:
- At least one test performs real click, fill, and select actions and asserts DOM state change.
- Asserted DOM text and API list responses reflect the state change after each action.
- Tests do not depend on prior-run data and do not rely on execution order.
- Static-content-only checks are not counted as UI coverage.
