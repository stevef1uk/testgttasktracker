# Implementation plan

Active delivery phase: phase-2-kanban (Tasks, Kanban UI, and full Go tests).
Phase verification: `cd taskboard && go test ./...`

## Route alignment

API routes come from SPEC.md §6: /api/health, /api/ready, /api/projects, /api/projects/{project_id}, /api/projects/{project_id}/tasks, /api/tasks/{task_id}.
Browser routes come from SPEC.md §7: GET / serves the HTML shell (taskboard/web/templates/index.html). Embedded static assets are served under /static/: /static/app.js and /static/style.css.

## Embed dependency

taskboard/web/assets.go owns the embed directive. Phase 2 extends it to `//go:embed templates/* static/*`. Both static assets depend on it; the phase is not complete until the embed includes static/*. assets.go has no bead in this phase's required_files, so the polecat must confirm the directive covers static/* when working ta-n4t or ta-9t7.

## Bead map

### ta-n4t: taskboard/web/static/app.js
- Scope: Vanilla JavaScript for project create, select, and delete (delete behind window.confirm); task create, edit (PATCH /api/tasks/{task_id}), move forward (todo to doing to done), and delete; status and priority filters sent as query parameters.
- Architecture: architecture.md phase-2-kanban; SPEC.md §6 and §7; served at /static/app.js.
- Interfaces: none exported (browser script). Consumes the REST routes in SPEC.md §6 and the DOM ids in taskboard/web/templates/index.html.
- Depends on: taskboard/web/assets.go (embed of static/*), ta-pk0 (taskboard/web/templates/index.html), ta-1ri (taskboard/internal/web/app.go, serves the routes it calls).
- Consumed by: none in this phase (the browser E2E spec belongs to phase-3-browser and is not planned here).
- Acceptance:
  - User text is inserted via textContent or safe DOM APIs only; no innerHTML interpolation of user input.
  - Move forward is hidden or disabled for done tasks.
  - Filtered-out cards are not rendered; empty states shown for no projects and for no matching tasks.
  - API failures are shown in the visible error region.
  - Real controls and event handlers exist for each action above.
- Verify: `cd taskboard && go test ./...`

### ta-9t7: taskboard/web/static/style.css
- Scope: Plain CSS for the three Kanban columns (To Do, Doing, Done), task cards, project selector, filter controls, error region, and empty states. No CSS framework, no build step.
- Architecture: architecture.md phase-2-kanban; SPEC.md §7; served at /static/style.css.
- Interfaces: none exported; packaged as static/style.css in the embedded FS.
- Depends on: taskboard/web/assets.go (embed of static/*), ta-pk0 (taskboard/web/templates/index.html, which references the classes).
- Consumed by: taskboard/internal/web/app_test.go (ta-8qv) and taskboard/internal/web/ui_content_test.go (ta-jl3), both checking the asset is served.
- Acceptance:
  - Three visible Kanban columns; layout usable at desktop and narrow widths.
  - Classes for cards, columns, error region, and empty states are referenced by taskboard/web/templates/index.html.
  - No animations, icon fonts, or external URLs.
- Verify: `cd taskboard && go test ./...`

### ta-8qv: taskboard/internal/web/app_test.go
- Scope: httptest tests for the App router: GET / returns the HTML shell; /static/app.js and /static/style.css return 200 with nonempty bodies and correct Content-Type; a missing static file returns 404.
- Architecture: architecture.md unit tests (app_test.go: HTML rendering on GET /, content types for static assets).
- Interfaces: uses App and NewApp from taskboard/internal/web/app.go and the embedded FS from taskboard/web/assets.go. Defines only test functions.
- Depends on: ta-1ri (taskboard/internal/web/app.go), taskboard/web/assets.go (embed of static/*).
- Consumed by: none.
- Acceptance:
  - Embedded FS includes static/*. If static assets return 404, the fix is the embed directive in taskboard/web/assets.go, not a change to these assertions.
  - Assertions check status, Content-Type, and nonempty body.
  - Cases are independent and do not depend on execution order.
  - No trivial always-pass tests.
- Verify: `cd taskboard && go test -timeout 30s -count=1 ./internal/web/...`

### ta-jl3: taskboard/internal/web/ui_content_test.go
- Scope: Assertions on GET / markup: TaskBoard heading; project input with Add project button; task form with required title, optional description, priority select (low, medium, high), and Add task button; headings To Do, Doing, Done; status filter options (all, todo, doing, done); priority filter options (all, low, medium, high); references to the embedded JavaScript and stylesheet assets.
- Architecture: architecture.md ui_content_test.go entry; SPEC.md §7.
- Interfaces: uses NewApp from taskboard/internal/web/app.go; markup from taskboard/web/templates/index.html. Defines only test functions.
- Depends on: ta-1ri (taskboard/internal/web/app.go), ta-pk0 (taskboard/web/templates/index.html).
- Consumed by: none.
- Acceptance:
  - Each required element is checked by element, attribute, or text, not a loose substring match.
  - Referenced JavaScript and stylesheet assets return 200 with nonempty bodies.
  - No trivial always-pass tests.
- Verify: `cd taskboard && go test -timeout 30s -count=1 ./internal/web/...`

## Future considerations
- Playwright E2E coverage (taskboard/playwright/taskboard.spec.ts) belongs to phase-3-browser and is not planned here.
