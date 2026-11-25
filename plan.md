# Implementation plan

Active phase: phase-3-browser (Playwright integration and browser QA).
Required files: taskboard/package.json, taskboard/package-lock.json, taskboard/playwright.config.ts, taskboard/playwright/taskboard.spec.ts.
Phase verify: cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright

## Bead map

### ta-o19: taskboard/package.json
- Scope: E2E test dependencies and scripts for the Playwright suite under taskboard/.
- Architecture: declares @playwright/test pinned to 1.62.1 in devDependencies; the test script runs Playwright on the chromium project.
- Interfaces: none (manifest only).
- Depends on: none.
- Consumed by: taskboard/package-lock.json (ta-qu7), taskboard/playwright.config.ts (ta-ff9), taskboard/playwright/taskboard.spec.ts (ta-l7z).
- Status: closed; content already validated. Do not recreate.
- Acceptance:
  - Parses as JSON and lists @playwright/test at exactly 1.62.1 under devDependencies.
  - A test script exists and runs Playwright on the chromium project.
  - No runtime dependencies beyond what architecture requires.
- Verify: cd taskboard && ./node_modules/.bin/playwright test --config playwright.config.ts --list

### ta-qu7: taskboard/package-lock.json
- Scope: lockfile matching taskboard/package.json so dependency provisioning reproduces the Playwright install exactly.
- Architecture: pins the same @playwright/test 1.62.1 declared in package.json.
- Interfaces: none (lockfile only).
- Depends on: taskboard/package.json (ta-o19).
- Consumed by: the Playwright verify step, which needs taskboard/node_modules/.bin/playwright provisioned from this lockfile.
- Status: closed. Do not recreate.
- Acceptance:
  - lockfileVersion is 3; root package name and devDependencies match package.json.
  - Provisioning under taskboard/ yields node_modules/.bin/playwright without modifying the lockfile.
  - No packages are added that package.json does not declare.
- Verify: cd taskboard && ./node_modules/.bin/playwright --version
- Environment note: node_modules is provisioned from this lockfile by an environment or operator step before the Playwright verify step. The polecat and planner do not provision dependencies.

### ta-ff9: taskboard/playwright.config.ts
- Scope: Playwright configuration. Test directory ./playwright, per-test timeout 10000 ms, single chromium project using Desktop Chrome.
- Architecture: default baseURL is http://127.0.0.1:8080, overridable via BASE_URL. The chromium launch args disable the HTTPS-upgrade features and pass --ignore-certificate-errors for local runs.
- Interfaces: default export from defineConfig with testDir, timeout, use.baseURL, and projects (one project named chromium).
- Depends on: taskboard/package.json (ta-o19) for @playwright/test; taskboard/package-lock.json (ta-qu7) for the installed binary.
- Consumed by: taskboard/playwright/taskboard.spec.ts (ta-l7z) through the Playwright runner.
- Status: open (in progress).
- Acceptance:
  - BASE_URL overrides use.baseURL; without it the base URL is http://127.0.0.1:8080.
  - forbidOnly is set when CI is present; workers is 1 under CI.
  - Reporters are html (open: never) and list.
  - Exactly one project, named chromium.
- Verify: cd taskboard && ./node_modules/.bin/playwright test --config playwright.config.ts --list

### ta-l7z: taskboard/playwright/taskboard.spec.ts
- Scope: browser E2E scenarios for the TaskBoard UI, run against the live app on the configured base URL.
- Architecture: covers the SPEC flows: load the home UI at the root path; create and delete projects; create and edit tasks; move a task through To Do, Doing, and Done with the Move forward control; filter by status (all, todo, doing, done) and by priority (all, low, medium, high); delete a task.
- Interfaces: Playwright test blocks only; no exported symbols.
- Depends on: taskboard/playwright.config.ts (ta-ff9), taskboard/package.json (ta-o19).
- Consumed by: the test script in taskboard/package.json and the Playwright verify step.
- Status: open.
- Acceptance:
  - One test per SPEC flow listed above; each test asserts the visible UI state after the action, not only that a request was sent.
  - The status filter and the priority filter each have a test that checks which task titles remain visible.
  - Tests create their own projects and tasks and clean them up, so they do not depend on execution order.
  - The spec passes with ./node_modules/.bin/playwright test --config playwright.config.ts playwright against a running app.
- Verify: cd taskboard && ./node_modules/.bin/playwright test --config playwright.config.ts playwright

## Dependency order
1. ta-o19 (taskboard/package.json), closed
2. ta-qu7 (taskboard/package-lock.json), closed; dependency provisioning from the lockfile
3. ta-ff9 (taskboard/playwright.config.ts), open
4. ta-l7z (taskboard/playwright/taskboard.spec.ts), open

## Blocker to resolve before verify
- ta-ff9 and ta-l7z cannot be verified until taskboard/node_modules/.bin/playwright exists. That binary must be provisioned from the locked lockfile by an environment or operator step, which is outside polecat and planner scope.

## Future considerations
- Go unit and integration tests for the backend belong to earlier phases and are not part of this plan.
