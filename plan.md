# Implementation plan

Active phase: `phase-3-browser` (Playwright integration and browser QA).

Open or in-progress implement beads for this phase's `required_files`:
- `ta-o19` → `taskboard/package.json` (closed, content validated)
- `ta-qu7` → `taskboard/package-lock.json` (closed, content validated)
- `ta-ev8` → `taskboard/playwright.config.ts` (in progress)
- `ta-m0z` → `taskboard/playwright/taskboard.spec.ts` (open)

No new beads are created in this turn. The bead set already covers every phase-3-browser required file.

## Bead map

### ta-o19: taskboard/package.json
- Scope: npm manifest declaring `@playwright/test` as a devDependency and a `test` script that runs Playwright on the chromium project.
- Architecture: Playwright E2E suite, phase-3-browser `required_files`.
- Interfaces: `scripts.test` runs `npx playwright test --project=chromium`; devDependency `@playwright/test` pinned to 1.62.1.
- Depends on: `<none>`
- Consumed by: taskboard/package-lock.json, taskboard/playwright.config.ts, taskboard/playwright/taskboard.spec.ts
- Acceptance: manifest is valid JSON; `@playwright/test` is pinned to 1.62.1; no runtime dependencies are added.
- Verify: read-only check that the file parses as JSON and contains the pinned devDependency.

### ta-qu7: taskboard/package-lock.json
- Scope: lockfile matching package.json with lockfileVersion 3.
- Architecture: phase-3-browser dependency pinning.
- Interfaces: root entry name `playwright-tests`, version 1.0.0, devDependency `@playwright/test` 1.62.1.
- Depends on: taskboard/package.json
- Consumed by: phase-3-browser install step performed by the operator, not by the planner.
- Acceptance: lockfile is valid JSON; root devDependencies match taskboard/package.json exactly.
- Verify: read-only comparison of the root devDependencies against taskboard/package.json. Do not run installs at the rig root.

### ta-ev8: taskboard/playwright.config.ts
- Scope: Playwright runner configuration. The cached content sets `testDir: './e2e'`, but the E2E spec lives under `taskboard/playwright/`. Set `testDir` to `./playwright` so the runner discovers `taskboard.spec.ts`.
- Architecture: "Playwright Browser E2E Test Suite" section. The base URL defaults to `http://127.0.0.1:8080` per SPEC, overridable with `BASE_URL`.
- Interfaces: default export from `defineConfig`; single project `chromium` using `devices['Desktop Chrome']`.
- Depends on: taskboard/package.json
- Consumed by: taskboard/playwright/taskboard.spec.ts
- Acceptance: the config exports a `defineConfig` result; `testDir` resolves to the `playwright/` directory; the chromium project is the only project; the baseURL default is 127.0.0.1:8080.
- Verify: read-only inspection of the file contents. The phase verify command is `cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`, run by QA.

### ta-m0z: taskboard/playwright/taskboard.spec.ts
- Scope: E2E browser tests for project creation through the UI, adding a task, moving a task todo to doing to done with the "Move forward" button, priority filtering (high, medium, low), and task deletion. Each test creates uniquely named data and cleans it up.
- Architecture: "Playwright Browser E2E Test Suite" section.
- Interfaces: test file only. Uses element IDs from `taskboard/web/templates/index.html`: `project-name`, `project-create`, `task-title`, `task-create`, `status-filter`, `filter-priority`.
- Depends on: taskboard/playwright.config.ts, taskboard/package.json
- Consumed by: phase-3-browser verify command.
- Acceptance: each SPEC flow has at least one test; no test depends on another test's data; cleanup runs after each test.
- Verify: QA runs the phase verify command. The planner does not run tests.

## QA rework (blocks phase verify)

The phase verify command is `cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`. The Go failures below must be green before phase-3-browser can close:
- `TestTaskCreateDefaultsAndIgnoresClientFields`: create must ignore a client-supplied status (taskboard/internal/web/api.go, createTask).
- `TestTaskPatchRejectsEmptyAndNullFields`: empty or null PATCH bodies must return 400 (taskboard/internal/web/api.go, patchTask).
- `TestTaskMissingEntitiesReturnNotFound/list_tasks_of_missing_project`: GET /api/projects/{id}/tasks for a missing project must return 404 (taskboard/internal/web/api.go, listTasks).
- Exact-option assertions for filter-status and filter-priority (taskboard/internal/web/ui_content_test.go).

These files belong to completed phases (phase-1-app and phase-2-kanban). Their owning beads are ta-oe1 and ta-s7n. The planner does not create beads for them. The QA rewind routes the fix to the owning completed phase. ta-ev8 stays open until phase verify is green.

## Future considerations
- The `testDir` mismatch in the cached config (`./e2e`) is resolved by the ta-ev8 acceptance above.
