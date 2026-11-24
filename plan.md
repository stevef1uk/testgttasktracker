# Implementation plan

Active phase: phase-2-kanban (Tasks, Kanban UI, and full Go tests).
Depends on: phase-1-app. Verify: `cd taskboard && go test ./...`

Layout root is `taskboard/`. Every path below is repository-relative with the `taskboard/` prefix and matches workflow required_files exactly.

## Bead map

### ta-91v: taskboard/web/static/app.js
- Scope: vanilla JavaScript for project selection, task create/edit/delete, Move forward (todo -> doing -> done), status and priority filters, and visible API errors.
- Architecture: taskboard/web/static/app.js owns browser UI state; it is served through the taskboard/web/assets.go embed of static/*.
- Interfaces: none exported; browser globals only.
- Depends on: taskboard/web/assets.go (embed patterns for static/*).
- Consumed by: taskboard/internal/web/ui_content_test.go checks static delivery.
- Acceptance: user text is inserted with textContent only; forward control hidden or disabled for done; filters send API query parameters; project delete requires window.confirm; filtered-out cards are not rendered.
- Verify: `cd taskboard && go test ./...`

### ta-qwk: taskboard/web/static/style.css
- Scope: plain CSS for three Kanban columns (To Do, Doing, Done), cards, forms, and the error region.
- Architecture: taskboard/web/static/style.css is embedded and served with text/css.
- Interfaces: none exported; CSS only.
- Depends on: taskboard/web/assets.go (embed patterns for static/*).
- Consumed by: taskboard/internal/web/ui_content_test.go checks static delivery.
- Acceptance: layout remains usable at desktop and narrow widths; no frameworks, build step, or animations.
- Verify: `cd taskboard && go test ./...`

### ta-oe1: taskboard/internal/web/app_test.go
- Scope: router-level httptest tests for GET /, GET /static/app.js and /static/style.css content types, and task REST behavior.
- Architecture: taskboard/internal/web/app.go and taskboard/internal/web/api.go handlers; tests assert state changes, not only HTTP 200.
- Interfaces: test functions only.
- Depends on: taskboard/internal/web/app.go (NewApp and router).
- Consumed by: none.
- Acceptance: client-supplied status is ignored on task create; PATCH rejects empty objects and explicit nulls with 400; listing tasks of a missing project returns 404; combined filters use AND.
- Note: defects in taskboard/internal/web/api.go are owned by completed phase-1-app. QA rewind handles them. This phase adds no new beads for them.
- Verify: `cd taskboard && go test ./...`

### ta-s7n: taskboard/internal/web/ui_content_test.go
- Scope: HTML content assertions on GET / for the TaskBoard heading, project and task forms, priority select options, and Kanban headings To Do, Doing, Done.
- Architecture: taskboard/internal/web/ui_content_test.go asserts template structure and exact option sets.
- Interfaces: test functions only.
- Depends on: taskboard/internal/web/app.go and taskboard/web/templates/index.html.
- Consumed by: none.
- Acceptance: exact priority option set low, medium, high; exact status filter option set all, todo, doing, done; priority filter option set all, low, medium, high; static JS and CSS return 200 with nonempty content.
- Verify: `cd taskboard && go test ./...`

## Future considerations
- Playwright E2E belongs to phase-3-browser and is out of scope for this phase.
