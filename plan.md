# Implementation plan

Active phase: phase-3-browser (Playwright integration and browser QA).
Scope: the four `required_files` for this phase only. Backend and frontend paths belong to earlier or later phases and have no beads here.

## Bead map

### ta-o19: taskboard/package.json
- Scope: Node manifest for the Playwright E2E harness under `taskboard/`. Declares a `test` script that runs Playwright on the chromium project and the `@playwright/test` devDependency pinned to 1.62.1.
- Architecture: architecture.md lists `taskboard/playwright/taskboard.spec.ts` as the browser E2E test file; the harness is driven from `taskboard/`.
- Interfaces: none (manifest only).
- Depends on: <none>
- Consumed by: taskboard/package-lock.json (ta-qu7), taskboard/playwright.config.ts (ta-ff9), taskboard/playwright/taskboard.spec.ts (ta-l7z)
- Status: closed; content already validated. Do not recreate.
- Acceptance:
  - Parses as JSON and lists `@playwright/test` at exactly 1.62.1 under devDependencies.
  - A `test` script exists and runs Playwright on the chromium project.
  - No runtime dependencies beyond what architecture requires.
- Verify: `cd taskboard && ./node_modules/.bin/playwright test --config playwright.config.ts --list`

### ta-qu7: taskboard/package-lock.json
- Scope: lockfile matching `taskboard/package.json`, so the install reproduces the Playwright install exactly.
- Architecture: pins the same `@playwright/test` 1.62.1 declared in package.json.
- Interfaces: none (lockfile only).
- Depends on: taskboard/package.json (ta-o19)
- Consumed by: the Playwright verify step, which needs `taskboard/node_modules/.bin/playwright` installed from this lockfile.
- Status: in progress.
- Acceptance:
  - `lockfileVersion` is 3; root package name and devDependencies match package.json.
  - Installing under `taskboard/` produces `node_modules/.bin/playwright` without modifying the lockfile.
  - No packages are added that package.json does not declare.
- Verify: `cd taskboard && ./node_modules/.bin/playwright test --config playwright.config.ts --list`

### ta-ff9: taskboard/playwright.config.ts
- Scope: Playwright configuration. Test directory `./playwright`, per-test timeout 10000 ms, single chromium project using Desktop Chrome.
- Architecture: default `baseURL` is `http://127.0.0.1:8080`, overridable via `BASE_URL`. The chromium launch args disable the HTTPS-upgrade features and pass `--ignore-certificate-errors` for local runs.
- Interfaces: default export from `defineConfig` with `testDir`, `timeout`, `use.baseURL`, and `projects` (one project named `chromium`).
- Depends on: taskboard/package.json (ta-o19) for `@playwright/test`
- Consumed by: taskboard/playwright/taskboard.spec.ts (ta-l7z) through the Playwright runner.
- Status: open.
- Acceptance:
  - `BASE_URL` overrides `use.baseURL`; without it the base URL is `http://127.0.0.1:8080`.
  - `forbidOnly` is set when `CI` is present; `workers` is 1 under CI.
  - Reporters are `html` (open: never) and `list`.
  - Exactly one project, named `chromium`.
- Verify: `cd taskboard && ./node_modules/.bin/playwright test --config playwright.config.ts --list`

### ta-l7z: taskboard/playwright/taskboard.spec.ts
- Scope: browser E2E scenarios for the TaskBoard UI, run against the live app on the configured base URL.
- Architecture: covers the SPEC flows: load the home UI at the root path; create and delete projects; create and edit tasks; move a task through To Do, Doing, and Done with the Move forward control; filter by status (all, todo, doing, done) and by priority (all, low, medium, high); delete a task.
- Interfaces: Playwright `test` blocks only; no exported symbols.
- Depends on: taskboard/playwright.config.ts (ta-ff9), taskboard/package.json (ta-o19)
- Consumed by: the `test` script in taskboard/package.json and the Playwright verify step.
- Status: open.
- Acceptance:
  - One test per SPEC flow listed above; each test asserts the visible UI state after the action, not only that a request was sent.
  - The status filter and the priority filter each have a test that checks which task titles remain visible.
  - Tests create their own projects and tasks and clean them up, so they do not depend on execution order.
  - The spec passes with `./node_modules/.bin/playwright test --config playwright.config.ts playwright` against a running app.
- Verify: `cd taskboard && ./node_modules/.bin/playwright test --config playwright.config.ts playwright`

## Dependency order
1. ta-o19 (taskboard/package.json), closed
2. ta-qu7 (taskboard/package-lock.json), in progress
3. ta-ff9 (taskboard/playwright.config.ts), open
4. ta-l7z (taskboard/playwright/taskboard.spec.ts), open

## Future considerations
- Go unit and integration tests for the backend belong to earlier phases and are not part of this plan.
