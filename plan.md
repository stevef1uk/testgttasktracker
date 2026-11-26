# Implementation plan (phase-3-browser)

## Bead map

### ta-4r9: taskboard/package.json
- Scope: Playwright test package manifest with the @playwright/test dependency and the chromium test script.
- Architecture: browser test toolchain for phase-3-browser (architecture.md, Deployment and Test sections).
- Interfaces: npm scripts `test`; devDependency `@playwright/test` 1.62.1.
- Depends on: <none>
- Consumed by: taskboard/package-lock.json, taskboard/playwright.config.ts
- Acceptance: manifest parses as JSON; the @playwright/test version is pinned; the lock file stays in sync.
- Verify: `cd taskboard && npm ci --ignore-scripts`

### ta-1jk: taskboard/package-lock.json
- Scope: lock file pinning the resolved Playwright dependency tree.
- Architecture: reproducible browser test install (architecture.md).
- Interfaces: none (generated).
- Depends on: taskboard/package.json
- Consumed by: npm ci --ignore-scripts in the verify step.
- Acceptance: lock file matches package.json; `npm ci --ignore-scripts` succeeds without network drift.
- Verify: `cd taskboard && npm ci --ignore-scripts`

### ta-aug: taskboard/playwright.config.ts
- Scope: Playwright config with testDir ./playwright, chromium project, baseURL from BASE_URL with default http://127.0.0.1:8080, and a webServer that starts the Go server.
- Architecture: browser QA layer runs against the real taskboard server (architecture.md, Kanban UI and Test sections).
- Interfaces: default export from defineConfig; project name `chromium`.
- Depends on: taskboard/package.json, taskboard/package-lock.json
- Consumed by: taskboard/playwright/taskboard.spec.ts
- Acceptance: config loads; the chromium project is defined; baseURL and webServer target the local server on port 8080.
- Verify: `cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`

### ta-ud0: taskboard/playwright/taskboard.spec.ts
- Scope: browser flow tests for the Kanban UI: initial shell, project create and select, task create into To Do, Move forward through Doing to Done, delete, and high-priority filter with reset.
- Architecture: Kanban UI behavior in SPEC (columns To do, Doing, Done; controls #project-select, #project-name, #task-title, #status).
- Interfaces: Playwright test file; uses page locators only, no server internals.
- Depends on: taskboard/playwright.config.ts
- Consumed by: phase verify command.
- Acceptance: the initial-shell test asserts the three kanban column sections (headings col-todo, col-doing, col-done, or the .column sections) are visible rather than the empty data-column containers, which have no height by themselves; all six browser flows pass against the running server.
- Verify: `cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`

## Out of scope for this phase

- style.css min-height changes belong to closed bead ta-sr8 (earlier phase). They are deferred to that phase and are not edited here.

## Future considerations

- None for phase-3-browser.
