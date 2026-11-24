# Implementation plan

Active phase: `phase-3-browser`. Required files: 4 (taskboard/package.json, taskboard/package-lock.json, taskboard/playwright.config.ts, taskboard/playwright/taskboard.spec.ts). Later-phase files are out of scope for this plan.

## Phase blocker (from QA failure)

Phase verify (`cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`) is red in `taskboard/internal/web`. These failures belong to completed-phase Task API work, not to the four active-phase files. They are repaired through the QA rewind to the owning phase (beads ta-oe1 and ta-s7n), not by these beads:
- TestTaskCreateDefaultsAndIgnoresClientFields: POST /api/projects/{id}/tasks must ignore client `status` and default to `todo`.
- TestTaskPatchRejectsEmptyAndNullFields: PATCH /api/tasks/{id} with an empty object, or with null title, description, status, or priority, must return 400.
- TestTaskMissingEntitiesReturnNotFound/list_tasks_of_missing_project: GET /api/projects/{missing}/tasks must return 404.

The ta-ev8 polecat must not close its bead while these remain red.

## Bead map

### ta-o19: taskboard/package.json
- Scope: JSON manifest for the Playwright E2E package under taskboard/.
- Architecture: taskboard/package.json in Planned file layout.
- Interfaces: scripts.test runs the Playwright chromium project; devDependency @playwright/test pinned to 1.62.1.
- Depends on: <none>
- Consumed by: taskboard/package-lock.json, taskboard/playwright.config.ts
- Acceptance:
  - Valid JSON with `name`, `scripts.test`, and `devDependencies["@playwright/test"]` present.
  - Version pin matches the lockfile entry in ta-qu7.
- Verify: `cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`

### ta-qu7: taskboard/package-lock.json
- Scope: lockfile resolving the dependencies declared in taskboard/package.json.
- Architecture: taskboard/package-lock.json in Planned file layout.
- Interfaces: none (generated lockfile).
- Depends on: taskboard/package.json
- Consumed by: dependency installation inside taskboard/.
- Acceptance:
  - Lockfile resolves @playwright/test 1.62.1 and its transitive dependencies.
  - Lockfile is consistent with taskboard/package.json (no drift on reinstall).
- Verify: `cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`

### ta-ev8: taskboard/playwright.config.ts
- Scope: Playwright runner configuration for browser E2E tests.
- Architecture: taskboard/playwright.config.ts in Planned file layout (Playwright test runner configuration).
- Interfaces: default export from `defineConfig`; baseURL from BASE_URL env, default http://localhost:8080; one `chromium` project using the Desktop Chrome device profile.
- Depends on: taskboard/package.json, taskboard/package-lock.json
- Consumed by: taskboard/playwright/taskboard.spec.ts
- Acceptance:
  - `testDir` points at the directory containing taskboard.spec.ts (`./playwright`). The cached content sets `./e2e`, which would discover zero tests for the `playwright` verify argument; fix before close.
  - `baseURL` honors BASE_URL and defaults to http://localhost:8080.
  - The chromium project uses the Desktop Chrome device profile.
  - Config loads without error when Playwright lists tests with the config file.
- Verify: `cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`

### ta-m0z: taskboard/playwright/taskboard.spec.ts
- Scope: browser E2E scenarios for the Kanban UI and REST-backed behavior.
- Architecture: taskboard/playwright/taskboard.spec.ts (Playwright E2E browser tests).
- Interfaces: test file only; it must not import Go code.
- Depends on: taskboard/playwright.config.ts, taskboard/package.json
- Consumed by: phase verify command.
- Acceptance:
  - Creating a project through the UI makes it appear in the project list.
  - Creating a task in a project shows it in the todo column with status `todo`, regardless of any client-set status.
  - Moving a task to another status via the UI persists after reload.
  - Filtering by status and priority shows only matching tasks.
  - Deleting a task removes it from the board.
  - No trivial always-pass assertions (`expect(true)`, empty tests).
  - Each test drives at least one real control (button, input, form) and asserts the resulting DOM state.
- Verify: `cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`

## Future considerations
- None for this phase.

## Phase context

Active delivery phase `phase-3-browser` only. Later phases start automatically after QA `all_passed`.
