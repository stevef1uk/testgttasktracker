# Implementation plan

Active delivery phase: phase-2-kanban (Tasks, Kanban UI, and full Go tests).
Phase verification: `cd taskboard && go test ./...`

## Scope

This phase owns exactly four required files, each with one implement bead:
- ta-n4t: taskboard/web/static/app.js (closed by a prior run; re-verify on rework)
- ta-9t7: taskboard/web/static/style.css (closed by a prior run; re-verify on rework)
- ta-8qv: taskboard/internal/web/app_test.go (closed by a prior run; re-verify on rework)
- ta-jl3: taskboard/internal/web/ui_content_test.go (in_progress; the only open phase-2 bead)

Phase 1 files (config, model, store, web app/api, main, assets.go, index.html) are already delivered. ta-pk0 (taskboard/web/templates/index.html) and ta-1ri (taskboard/internal/web/app.go) are closed phase-1 beads, referenced only as dependencies. taskboard/web/assets.go is extended to embed static/* but gets no bead of its own in this phase.

## Bead map

### ta-n4t: taskboard/web/static/app.js
- Scope: Vanilla JavaScript for project create, select, and delete (delete behind window.confirm); task create, edit (PATCH /api/tasks/{task_id}), move forward (todo to doing to done), and delete; status and priority filters sent as query parameters.
- Architecture: architecture.md phase-2-kanban; SPEC.md sections 6 and 7; served at /static/app.js.
- Interfaces: none exported (browser script). Consumes the REST routes in SPEC.md section 6 and DOM ids in taskboard/web/templates/index.html.
- Depends on: ta-pk0 (taskboard/web/templates/index.html), ta-1ri (taskboard/internal/web/app.go), taskboard/web/assets.go (embed of static/*).
- Consumed by: ta-8qv (taskboard/internal/web/app_test.go), ta-jl3 (taskboard/internal/web/ui_content_test.go).
- Acceptance:
  - User text is inserted via textContent or safe DOM APIs only; no innerHTML interpolation of user input.
  - Move forward is hidden or disabled for done tasks.
  - Filtered-out cards are not rendered; empty states shown for no projects and for no matching tasks.
  - API failures are shown in the visible error region.
- Verify: `cd taskboard && go test ./...`

### ta-9t7: taskboard/web/static/style.css
- Scope: Plain CSS for the three Kanban columns (To Do, Doing, Done), task cards, project selector, filter controls, error region, and empty states. No CSS framework, no build step.
- Architecture: architecture.md phase-2-kanban; SPEC.md section 7; served at /static/style.css.
- Interfaces: none exported; packaged as static/style.css in the embedded FS.
- Depends on: ta-pk0 (taskboard/web/templates/index.html, which references the classes), taskboard/web/assets.go (embed of static/*).
- Consumed by: ta-8qv (taskboard/internal/web/app_test.go), ta-jl3 (taskboard/internal/web/ui_content_test.go).
- Acceptance:
  - Three visible Kanban columns; layout usable at desktop and narrow widths.
  - Classes for cards, columns, error region, and empty states are referenced by taskboard/web/templates/index.html.
  - No animations, icon fonts, or external URLs.
- Verify: `cd taskboard && go test ./...`

### ta-8qv: taskboard/internal/web/app_test.go
- Scope: httptest tests for the App router: GET / returns the HTML shell; /static/app.js and /static/style.css return 200 with nonempty bodies and correct Content-Type; a missing static file returns 404.
- Architecture: architecture.md unit tests (app_test.go: HTML rendering on GET /, content types for static assets).
- Interfaces: uses App and NewApp from taskboard/internal/web/app.go and the embedded FS from taskboard/web/assets.go. Defines only test functions.
- Depends on: ta-1ri (taskboard/internal/web/app.go), ta-n4t (taskboard/web/static/app.js), ta-9t7 (taskboard/web/static/style.css).
- Consumed by: none.
- Acceptance:
  - Embedded FS includes static/*. If static assets return 404, the fix is the embed directive in taskboard/web/assets.go, not a change to these assertions.
  - Assertions check status, Content-Type, and nonempty body.
  - Cases are independent and do not depend on execution order.
  - No trivial always-pass tests.
- Verify: `cd taskboard && go test ./...`

### ta-jl3: taskboard/internal/web/ui_content_test.go
- Scope: Assertions on GET / markup: TaskBoard heading; project input with Add project button; task form with required title, optional description, priority select (low, medium, high), and Add task button; headings To Do, Doing, Done; status filter options (all, todo, doing, done); priority filter options (all, low, medium, high); references to the embedded JavaScript and stylesheet assets.
- Architecture: architecture.md ui_content_test.go entry; SPEC.md section 7.
- Interfaces: uses NewApp from taskboard/internal/web/app.go; markup from taskboard/web/templates/index.html. Defines only test functions.
- Depends on: ta-1ri (taskboard/internal/web/app.go), ta-pk0 (taskboard/web/templates/index.html), ta-n4t (taskboard/web/static/app.js), ta-9t7 (taskboard/web/static/style.css).
- Consumed by: none.
- Acceptance:
  - Each required element is checked by element, attribute, or text, not a loose substring match.
  - Referenced JavaScript and stylesheet assets return 200 with nonempty bodies.
  - No trivial always-pass tests.
- Verify: `cd taskboard && go test ./...`

## Future considerations
- Playwright E2E coverage (taskboard/playwright/taskboard.spec.ts) belongs to phase-3-browser and is not planned here.

## Rework amendment (planning review)

Status:
- ta-n4t, ta-9t7, ta-8qv, ta-1ri, ta-pk0: closed. Their files are on disk. Re-run `cd taskboard && go test ./...` before phase-2 sign-off.
- ta-jl3 (taskboard/internal/web/ui_content_test.go): in_progress. `cd taskboard && go mod tidy && go test -timeout 30s -count=1 ./internal/web/...` returned ok in the implementation session. The bead stays open because the polecat scope gate rejected ` bd dolt commit -m "auto-commit after bd close"
