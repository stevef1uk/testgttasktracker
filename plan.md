# Implementation plan

Active delivery phase: phase-2-kanban (Tasks, Kanban UI, and full Go tests).
Phase verification: `cd taskboard && go test ./...`

## Route alignment

API routes come from SPEC.md §6: /api/health, /api/ready, /api/projects, /api/projects/{project_id}, /api/projects/{project_id}/tasks, /api/tasks/{task_id}.
Browser routes come from SPEC.md §7: GET / serves the HTML shell, and the embedded static assets are served with appropriate content types. The JavaScript asset is served at /static/app.js. The stylesheet is the embedded `static/style.css` asset.
Open item for architect: SPEC.md:179 and §7 name the stylesheet route explicitly, but the validator's canonical list does not. Resolve in SPEC.md or the validator; this plan does not rename or remove the stylesheet.

## Bead map

### ta-9t7: taskboard/web/static/style.css
- Scope: Plain CSS for the three Kanban columns (To Do, Doing, Done), task cards, project selector, filter controls, error region, and empty states. No CSS framework, no build step.
- Architecture: architecture.md phase-2-kanban; SPEC.md §7 (embedded static stylesheet asset).
- Interfaces: none exported; packaged as `static/style.css` in the embedded FS.
- Depends on: <none>
- Consumed by: taskboard/internal/web/app_test.go, taskboard/internal/web/ui_content_test.go.
- Acceptance:
  - Three visible Kanban columns; layout usable at desktop and narrow widths.
  - Classes for cards, columns, error region, and empty states are referenced by taskboard/web/templates/index.html.
  - No animations, icon fonts, or external URLs.
- Verify: `cd taskboard && go test ./...`

### ta-n4t: taskboard/web/static/app.js
- Scope: Vanilla JavaScript: load projects from GET /api/projects; select project and load its tasks; create and delete projects (delete after window.confirm); create, edit (PATCH /api/tasks/{task_id}), move forward (todo to doing to done), and delete tasks; status and priority filters as query parameters.
- Architecture: architecture.md HTTP routes table; SPEC.md §6 and §7; served at /static/app.js per SPEC.md §7.
- Interfaces: none exported; consumes the REST routes in SPEC.md §6.
- Depends on: <none>
- Consumed by: taskboard/playwright/taskboard.spec.ts (phase-3-browser).
- Acceptance:
  - User text inserted via textContent or safe DOM APIs only; no innerHTML interpolation of user input.
  - Move forward hidden or disabled for done tasks.
  - Filtered-out cards are not rendered; empty states for no projects and no matching tasks.
  - API failures shown in the visible error region.
  - Real controls and event handlers exist for each action above.
- Verify: `cd taskboard && go test ./...`

### ta-8qv: taskboard/internal/web/app_test.go
- Scope: httptest tests for the App router: GET / returns the HTML shell; the embedded JavaScript and stylesheet assets return 200 with nonempty bodies and correct Content-Type; a missing static file returns 404.
- Architecture: architecture.md unit tests (app_test.go: HTML rendering on GET /, content types for static assets).
- Interfaces: uses App and NewApp from taskboard/internal/web/app.go and FS from taskboard/web/assets.go.
- Depends on: taskboard/internal/web/app.go (delivered in phase-1-app).
- Consumed by: none.
- Acceptance:
  - Embedded FS must include `static/*`. If static assets return 404, the fix is `//go:embed templates/* static/*` in taskboard/web/assets.go, not a change to these assertions.
  - Assertions check status, Content-Type, and nonempty body.
  - Cases are independent and do not depend on execution order.
  - No trivial always-pass tests.
- Verify: `cd taskboard && go mod tidy && go test -timeout 30s -count=1 ./internal/web/...`

### ta-jl3: taskboard/internal/web/ui_content_test.go
- Scope: Assertions on GET / markup: TaskBoard heading; project input with Add project button; task form with required title, optional description, priority select (low, medium, high), and Add task button; headings To Do, Doing, Done; status filter options (all, todo, doing, done); priority filter options (all, low, medium, high); references to the embedded JavaScript and stylesheet assets.
- Architecture: architecture.md ui_content_test.go entry; SPEC.md §7.
- Interfaces: uses NewApp from taskboard/internal/web/app.go; markup from taskboard/web/templates/index.html.
- Depends on: taskboard/internal/web/app.go (delivered in phase-1-app).
- Consumed by: none.
- Acceptance:
  - Each required element is checked by element, attribute, or text, not a loose substring match.
  - Referenced JavaScript and stylesheet assets return 200 with nonempty bodies.
  - No trivial always-pass tests.
- Verify: `cd taskboard && go mod tidy && go test -timeout 30s -count=1 ./internal/web/...`

## Future considerations
- Playwright E2E coverage belongs to phase-3-browser and is not planned here.
