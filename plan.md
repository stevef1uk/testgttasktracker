# Implementation plan: phase-2-kanban

Scope: Tasks, Kanban UI, and full Go tests. Only the four files below belong to this phase. Later phases get their own beads. Bead IDs come from `bd list --limit=0 --status=open,in_progress` in this session.

## Bead map

### ta-bsk: taskboard/web/static/app.js
- Scope: Frontend JavaScript for project selection, task create/edit/delete, move-forward between To Do, Doing, and Done, status and priority filters, visible errors, and empty states. No new frontend build tool.
- Architecture: architecture.md lists app.js under frontend JS dynamics, fetch calls, status updates, and filters, depending on the template structure. SPEC route table: `GET /static/app.js` serves the embedded JavaScript asset.
- Interfaces: served at `GET /static/app.js` with a JavaScript content type. Calls the REST routes in the SPEC route table (`/api/projects`, `/api/health`, `/api/ready`, plus the task routes defined in SPEC §7).
- Depends on: taskboard/web/templates/index.html (element IDs and form structure), taskboard/internal/web/api.go (REST handlers).
- Consumed by: taskboard/internal/web/ui_content_test.go (asserts the asset is non-empty and served).
- Acceptance: the file is non-empty and references the three Kanban columns To Do, Doing, and Done. Move-forward issues a task status update request. Status and priority filters and visible error display are wired to the page.
- Verify: cd taskboard && go test ./...

### ta-lkt: taskboard/web/static/style.css
- Scope: Stylesheet for the TaskBoard layout, forms, and responsive Kanban columns.
- Architecture: architecture.md lists style.css under UI styles and column responsive design, independent of JS behavior. It is embedded in taskboard/web/assets.go and served by the static asset handler with a CSS content type (SPEC §7 prose).
- Interfaces: static asset only; no exported Go symbols. The served route is recorded under Future considerations because the SPEC route table does not list it.
- Depends on: taskboard/web/templates/index.html (class names used by the template).
- Consumed by: taskboard/internal/web/ui_content_test.go (asserts the asset is non-empty).
- Acceptance: the file is non-empty. Kanban columns sit side by side on wide viewports and wrap on narrow ones.
- Verify: cd taskboard && go test ./...

### ta-88r: taskboard/internal/web/app_test.go
- Scope: Tests for HTTP routing, static asset routing, and server initialization, using direct store integration.
- Architecture: architecture.md lists app_test.go for test router setup, static asset routing, and server lifecycle. It depends on the store and api.go handlers.
- Interfaces: test functions only; no exported symbols.
- Depends on: taskboard/internal/web/api.go, taskboard/internal/web/app.go, taskboard/web/assets.go.
- Consumed by: none.
- Acceptance: `GET /` returns HTML containing TaskBoard and the three Kanban headings. `GET /static/app.js` returns 200 with a JavaScript content type. Health and ready routes return 200. Project and task CRUD and filtering behave as SPEC §7 describes.
- Verify: cd taskboard && go test ./...

### ta-kf2: taskboard/internal/web/ui_content_test.go
- Scope: Unit tests asserting that embedded templates and static assets render valid, non-empty responses.
- Architecture: architecture.md lists ui_content_test.go for asserting the index template and static assets render valid non-empty responses. It must not duplicate REST JSON tests.
- Interfaces: test functions only; no exported symbols.
- Depends on: taskboard/web/assets.go (embedded files), taskboard/web/static/app.js, taskboard/web/static/style.css.
- Consumed by: none.
- Acceptance: the embedded index template renders without error and contains the three Kanban headings. app.js and style.css are non-empty.
- Verify: cd taskboard && go test ./...

## Future considerations
- SPEC §7 prose says `/static/style.css` is served, but the SPEC route table lists only `/static/app.js`. This plan does not add a stylesheet route to any HTTP table. The architect should confirm whether the stylesheet route belongs in the canonical table before phase-2-kanban is accepted.
- Anything outside phase-2-kanban is deferred to its own phase.
