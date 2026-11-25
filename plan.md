# Implementation plan

Active phase: phase-3-browser (Playwright integration and browser QA).
Required files: taskboard/package.json, taskboard/package-lock.json, taskboard/playwright.config.ts, taskboard/playwright/taskboard.spec.ts.
Phase verify: cd taskboard && go test ./... && ./node_modules/.bin/playwright test --config playwright.config.ts playwright
Precondition: the TaskBoard server is already running on 127.0.0.1:8080 (or BASE_URL). Node dependencies are provisioned from taskboard/package-lock.json by an operator or environment step before test review; the polecat and planner do not install them.

## Bead map

### ta-o19: taskboard/package.json
- Scope: E2E devDependency and test script for the Playwright suite under taskboard/. No runtime dependencies, no frontend build step.
- Architecture: declares @playwright/test pinned to exactly 1.62.1 in devDependencies; the test script runs Playwright on the chromium project.
- Interfaces: none (manifest only).
- Depends on: none.
- Consumed by: taskboard/package-lock.json (ta-qu7), taskboard/playwright.config.ts (ta-ff9), taskboard/playwright/taskboard.spec.ts (ta-l7z).
- Status: closed; content already validated. Do not recreate.
- Acceptance:
  - Parses as JSON and lists @playwright/test at exactly 1.62.1 under devDependencies.
  - A test script exists and runs Playwright on the chromium project.
  - No dependencies beyond @playwright/test.
- Verify: cd taskboard && node -e "JSON.parse(require('fs').readFileSync('package.json','utf8'))"

### ta-qu7: taskboard/package-lock.json
- Scope: lockfile matching taskboard/package.json so provisioning reproduces the Playwright install exactly.
- Architecture: pins the same @playwright/test 1.62.1 declared in package.json; lockfileVersion 3.
- Interfaces: none (lockfile only).
- Depends on: taskboard/package.json (ta-o19).
- Consumed by: the Playwright verify step, which needs taskboard/node_modules/.bin/playwright provisioned from this lockfile.
- Status: closed. Do not recreate.
- Acceptance:
  - lockfileVersion is 3; root package name and devDependencies match package.json.
  - Provisioning under taskboard/ yields node_modules/.bin/playwright without modifying the lockfile.
  - No packages are present that package.json does not declare.
- Verify: cd taskboard && ./node_modules/.bin/playwright --version

### ta-ff9: taskboard/playwright.config.ts
- Scope: Playwright configuration. testDir ./playwright, per-test timeout 10000 ms, one chromium project using Desktop Chrome.
- Architecture: use.baseURL defaults to http://127.0.0.1:8080 and is overridable via BASE_URL. Chromium launch args disable HTTPS-upgrade features and pass --ignore-certificate-errors for local runs.
- Interfaces: default export from defineConfig with testDir, timeout, use.baseURL, and projects (exactly one project named chromium).
- Depends on: taskboard/package.json (ta-o19) for @playwright/test; taskboard/package-lock.json (ta-qu7) for the installed binary.
- Consumed by: taskboard/playwright/taskboard.spec.ts (ta-l7z) through the Playwright runner.
- Status: open (in progress). Previous attempt was rejected because the content was valid but the binary was not provisioned; close only after the verify command passes.
- Acceptance:
  - BASE_URL overrides use.baseURL; without it the base URL is http://127.0.0.1:8080.
  - forbidOnly is true when CI is set; workers is 1 when CI is set.
  - Reporters are html (open: never) and list.
  - Exactly one project, named chromium.
- Verify: cd taskboard && ./node_modules/.bin/playwright test --config playwright.config.ts --list

### ta-l7z: taskboard/playwright/taskboard.spec.ts
- Scope: browser E2E scenarios for the TaskBoard UI (SPEC §8 Playwright section), run against the live app on the configured base URL. Replaces the current smoke-only content.
- Architecture: flows are load the home page; create project and task through browser controls; move a task To Do to Doing to Done with Move forward; delete a task; create two tasks with different priorities and filter by high; edit a task; delete a project after confirm.
- Interfaces: Playwright test blocks only; no exported symbols.
- Depends on: taskboard/playwright.config.ts (ta-ff9), taskboard/package.json (ta-o19).
- Consumed by: the test script in taskboard/package.json and the Playwright verify step.
- Status: open.
- Acceptance:
  - Initial page shows the TaskBoard heading and the three columns To Do, Doing, Done.
  - Create a project and a task using the real form controls; the task title appears in the To Do column.
  - Click Move forward: the card moves to Doing, then Done; on Done the Move forward control is hidden or disabled. Each step asserts the card's column in the DOM.
  - Delete the task through its Delete control; the title is no longer in the DOM.
  - Create two tasks with priorities high and low; select priority filter high; only the high task title is visible; reset to all; both are visible.
  - Status filter: with tasks in todo and doing, selecting doing shows only the doing task title.
  - Edit a task title through Edit controls; the new title is visible after save.
  - Delete a project with window.confirm accepted; it disappears from the project selector.
  - Every test uses a unique project name (timestamp suffix) and removes the projects it created in afterEach, so tests do not depend on execution order or earlier runs.
  - Passes with ./node_modules/.bin/playwright test --config playwright.config.ts playwright against a running app.
- Verify: cd taskboard && ./node_modules/.bin/playwright test --config playwright.config.ts playwright

## Dependency order
1. ta-o19 (taskboard/package.json), closed
2. ta-qu7 (taskboard/package-lock.json), closed; provisioning from the lockfile
3. ta-ff9 (taskboard/playwright.config.ts), open
4. ta-l7z (taskboard/playwright/taskboard.spec.ts), open

## Blocker to resolve before verify
- ta-ff9 and ta-l7z cannot be verified until taskboard/node_modules/.bin/playwright exists. That binary must be provisioned from the locked lockfile by an environment or operator step, which is outside polecat and planner scope.

## Future considerations
- Backend Go unit and integration tests belong to earlier phases and are not part of this plan.
