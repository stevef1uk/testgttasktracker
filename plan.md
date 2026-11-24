# Implementation plan

Active phase: phase-2-kanban (Tasks, Kanban UI, and full Go tests). Phase verification: `cd taskboard && go test ./...`.
Only the four first-delivery files below are in scope. Depends on phase-1-app (`taskboard/internal/web/app.go`, `taskboard/web/assets.go`, `taskboard/internal/web/api.go`, `taskboard/internal/store/store.go`).

## Bead map

### ta-91v: taskboard/web/static/app.js
- Scope: vanilla JavaScript client for the Kanban board. No framework, no React state, no build step.
- Architecture: SPEC §7 browser contract; architecture.md "Server Wiring" and phase-2-kanban file dependency on `taskboard/web/assets.go`.
- Interfaces: none exported to Go. Consumes REST routes only: GET/POST `/api/projects`, DELETE `/api/projects/{project_id}`, GET/POST `/api/projects/{project_id}/tasks`, PATCH/DELETE `/api/tasks/{task_id}`.
- Depends on: taskboard/web/assets.go (embeds static/*), taskboard/internal/web/app.go (serves /static/app.js).
- Consumed by: taskboard/internal/web/app_test.go, taskboard/internal/web/ui_content_test.go, taskboard/playwright/taskboard.spec.ts (phase-3).
- Acceptance:
  - On page load, the project selector is populated by fetching `/api/projects`.
  - Selecting a project loads and displays its tasks from `/api/projects/{project_id}/tasks`.
  - Task form requires a title (client-side and server-error surfaced), has optional description, and a priority select with low/medium/high (default medium).
  - Three columns rendered with headings To Do, Doing, Done; each card shows title and priority.
  - Each card has Move forward, Edit, and Delete controls. Move forward goes todo to doing to done via PATCH `status`; the forward control is hidden or disabled on done cards.
  - Edit supports title, description, and priority via native controls and PATCH.
  - Status filter (all/todo/doing/done) and priority filter (all/low/medium/high) refresh the list using API query parameters; filtered-out cards are not rendered.
  - Project delete uses `window.confirm`, cascades on the server, and refreshes the selection.
  - Empty states shown for no projects and for no matching tasks.
  - API failures are shown in a visible error region.
  - User text is inserted with `textContent` or DOM APIs only; never `innerHTML` with user data.
  - Verify: `cd taskboard && go test ./...`
  - Bead ta-91v is closed only after verification passes.

### ta-qwk: taskboard/web/static/style.css
- Scope: plain CSS for the three Kanban columns, cards, controls, filters, error region, and empty states.
- Architecture: SPEC §7 layout requirement; architecture.md phase-2-kanban file dependency on `taskboard/web/assets.go`.
- Interfaces: none (static asset served by the embedded FS).
- Depends on: taskboard/web/assets.go.
- Consumed by: taskboard/web/templates/index.html (class names), taskboard/internal/web/app_test.go.
- Acceptance:
  - File is nonempty real CSS (not a placeholder comment) and is served at `/static/style.css` with HTTP 200 and content type `text/css`.
  - Styles three Kanban columns (To Do, Doing, Done) and task cards distinctly.
  - Layout uses flex or grid so columns sit side by side on desktop and stack or wrap on narrow widths (for example under 600px).
  - Styles form controls, filter controls, error region, and empty-state text.
  - No external font, icon, or framework dependencies.
  - Verify: `cd taskboard && go test ./...`
  - Bead ta-qwk is closed only after verification passes.

### ta-oe1: taskboard/internal/web/app_test.go
- Scope: httptest tests for the web app shell and static asset delivery.
- Architecture: architecture.md "Unit tests" (app_test.go: HTML rendering on GET /, content types for static assets).
- Interfaces: uses `web.NewApp(store, assets)` from taskboard/internal/web/app.go and the embedded FS from taskboard/web/assets.go.
- Depends on: taskboard/internal/web/app.go, taskboard/web/assets.go, taskboard/internal/store/sqlite.go.
- Consumed by: none.
- Acceptance:
  - Uses an isolated temp SQLite database via `t.TempDir()`, never `taskboard.db`.
  - `httptest` GET `/` returns 200 with `text/html` content type and a body containing `TaskBoard`, the project form (input and `Add project`), the task form elements (title, description, priority select, `Add task`), and the headings `To Do`, `Doing`, `Done`.
  - GET `/static/app.js` returns 200 with a JavaScript content type and a nonempty body.
  - GET `/static/style.css` returns 200 with `text/css` and a nonempty body.
  - A missing static file returns 404 (negative case).
  - Tests do not depend on execution order and each builds its own app and store.
  - Verify: `cd taskboard && go test ./...`
  - Bead ta-oe1 is closed only after verification passes.

### ta-s7n: taskboard/internal/web/ui_content_test.go
- Scope: DOM and structure assertions on the served HTML and the static UI contract.
- Architecture: architecture.md "Unit tests" (ui_content_test.go: HTML structure for forms, headings, Kanban columns).
- Interfaces: uses `web.NewApp(store, assets)` from taskboard/internal/web/app.go.
- Depends on: taskboard/internal/web/app.go, taskboard/web/assets.go, taskboard/web/static/app.js, taskboard/web/static/style.css.
- Consumed by: none.
- Acceptance:
  - Uses an isolated temp SQLite database via `t.TempDir()`.
  - GET `/` contains the element ids or attributes the UI needs: project input and `Add project` button; task title input (required), description textarea, priority select with options low, medium, high; status filter with all/todo/doing/done; priority filter with all/low/medium/high; project selector; and an error region.
  - Three Kanban column containers with headings exactly `To Do`, `Doing`, `Done`.
  - Served `app.js` references the API routes `/api/projects` and `/api/projects/` and uses `textContent` and no unescaped `innerHTML` assignment of user data.
  - Served `style.css` is nonempty and contains column/card selectors used by the template.
  - Verify: `cd taskboard && go test ./...`
  - Bead ta-s7n is closed only after verification passes.

## Phase context

Active delivery phase `phase-2-kanban` only. Later phases start automatically after QA `all_passed`.
- Phase verification: `cd taskboard && go test ./...`

## Future considerations

- Playwright E2E coverage belongs to phase-3-browser and is not part of this plan.

## Rework: api.go prerequisite for ta-oe1

ta-oe1 (`taskboard/internal/web/app_test.go`) cannot close until `taskboard/internal/web/api.go` satisfies the SPEC §6 behaviors its tests assert. That file is owned by the closed phase-1-app implement bead and must be reopened through planning sync, not by this phase's beads.
- File: taskboard/internal/web/api.go
- Owner: closed phase-1-app implement bead (reopen via `gt rig sync-planning tasktracker`; do not create a new bead for it).
- Required fix 1: createTask validates any supplied status but always stores `todo` (SPEC §6: status is always `todo` at creation).
- Required fix 2: patchTask rejects an empty patch object and explicit null patched fields with 400 (SPEC §6).
- Required fix 3: listTasks calls GetProject before ListTasks and returns 404 on ErrNotFound (SPEC §6: 404 missing project).
- Verify after fix: `cd taskboard && go test ./internal/web/...` must pass the three failing tests before ta-oe1 is re-verified.
- Depends on: taskboard/internal/web/api.go is a prerequisite of ta-oe1 and of ta-s7n's route checks.
