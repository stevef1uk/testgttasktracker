# Implementation plan

Active phase: phase-2-kanban (Tasks, Kanban UI, full Go tests).
Scope: only the four required_files for this phase. Other phases get their own beads later.

## Bead map

### ta-bsk: taskboard/web/static/app.js
- Scope: Frontend JS dynamics: fetch calls to the SPEC API routes (GET /api/projects, GET /api/projects/{project_id}/tasks, POST /api/projects/{project_id}/tasks, PATCH /api/tasks/{task_id}, DELETE /api/tasks/{task_id}), task status updates, and status and priority filters.
- Architecture: taskboard/web/static/app.js owns frontend JS dynamics, fetch calls, status updates, and filters. It must not define template structure.
- Interfaces: none (browser script, no Go exports).
- Depends on: <none>
- Consumed by: taskboard/internal/web/ui_content_test.go (asserts the asset is embedded and non-empty).
- Acceptance: the script is served at GET /static/app.js with a JavaScript content type.
- Acceptance: moving a task between todo, doing, and done issues PATCH /api/tasks/{task_id} with the new status.
- Acceptance: the status and priority filters update the rendered board without a page reload.
- Verify: cd taskboard && go test ./internal/web/...

### ta-lkt: taskboard/web/static/style.css
- Scope: UI styles for the Kanban board, including column layout and responsive design.
- Architecture: taskboard/web/static/style.css owns UI styles and column responsive design. It must not define JS behavior.
- Interfaces: none (stylesheet).
- Depends on: <none>
- Consumed by: taskboard/internal/web/ui_content_test.go (asserts the stylesheet is embedded and non-empty).
- Acceptance: the stylesheet is embedded in taskboard/web/assets.go and served by the same static asset handler as app.js.
- Acceptance: the To Do, Doing, and Done columns lay out side by side and wrap on narrow viewports.
- Verify: cd taskboard && go test ./internal/web/...

### ta-88r: taskboard/internal/web/app_test.go
- Scope: Tests for HTTP router setup, static asset routing, and server lifecycle.
- Architecture: taskboard/internal/web/app_test.go owns router and server initialization tests. It must not contain direct store integration tests.
- Interfaces: none (test file).
- Depends on: taskboard/internal/web/app.go, taskboard/web/assets.go
- Consumed by: <none>
- Acceptance: GET / returns the rendered index template.
- Acceptance: GET /static/app.js returns 200 with a JavaScript content type.
- Acceptance: the server starts and shuts down gracefully on context cancel.
- Verify: cd taskboard && go test ./internal/web/...

### ta-kf2: taskboard/internal/web/ui_content_test.go
- Scope: Unit tests asserting the embedded index template and static assets render non-empty responses.
- Architecture: taskboard/internal/web/ui_content_test.go owns asset and template content tests. It must not contain REST API JSON tests.
- Interfaces: none (test file).
- Depends on: taskboard/web/assets.go, taskboard/web/static/app.js, taskboard/web/static/style.css
- Consumed by: <none>
- Acceptance: the index template is non-empty and contains the task list and project list containers.
- Acceptance: taskboard/web/static/app.js and taskboard/web/static/style.css are non-empty embedded assets.
- Verify: cd taskboard && go test ./internal/web/...

## Future considerations
- The index template taskboard/web/templates/index.html belongs to completed phase-1-app. Its Kanban column and status-control repair is handled by the QA rewind, not this phase's plan.
