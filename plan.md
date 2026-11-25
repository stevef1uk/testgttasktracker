# Implementation plan — phase-2-kanban

Active phase: Tasks, Kanban UI, and full Go tests. Depends on phase-1-app.
Phase gate: `cd taskboard && go test ./...`

Required files (exact repository-relative paths):
- taskboard/web/static/app.js
- taskboard/web/static/style.css
- taskboard/internal/web/app_test.go
- taskboard/internal/web/ui_content_test.go

## Bead map

### ta-bsk: taskboard/web/static/app.js
- Scope: Vanilla JavaScript client. Populates the project selector from GET /api/projects on page load. Loads tasks for the selected project. Renders three columns To Do, Doing, Done. Task creation form with required title, optional description, and priority select low/medium/high. Move forward follows todo → doing → done and is hidden or disabled on done. Edit title, description, and priority. Delete task. Status and priority filters refresh the list through API query parameters. Project deletion requires window.confirm. Visible error region for API failures. Empty states for no projects and no matching tasks.
- Architecture: taskboard/web/static/app.js owns frontend dynamics and fetch calls. Served by the embedded /static/ file server (SPEC §7).
- Interfaces: none exported (browser script). Calls the REST API defined in taskboard/internal/web/api.go.
- Depends on: taskboard/web/templates/index.html, taskboard/internal/web/api.go
- Consumed by: taskboard/internal/web/app_test.go, taskboard/internal/web/ui_content_test.go
- Acceptance: user-provided text is inserted with textContent or safe DOM APIs only, never innerHTML. Move forward only advances todo → doing → done. Filtered-out cards are not rendered. Status and priority filters combine with AND through query parameters.
- Verify: `cd taskboard && go test ./...`

### ta-lkt: taskboard/web/static/style.css
- Scope: Plain CSS for the three-column Kanban board, cards, forms, filter controls, error region, and empty states. Usable at desktop and narrow widths. No animations, icon dependencies, or frameworks.
- Architecture: taskboard/web/static/style.css owns UI styles and responsive column layout. Served by the embedded /static/ file server (SPEC §7).
- Interfaces: none (stylesheet).
- Depends on: taskboard/web/templates/index.html
- Consumed by: taskboard/internal/web/app_test.go, taskboard/internal/web/ui_content_test.go
- Acceptance: the stylesheet is nonempty and served with a CSS content type. Columns remain readable at narrow widths. No drag-and-drop or animation rules.
- Verify: `cd taskboard && go test ./...`

### ta-s3j: taskboard/internal/web/app_test.go
- Scope: httptest tests for routing and static serving. GET / returns the rendered index. The embedded app.js and style.css are served through the /static/ file server with 200 status, correct content type, and nonempty bodies. Unknown static paths return 404. The app built with web.NewApp(store, assets) serves the same routes as production wiring.
- Architecture: tests HTTP routing and server initialization (taskboard/internal/web/app.go). No direct SQL. Uses an isolated SQLite file under t.TempDir().
- Interfaces: uses web.NewApp and the embedded assets from taskboard/web/assets.go.
- Depends on: taskboard/internal/web/app.go, taskboard/web/static/app.js, taskboard/web/static/style.css, taskboard/web/assets.go
- Consumed by: none
- Acceptance: assertions check status, content type, and nonempty body, not only HTTP 200. Tests do not depend on execution order.
- Verify: `cd taskboard && go test ./internal/web/...`

### ta-5s6: taskboard/internal/web/ui_content_test.go
- Scope: Asserts GET / contains the text TaskBoard, the project form, the task form, and the three Kanban headings To Do, Doing, Done. Asserts the embedded app.js and style.css return 200 with real content: the app.js body contains the fetch call to /api/projects, and the style.css body contains a column layout rule.
- Architecture: asserts the index template and static assets render valid nonempty responses (taskboard/web/templates/index.html). No REST JSON assertions.
- Interfaces: uses web.NewApp and the embedded assets from taskboard/web/assets.go.
- Depends on: taskboard/web/templates/index.html, taskboard/web/static/app.js, taskboard/web/static/style.css, taskboard/web/assets.go
- Consumed by: none
- Acceptance: each heading and form control is asserted by text in the response body. Static assets are checked for nonempty content, not only status.
- Verify: `cd taskboard && go test ./internal/web/...`
