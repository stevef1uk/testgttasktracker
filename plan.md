# Implementation plan

Active phase: `phase-2-kanban` (Tasks, Kanban UI, full Go tests). Phase gate: `cd taskboard && go test ./...`.
Scope is limited to the four first-delivery files below. Later phases (store, API, Playwright, Docker, k8s) get their own beads.

## Bead map

### ta-lkt: taskboard/web/static/style.css
- Scope: Styles for the Kanban layout and forms. No JavaScript behavior in this file.
- Architecture: `taskboard/web/static/style.css` owns UI styles and column responsive design (architecture.md module ownership table). It is embedded by `taskboard/web/assets.go`.
- Verify: `cd taskboard && go test ./internal/web/...`
- Acceptance:
  - SPEC §7: three Kanban columns (`To Do`, `Doing`, `Done`) render side by side at desktop width and stack or scroll at narrow width.
  - Project and task forms, filter controls, and the visible error region are styled and usable on desktop and narrow viewports.
  - Cards show title and priority with room for `Move forward`, `Edit`, and `Delete` buttons.
  - No animations, icon fonts, or external CSS dependencies (SPEC §7).
  - The embedded stylesheet is nonempty real CSS, as checked by `taskboard/internal/web/ui_content_test.go`.
  - Bead `ta-lkt` is closed only after the Verify command passes.

### ta-bsk: taskboard/web/static/app.js
- Scope: Vanilla JavaScript client logic: fetch calls, DOM rendering, filters, and user actions. No frameworks and no build step.
- Architecture: `taskboard/web/static/app.js` owns frontend JS dynamics, fetch calls, status updates, and filters. It is served as `/static/app.js` (SPEC §7).
- Verify: `cd taskboard && go test ./internal/web/...`
- Acceptance:
  - On load, fetches `/api/projects` and populates the project selector; selecting a project loads its tasks.
  - `Add project` and `Add task` submit through fetch. Task title is required; priority is one of `low`, `medium`, `high`.
  - Tasks render in the column matching their status. `Move forward` goes `todo → doing → done`, and the control is hidden or disabled for `done`.
  - `Edit` updates title, description, and priority through native controls. The UI changes status only through `Move forward`.
  - Status and priority filters pass `status` and `priority` as API query parameters; filtered-out cards are not rendered.
  - Project deletion calls `window.confirm`, then deletes and refreshes the selection.
  - Empty states appear for no projects and for no matching tasks. API failures show in the visible error region.
  - User text is inserted only with `textContent` or safe DOM APIs, never interpolated into `innerHTML`.
  - Bead `ta-bsk` is closed only after the Verify command passes.

### ta-88r: taskboard/internal/web/app_test.go
- Scope: Go `httptest` tests for HTTP routing and server initialization.
- Architecture: `taskboard/internal/web/app_test.go` covers HTTP routing and server initialization. Store integration tests are out of scope.
- Verify: `cd taskboard && go mod tidy && go test -timeout 30s -count=1 ./internal/web/...`
- Acceptance:
  - SPEC §7: `GET /` returns 200 with `text/html` rendered by `html/template` from the embedded `taskboard/web/templates/index.html`.
  - `GET /static/app.js` returns 200 with a JavaScript content type and a nonempty body. The embedded stylesheet is served with a CSS content type and a nonempty body.
  - Unknown static paths return 404 and do not serve a directory listing.
  - Tests use `t.TempDir()` SQLite files only, never `taskboard.db`. Tests do not depend on execution order.
  - Each assertion checks a real body or header value, not only the status code.
  - Bead `ta-88r` is closed only after the Verify command passes.

### ta-kf2: taskboard/internal/web/ui_content_test.go
- Scope: Unit tests for embedded UI content and template output.
- Architecture: `taskboard/internal/web/ui_content_test.go` asserts embedded UI assets and templates render correctly. REST API JSON tests are out of scope.
- Verify: `cd taskboard && go mod tidy && go test -timeout 30s -count=1 ./internal/web/...`
- Acceptance:
  - SPEC §8 UI content test 1: the body of `GET /` contains `TaskBoard`, the project form, the task form, and the headings `To Do`, `Doing`, and `Done`.
  - SPEC §8 UI content test 2: the static JS and CSS responses are 200 with nonempty real content (not placeholders).
  - Assertions check the exact strings above. A test that only checks status 200 does not satisfy this bead.
  - Bead `ta-kf2` is closed only after the Verify command passes.

## Phase context
- Phase verification: `cd taskboard && go test ./...`
- Manual gate (SPEC Phase 2): browser smoke for project/task creation, movement, edit, filter, and deletion. Both static resources are nonempty and the real UI renders.
