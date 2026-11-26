# Implementation plan

Active phase: `phase-2-kanban` (Tasks, Kanban UI, full Go tests). Phase verify: `cd taskboard && go test ./...`. Depends on `phase-1-app` (store, `web.NewApp`, project routes, embedded HTML shell already delivered).

## Bead map

### ta-tba: taskboard/web/static/app.js
- Scope: vanilla JavaScript client for the Kanban board. No framework, no build step, no innerHTML for user text.
- Architecture: browser layer in architecture.md; SPEC §7 Browser/UI contract.
- Interfaces: none exported; consumes the JSON API in SPEC §6 (`/api/projects`, `/api/projects/{id}/tasks`, `/api/tasks/{id}`).
- Depends on: taskboard/internal/web/api.go (existing project routes); taskboard/web/templates/index.html element IDs.
- Consumed by: taskboard/internal/web/ui_content_test.go (asserts `/static/app.js` returns 200 with nonempty content).
- Acceptance:
  - Project selector is populated from GET /api/projects on load; selecting a project fetches GET /api/projects/{id}/tasks and renders cards.
  - Project create (Add project) POSTs `{"name":...}` and refreshes the selector; blank names are shown as a visible error from the 400 response.
  - Project delete requires `window.confirm`, sends DELETE, and clears the selection.
  - Task form POSTs `{"title","description","priority"}`; required title; priority select is low/medium/high (default medium).
  - Three columns `To Do`, `Doing`, `Done` render tasks by status; each card shows title and priority pill.
  - `Move forward` PATCHes `{"status":next}` with todo→doing→done; the control is hidden or disabled for done.
  - Edit supports title, description, and priority via PATCH; Delete sends DELETE and removes the card on success.
  - Status filter (all/todo/doing/done) and priority filter (all/low/medium/high) pass query parameters; filtered-out cards are not rendered.
  - Empty states for no projects and no matching tasks; API failures appear in a visible error region.
  - All user text is inserted via textContent or DOM APIs only.
- Verify: `cd taskboard && go test ./...`
- Bead `ta-tba` closes only after verify passes.

### ta-sr8: taskboard/web/static/style.css
- Scope: plain CSS implementing the SPEC visual design system.
- Architecture: SPEC §7 "Visual design system & styling".
- Interfaces: none; class/ID hooks must match app.js and index.html (`#kanban-board`, column headers, card, priority pill classes).
- Depends on: taskboard/web/templates/index.html; taskboard/web/static/app.js class names.
- Consumed by: taskboard/web/assets.go (embedded via `static/*`).
- Acceptance:
  - Page background `#f8fafc`; cards `#ffffff` with border `#e2e8f0`; primary text `#0f172a`; secondary text `#64748b`; accent `#4f46e5`.
  - `#kanban-board` is a 3-column grid or flex with equal-width columns on desktop and wraps or scrolls on narrow widths.
  - Column headers have 8px radius, subtle tint, and a count chip.
  - Cards have 8px radius and the specified box-shadow; priority pills use low/medium/high colors from SPEC.
  - Inputs and selects use 6px radius, border `#cbd5e1`, and `outline: 2px solid #6366f1; outline-offset: 1px` on focus.
- Verify: `cd taskboard && go test ./...`
- Bead `ta-sr8` closes only after verify passes.

### ta-cqs: taskboard/internal/web/app_test.go
- Scope: httptest coverage of task CRUD, validation, filters, and error paths against a real SQLite store in `t.TempDir()`.
- Architecture: SPEC §8 API tests 3–8; architecture.md Unit tests.
- Interfaces: uses `web.NewApp(store, assets)` from taskboard/internal/web/app.go; no exported symbols.
- Depends on: taskboard/internal/web/api.go; taskboard/internal/store/sqlite.go (`OpenSQLite`).
- Consumed by: none.
- Acceptance:
  - Task create returns 201 with `status:"todo"` and `priority:"medium"` when omitted; client-supplied status, id, and timestamps are ignored.
  - Task list returns `[]` (not null) for an empty project and ascending-ID order otherwise.
  - PATCH with a nonempty subset changes only the named fields; empty PATCH body and explicit null fields return 400; project_id reassignment is rejected.
  - Combined `status` and `priority` filters use AND; invalid filter values return 400.
  - Blank or overlong title (>120 code points) returns 400; invalid enum status or priority returns 400.
  - Missing project on task create returns 404; missing task on PATCH or DELETE returns 404.
  - DELETE task returns 204 with an empty body and the task is gone on the next list.
- Verify: `cd taskboard && go test ./...`
- Bead `ta-cqs` closes only after verify passes.

### ta-omt: taskboard/internal/web/ui_content_test.go
- Scope: content assertions for `GET /` and static assets.
- Architecture: SPEC §8 UI content tests 1–2.
- Interfaces: none exported.
- Depends on: taskboard/web/assets.go; taskboard/web/templates/index.html; taskboard/web/static/app.js; taskboard/web/static/style.css.
- Consumed by: none.
- Acceptance:
  - GET `/` returns 200 and the body contains `TaskBoard`, the project input and `Add project` button, the task form, and the headings `To Do`, `Doing`, `Done`.
  - GET `/static/app.js` and `/static/style.css` return 200 with nonempty bodies and correct content types.
  - Test asserts on real body content, not only status codes.
- Verify: `cd taskboard && go test ./...`
- Bead `ta-omt` closes only after verify passes.

## Future considerations
- SPEC §8 Playwright E2E belongs to phase-3-browser and is not planned here.

## Phase context
Active delivery phase `phase-2-kanban` only. Later phases start after QA `all_passed`.
